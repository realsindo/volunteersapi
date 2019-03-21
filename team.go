package main

import (
	"github.com/gin-gonic/gin"
)

//Used constants
const cSelectVolunteerEmail = "VolunteerEmailSelect"

//createTeam for authentificated user
func createTeam(c *gin.Context) {
	var tm Team
	//Validates json
	if err := c.BindJSON(&tm); err != nil {
		createBadRequestResponse(c, err)
		return
	}

	//Creates Team in database
	if err := db.Create(&tm).Error; err != nil {
		createStatusConflictResponse(c)
		return
	}
	c.JSON(200, tm)
}

//getTeams only for reporter
func getTeams(c *gin.Context) {
	var tms []Team
	//Reads from database
	if err := db.Preload("VolunteerEmails").Find(&tms).Error; err != nil {
		createNotFoundResponse(c)
		return
	}
	//Checks if user is reporter
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
		createNotFoundResponse(c)
		return
	}
	c.JSON(200, tm)

}

//deleteTeam only for authentificated user
func deleteTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")
	var tm Team
	if err := db.Where("identifier = ?", identifier).Find(&tm).Error; err != nil {
		createNotFoundResponse(c)
		return
	}
	db.Delete(&tm)
	c.JSON(200, gin.H{"Message": identifier + " deleted"})
}

//Assigns Volunteer to the Team
func assignVolunteerToTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")

	var ve VolunteerEmail
	//Validates json
	if err := c.BindJSON(&ve); err != nil {
		createBadRequestResponse(c, err)
		return
	}
	email := ve.VolunteerEmail

	//Gets VolunteerEmail from database
	err := db.Raw(getCfgString(cSelectVolunteerEmail), ve.VolunteerEmail, identifier).Find(&ve).Error
	if err == nil {
		createStatusConflictResponse(c)
		return
	}

	//Gets Team from database
	var tm Team
	if err := db.Where("identifier = ?", identifier).First(&tm).Error; err != nil {
		createInternalErrorResponse(c)
		return
	}

	//Writes VolunterEmail to the database
	ve = VolunteerEmail{tm.ID, email}
	if err := db.Create(&ve).Error; err != nil {
		createInternalErrorResponse(c)
		return
	}
	c.JSON(200, ve)

}

//deassign Volunteer to team
func deassignVolunteerFromTeam(c *gin.Context) {
	identifier := c.Params.ByName("identifier")

	var ve VolunteerEmail
	//Validates json
	if err := c.BindJSON(&ve); err != nil {
		createBadRequestResponse(c, err)
		return
	}
	email := ve.VolunteerEmail

	//Gets VolunteerEmail from database
	err := db.Raw(getCfgString(cSelectVolunteerEmail), ve.VolunteerEmail, identifier).Find(&ve).Error
	if err != nil {
		createNotFoundResponse(c)
		return
	}

	//Gets Team from database
	var tm Team
	if err := db.Where("identifier = ?", identifier).First(&tm).Error; err != nil {
		createInternalErrorResponse(c)
		return
	}

	//Deletes VolunterEmail from database
	ve = VolunteerEmail{tm.ID, email}
	if err := db.Delete(VolunteerEmail{}, "volunteer_email=? and team_id=?", ve.VolunteerEmail, ve.TeamID).Error; err != nil {
		createInternalErrorResponse(c)
		return
	}
	c.JSON(200, ve)

}
