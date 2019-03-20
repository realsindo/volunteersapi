package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sethvargo/go-password/password"
)

//Used constatnts
const errorNotFound = "ErrorNotFound"
const errorUnauthorized = "ErrorUnauthorized"
const jsonDecoding = "JsonDecoding"
const entityExists = "EntityExists"
const errorInternal = "InternalError"
const reporterUser = "ReporterUser"
const reporterPassword = "ReporterPassword"

//authentificate user if data contain user data
func volunteerAuth(c *gin.Context, vol *Volunteer) bool {
	email, _, _ := c.Request.BasicAuth()
	if email != vol.Email {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":  conf[errorUnauthorized].(string),
			"status": http.StatusUnauthorized,
		})
		return false
	}
	return true
}

//this is preparation for reporter user which will be able to get all data
func reporterAuth(c *gin.Context) bool {
	repuser, passwd, _ := c.Request.BasicAuth()

	if repuser != conf[reporterUser].(string) || passwd != conf[reporterPassword].(string) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":  conf[errorUnauthorized].(string),
			"status": http.StatusUnauthorized,
		})
		return false
	}
	return true
}

//deleteVolunteer for authentificated user and his data
func deleteVolunteer(c *gin.Context) {
	email := c.Params.ByName("email")
	var vol Volunteer
	if err := db.Where("email = ?", email).Find(&vol).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	//User check
	if !volunteerAuth(c, &vol) {
		return
	}
	//db delete
	db.Delete(&vol)

	//delete from authentification map
	delete(authMap, vol.Email)
	c.JSON(200, gin.H{"Message": email + " deleted"})
}

//updateVolunteer for authentificated user and his data
func updateVolunteer(c *gin.Context) {

	var vol Volunteer
	email := c.Params.ByName("email")

	//check json data
	if err = c.BindJSON(&vol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  conf[jsonDecoding].(string) + err.Error(),
			"status": http.StatusBadRequest,
		})
		return
	}

	//get volunteer from database
	var oldvol Volunteer
	if err := db.Where("email = ?", email).First(&oldvol).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	//sets data which could not be changed (I am still not sure if path have to be with :email )
	vol.ID = oldvol.ID
	vol.Email = oldvol.Email
	if vol.Password == "" {
		vol.Password = oldvol.Password
	}
	if !volunteerAuth(c, &vol) {
		return
	}

	//Save to db
	if err := db.Save(&vol).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  conf[entityExists].(string),
			"status": http.StatusConflict,
		})
		return
	}
	//change password in auth map
	authMap[vol.Email] = vol.Password
	c.JSON(200, vol)

}

//creates new Volunteer
func createVolunteer(c *gin.Context) {

	var vol Volunteer
	//check json data
	if err = c.BindJSON(&vol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  conf[jsonDecoding].(string) + err.Error(),
			"status": http.StatusBadRequest,
		})
		return
	}

	//generate password
	res, _ := password.Generate(16, 5, 3, false, true)
	vol.Password = res
	//writes to database
	if err := db.Create(&vol).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  conf[entityExists].(string),
			"status": http.StatusConflict,
		})
		return
	}
	//add to auth map
	authMap[vol.Email] = res

	//send email with password
	vol.sendEmailWithPassword()
	c.JSON(200, gin.H{"Message": vol.Email + " created"})
}

//getVolunteer for authentificated userand his data
//Question if is ok to show password I think no ;)
func getVolunteer(c *gin.Context) {
	email := c.Params.ByName("email")
	var vol Volunteer
	//read volunteer from database
	if err := db.Where("email = ?", email).First(&vol).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	//check user
	if !volunteerAuth(c, &vol) {
		return
	}
	//set volunteer json
	c.JSON(200, vol)
}

//getVolunteers only for reporter
func getVolunteers(c *gin.Context) {
	var vols []Volunteer
	//read volunteers from database
	if err := db.Find(&vols).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}

	if !reporterAuth(c) {
		return
	}
	c.JSON(200, vols)
}
