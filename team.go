package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//Used constant
const selectVolunteerEmail = "VolunteerEmailSelect"

//createTeam for authentificated user
func createTeam(c *gin.Context) {
	var tm Team
	//json validation
	if err = c.BindJSON(&tm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  conf[jsonDecoding].(string) + err.Error(),
			"status": http.StatusBadRequest,
		})
		return
	}

	//db creation
	if err := db.Create(&tm).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  conf[entityExists].(string),
			"status": http.StatusConflict,
		})
		return
	}
	c.JSON(200, tm)
}

//getTeams only for reporter
func getTeams(c *gin.Context) {
	var tms []Team
	//read from db
	if err := db.Preload("VolunteerEmails").Find(&tms).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	//repoter check
	if !reporterAuth(c) {
		return
	}
	c.JSON(200, tms)
}

//getTeam only for authentificated user
func getTeam(c *gin.Context) {
	id := c.Params.ByName("identifier")
	var tm Team
	if err := db.Where("identifier = ?", id).Preload("VolunteerEmails").First(&tm).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	c.JSON(200, tm)

}

func deleteTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")
	var tm Team
	if err := db.Where("identifier = ?", identifier).Find(&tm).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}
	db.Delete(&tm)
	c.JSON(200, gin.H{"Message": identifier + " deleted"})
}

//assign Volunteer to team
func assignVolunteerToTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")

	var ve VolunteerEmail
	//validate json
	if err = c.BindJSON(&ve); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  conf[jsonDecoding].(string) + err.Error(),
			"status": http.StatusBadRequest,
		})
		return
	}
	email := ve.VolunteerEmail

	//get VolunteerEmail from database
	err := db.Raw(conf[selectVolunteerEmail].(string), ve.VolunteerEmail, identifier).Find(&ve).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  conf[entityExists].(string),
			"status": http.StatusConflict,
		})
		return
	}

	//get Team from database
	var tm Team
	if err := db.Where("identifier = ?", identifier).First(&tm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  conf[errorInternal].(string),
			"status": http.StatusInternalServerError,
		})
		return
	}

	//write to database
	ve = VolunteerEmail{tm.ID, email}
	if err := db.Create(&ve).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  conf[errorInternal].(string),
			"status": http.StatusInternalServerError,
		})
		return
	}
	c.JSON(200, ve)

}

//deassign Volunteer to team
func deassignVolunteerFromTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")

	var ve VolunteerEmail
	//validate json
	if err = c.BindJSON(&ve); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  conf[jsonDecoding].(string) + err.Error(),
			"status": http.StatusBadRequest,
		})
		return
	}
	email := ve.VolunteerEmail

	//get VolunteerEmail from database
	err := db.Raw(conf[selectVolunteerEmail].(string), ve.VolunteerEmail, identifier).Find(&ve).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  conf[errorNotFound].(string),
			"status": http.StatusNotFound,
		})
		return
	}

	//get Team from database
	var tm Team
	if err := db.Where("identifier = ?", identifier).First(&tm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  conf[errorInternal].(string),
			"status": http.StatusInternalServerError,
		})
		return
	}

	//delete from database
	ve = VolunteerEmail{tm.ID, email}
	if err := db.Delete(VolunteerEmail{}, "volunteer_email=? and team_id=?", ve.VolunteerEmail, ve.TeamID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  conf[errorInternal].(string),
			"status": http.StatusInternalServerError,
		})
		return
	}
	c.JSON(200, ve)

}
