package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sethvargo/go-password/password"
)

//deleteVolunteer for authentificated user and his data
func deleteVolunteer(c *gin.Context) {
	email := c.Params.ByName("email")
	var vol Volunteer
	if err := db.Where("email = ?", email).Find(&vol).Error; err != nil {
		createNotFoundResponse(c)
		return
	}

	//Checks if data belongs to the user
	if !volunteerAuth(c, &vol) {
		return
	}
	//Deletes from database
	db.Delete(&vol)

	//Deletes from authentification map
	delete(authMap, vol.Email)
	c.JSON(200, gin.H{"Message": email + " deleted"})
}

//updateVolunteer for authentificated user and his data
func updateVolunteer(c *gin.Context) {

	var vol Volunteer
	email := c.Params.ByName("email")

	//Checks json data
	if err := c.BindJSON(&vol); err != nil {
		createBadRequestResponse(c, err)
		return
	}

	//Gets volunteer from database
	var oldvol Volunteer
	if err := db.Where("email = ?", email).First(&oldvol).Error; err != nil {
		createNotFoundResponse(c)
		return
	}
	//Sets data which could not be changed (I am still not sure if url path have to be with :email )
	vol.ID = oldvol.ID
	vol.Email = oldvol.Email
	if vol.Password == "" {
		vol.Password = oldvol.Password
	}

	//Checks if data belongs to the user
	if !volunteerAuth(c, &vol) {
		return
	}

	//Saves Volunteer to the database
	if err := db.Save(&vol).Error; err != nil {
		createStatusConflictResponse(c)
		return
	}
	//change password in auth map
	authMap[vol.Email] = vol.Password
	c.JSON(200, vol)

}

//Creates new Volunteer
func createVolunteer(c *gin.Context) {

	var vol Volunteer
	//Checks json data
	if err := c.BindJSON(&vol); err != nil {
		createBadRequestResponse(c, err)
		return
	}

	//Generates password
	res, _ := password.Generate(16, 5, 3, false, true)
	vol.Password = res
	//Writes user to the database
	if err := db.Create(&vol).Error; err != nil {
		createStatusConflictResponse(c)
		return
	}
	//Add user credential to auth map
	authMap[vol.Email] = res

	//Sends email with password
	vol.sendEmailWithPassword()
	c.JSON(200, gin.H{"Message": vol.Email + " created"})
}

//getVolunteer for authentificated userand his data
//Question if is ok to show password I think no ;)
func getVolunteer(c *gin.Context) {
	email := c.Params.ByName("email")
	var vol Volunteer
	//Reads Volunteer from database
	if err := db.Where("email = ?", email).First(&vol).Error; err != nil {
		createNotFoundResponse(c)
		return
	}
	//Checks if data belongs to the user
	if !volunteerAuth(c, &vol) {
		return
	}
	//Sets volunteer json
	c.JSON(200, vol)
}

//getVolunteers only for reporter
func getVolunteers(c *gin.Context) {
	var vols []Volunteer
	//Read volunteers from database
	if err := db.Find(&vols).Error; err != nil {
		createNotFoundResponse(c)
		return
	}

	//Authorization if user is reporter
	if !reporterAuth(c) {
		return
	}
	c.JSON(200, vols)
}
