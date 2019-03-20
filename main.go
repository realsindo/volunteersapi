package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-config"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	//on windows I use postgress backend because sqllite did not work and I was in hurry
	//_ "github.com/jinzhu/gorm/dialects/postgres"
)

//Used contants
const dbFile = "volunteersdb"
const confFile = "config.json"
const authSelect = "AuthSelect"
const realmPrefix = "RealmPrefix"
const authorizationRequired = "AuthorizationRequired"
const wwwAuthentificate = "WwwAuthentificate"
const emailConfig = "EmailConfig"
const smtpServer = "SmtpServer"
const emailSubject = "Subject"
const emailMessage = "Message"
const emailSender = "Sender"

// Global variables
var db *gorm.DB
var err error
var conf map[string]interface{}
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

//Fill authorization map with users from database
func fillAuthMap() {
	rows, err := db.Raw(conf[authSelect].(string)).Rows()
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
	//Load config file and map it to the global variable
	config.LoadFile(confFile)
	conf = config.Map()
	//make global variablr
	authMap = make(gin.Accounts)

	//Connect to the database
	db, err = gorm.Open("sqlite3", dbFile)
	//I am using postgress on Windows because sqllite backen did not work for me and I was in hurry
	//db, err = gorm.Open("postgres", "host=localhost port=5432 user=postgres sslmode=disable dbname=test password=heslo")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	//only for sqllite to allow foreign keys
	db.Exec("PRAGMA foreign_keys = ON")

	//Setup database
	db.AutoMigrate(&Volunteer{}, &Team{}, &VolunteerEmail{})

	//Add foreign keys for postgres database (does not work for sqllite)
	db.Debug().Model(&VolunteerEmail{}).AddForeignKey("team_id", "teams(id)", "CASCADE", "CASCADE")
	db.Debug().Model(&VolunteerEmail{}).AddForeignKey("volunteer_email", "volunteers(email)", "CASCADE", "CASCADE")

	//fill authorization map from database
	fillAuthMap()

	// add reporter user
	authMap[conf[reporterUser].(string)] = conf[reporterPassword].(string)

	//setup routing for volunteer
	r := gin.Default()
	v1 := r.Group("/v1")
	a1 := r.Group("/v1")
	//add basic authentification
	a1.Use(authRequired())

	//routing will be loaded from config file.
	a1.GET("/volunteers/", getVolunteers)
	a1.GET("/volunteers/:email", getVolunteer)
	v1.POST("/volunteers", createVolunteer)
	a1.PUT("/volunteers/:email", updateVolunteer)
	a1.DELETE("/volunteers/:email", deleteVolunteer)

	//setup routing for teams
	a1.POST("/teams", createTeam)
	a1.GET("/teams/", getTeams)
	a1.GET("/teams/:identifier", getTeam)
	a1.PUT("/teams/:identifier/sign", assignVolunteerToTeam)
	a1.PUT("/teams/:identifier/deassign", deassignVolunteerFromTeam)
	a1.DELETE("/teams/:identifier", deleteTeam)

	r.Run(":8080")
}

//check basic authentification if needed
func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, _ := c.Request.BasicAuth()
		if authMap[username] == "" || authMap[username] != password {
			//realm := "Basic realm=" + strconv.Quote("Authorization Required")
			realm := conf[realmPrefix].(string) + strconv.Quote(conf[authorizationRequired].(string))
			//c.Header("WWW-Authenticate", realm)
			c.Header(conf[wwwAuthentificate].(string), realm)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

//send email with password
func (vol *Volunteer) sendEmailWithPassword() {
	emailcfg := conf[emailConfig].(map[string]interface{})
	c, err := smtp.Dial(emailcfg[smtpServer].(string))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	c.Mail(emailcfg[emailSender].(string))
	c.Rcpt(vol.Email)

	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString(emailcfg[emailSubject].(string) + vol.Password)
	if _, err = buf.WriteTo(wc); err != nil {
		log.Fatal(err)
	}
}
