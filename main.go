package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-config"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

//Used constants
const cListener = "Listener"
const cDatabase = "Database"
const cAuthSelect = "AuthSelect"
const cRealmPrefix = "RealmPrefix"
const cAuthorizationRequired = "AuthorizationRequired"
const cWwwAuthentificate = "WwwAuthentificate"
const cEmailConfig = "EmailConfig"
const cSMTPServer = "SmtpServer"
const cEmailSubject = "Subject"
const cEmailMessage = "Message"
const cEmailSender = "Sender"
const cErrorNotFound = "ErrorNotFound"
const cErrorUnauthorized = "ErrorUnauthorized"
const cJSONDecoding = "JsonDecoding"
const cEntityExists = "EntityExists"
const cErrorInternal = "InternalError"
const cReporterUser = "ReporterUser"
const cReporterPassword = "ReporterPassword"
const cUsageLine1 = "Config file parameter missing"
const cUsageLine2 = "Usage: %s config_file\n"
const cUsageLine3 = "config_file - path to config file\n"

// Global variables
var db *gorm.DB
var authMap gin.Accounts

//Volunteer struct is representation of Volunteer
type Volunteer struct {
	ID        uint   `json:"id" gorm:"primary_key"`
	Email     string `json:"email" binding:"required,email,max=255" gorm:"not null;unique_index"`
	FirstName string `json:"firstname" binding:"max=255"`
	LastName  string `json:"lastname" binding:"required,max=255" gorm:"not null"`
	Password  string `json:"password" binding:"max=255" gorm:"not null"`
}

//Team struct is representation of Team
type Team struct {
	ID              uint             `json:"id" gorm:"primary_key"`
	Identifier      string           `json:"identifier" binding:"required,max=255" gorm:"not null;unique_index"`
	Name            string           `json:"name" binding:"max=255"`
	VolunteerEmails []VolunteerEmail `json:"volunteeremails"`
}

//VolunteerEmail struct is representation of volunter assigned to the team
type VolunteerEmail struct {
	TeamID         uint
	VolunteerEmail string `json:"volunteeremail" binding:"required,email,max=255" ` //
}

//Fills authorization map with users from database
func fillAuthMap() {
	rows, err := db.Raw(getCfgString(cAuthSelect)).Rows()
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	var email string
	var passwd string
	for rows.Next() {
		rows.Scan(&email, &passwd)
		authMap[email] = passwd
	}
}

//Main function
func main() {
	//Checks if has argument for config
	if len(os.Args) < 2 {
		fmt.Println(cUsageLine1)
		progname, _ := os.Executable()
		fmt.Printf(cUsageLine2, path.Base(progname))
		fmt.Println(cUsageLine3)
		os.Exit(0)
	}
	//Loads config file
	config.LoadFile(os.Args[1])

	//Makes global variable
	authMap = make(gin.Accounts)

	//Connects to the database
	var err error
	db, err = gorm.Open("sqlite3", getCfgString(cDatabase))
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	//Only for sqllite to allow foreign keys
	db.Exec("PRAGMA foreign_keys = ON")

	//Setup database
	db.AutoMigrate(&Volunteer{}, &Team{}, &VolunteerEmail{})

	//Add foreign keys for postgres database (does not work for sqllite)
	db.Debug().Model(&VolunteerEmail{}).AddForeignKey("team_id", "teams(id)", "CASCADE", "CASCADE")
	db.Debug().Model(&VolunteerEmail{}).AddForeignKey("volunteer_email", "volunteers(email)", "CASCADE", "CASCADE")

	//Fills authorization map from database
	fillAuthMap()

	//Adds reporter user
	authMap[getCfgString(cReporterUser)] = getCfgString(cReporterPassword)

	//Setup routing for volunteer
	r := gin.Default()
	v1 := r.Group("/v1")
	a1 := r.Group("/v1")
	//Adds basic authentification
	a1.Use(authRequired())

	a1.GET("/volunteers/", getVolunteers)
	a1.GET("/volunteers/:email", getVolunteer)
	v1.POST("/volunteers", createVolunteer)
	a1.PUT("/volunteers/:email", updateVolunteer)
	a1.DELETE("/volunteers/:email", deleteVolunteer)

	//Setup routing for teams
	a1.POST("/teams", createTeam)
	a1.GET("/teams/", getTeams)
	a1.GET("/teams/:identifier", getTeam)
	a1.PUT("/teams/:identifier/sign", assignVolunteerToTeam)
	a1.PUT("/teams/:identifier/deassign", deassignVolunteerFromTeam)
	a1.DELETE("/teams/:identifier", deleteTeam)

	r.Run(getCfgString(cListener))
}

//Check basic authentification if needed
func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, _ := c.Request.BasicAuth()
		if authMap[username] == "" || authMap[username] != password {
			realm := getCfgString(cRealmPrefix) + strconv.Quote(getCfgString(cAuthorizationRequired))
			c.Header(getCfgString(cWwwAuthentificate), realm)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

//Authentificates user if data belong to the user
func volunteerAuth(c *gin.Context, vol *Volunteer) bool {
	email, _, _ := c.Request.BasicAuth()
	if email != vol.Email {
		return false
	}
	return true
}

//This is preparation for reporter user which will be able to get all data
func reporterAuth(c *gin.Context) bool {
	repuser, passwd, _ := c.Request.BasicAuth()

	if repuser != getCfgString(cReporterUser) || passwd != getCfgString(cReporterPassword) {
		createUnauthorizedResponse(c)
		return false
	}
	return true
}

//Gets string value from config
func getCfgString(name ...string) string {
	return getCfgStringDefault("", name...)
}

//Gets string value from config with default value
func getCfgStringDefault(def string, name ...string) string {
	return config.Get(name...).String(def)
}

//Sends email with password
func (vol *Volunteer) sendEmailWithPassword() {
	c, err := smtp.Dial(getCfgString(cEmailConfig, cSMTPServer))
	if err != nil {
		log.Panicln(err)
	}
	defer c.Close()

	from := getCfgString(cEmailConfig, cEmailSender)
	c.Mail(from)
	c.Rcpt(vol.Email)

	wc, err := c.Data()
	if err != nil {
		log.Panicln(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString("From: " + from + "\r\n" + "To: " + vol.Email + "\r\n" +
		"Subject: " + getCfgString(cEmailConfig, cEmailSubject) + "\r\n" + "\r\n" +
		getCfgString(cEmailConfig, cEmailMessage) + vol.Password)
	if _, err = buf.WriteTo(wc); err != nil {
		log.Panicln(err)
	}
}

//Creates Unauthorized response
func createUnauthorizedResponse(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error":  getCfgString(cErrorUnauthorized),
		"status": http.StatusUnauthorized,
	})
}

//Creates NotFound response
func createNotFoundResponse(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":  getCfgString(cErrorNotFound),
		"status": http.StatusNotFound,
	})
}

//Creates BadRequest response
func createBadRequestResponse(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":  getCfgString(cJSONDecoding) + err.Error(),
		"status": http.StatusBadRequest,
	})
}

//Creates StatusConflict response
func createStatusConflictResponse(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{
		"error":  getCfgString(cEntityExists),
		"status": http.StatusConflict,
	})
}

//Creates InternalServerError response
func createInternalErrorResponse(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":  getCfgString(cErrorInternal),
		"status": http.StatusInternalServerError,
	})
}
