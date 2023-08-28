package api

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	log "github.com/sirupsen/logrus"
)

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get the user from jwt
		userID, err := jwttoken.GetUserFromToken(c)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": err.Error()})
			return
		}
		if userID == "" {
			c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": "Not logged in"})
			return
		}

		// Get the user from the database
		user, err := database.GetUserByID(userID, db)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": err.Error()})
			return
		}

		// Check if the user is an admin
		if user.AccountType != "admin" {
			c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": "Not an admin"})
			return
		}

		// Continue
		c.Next()
	}
}

func apiGetAllUsers(c *gin.Context) {
	// Get all users
	users, err := database.GetUsers(db)
	if err != nil {
		c.JSON(500, gin.H{"ret": 500, "err": err.Error()})
		return
	}

	// Return the users
	c.JSON(200, gin.H{"ret": 200, "data": users})
}

func apiGetUser(c *gin.Context) {
	// Get the user id
	userID := c.Param("user_id")

	// Get the user
	user, err := database.GetUserByID(userID, db)
	if err != nil {
		c.JSON(500, gin.H{"ret": 500, "err": err.Error()})
		return
	}

	user.Password = ""

	// Return the user
	c.JSON(200, gin.H{"ret": 200, "data": user})
}

func apiCreateUser(c *gin.Context) {
	// Get the user data
	var userdata database.User
	err := c.ShouldBind(&userdata)
	if err != nil {
		c.JSON(400, gin.H{"ret": 400, "err": err.Error()})
		return
	}

	err = database.SignupUser(userdata, db)
	if err != nil {
		c.JSON(409, gin.H{"ret": 409, "err": err.Error()})
		return
	}

	// Return the user
	c.JSON(200, gin.H{"ret": 200, "data": userdata})
}

func apiDeleteUser(c *gin.Context) {
	// Get the user id
	reqUser := c.Param("user_id")

	userId, err := jwttoken.GetUserFromToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token",
		})
		c.Abort()
		return
	}

	// Check if the user is deleting themselves
	if reqUser == userId {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Cannot delete yourself using admin api",
		})
		c.Abort()
		return
	}

	// Check if the reqUser is an admin
	user, err := database.GetUserByID(reqUser, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user",
		})
		c.Abort()
		return
	}

	if user.AccountType == "admin" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Cannot delete an admin, must clear admin status first",
		})
		c.Abort()
		return
	}

	// Delete the user
	err = database.DeleteUser(reqUser, db)
	if err != nil {
		c.JSON(500, gin.H{"ret": 500, "err": err.Error()})
		return
	}

	log.Warn("Admin '" + userId + "' updated the password for user '" + reqUser + "'")

	// Return the user
	c.JSON(200, gin.H{"ret": 200})
}

func apiUpdateUserPassword(c *gin.Context) {
	reqUser := c.Param("user_id")

	userId, err := jwttoken.GetUserFromToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token",
		})
		c.Abort()
		return
	}

	if reqUser == userId {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Cannot update your own password using admin api",
		})
		c.Abort()
		return
	}

	// Get user from database
	user, err := database.GetUserByID(reqUser, db.Select("user_id", "password"))
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user",
		})
		c.Abort()
		return
	}

	var passwords struct {
		NewPassword string
	}

	// bind the request body to the struct
	err = json.NewDecoder(io.LimitReader(c.Request.Body, 1048576)).Decode(&passwords)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Invalid request body",
		})
	}

	// check if passwords is empty
	if passwords.NewPassword == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Password is empty",
		})

		c.Abort()
		return
	}

	// hash the new password
	hashedPassword, err := database.HashPassword(passwords.NewPassword)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error hashing password",
		})
		c.Abort()
		return
	}

	// update the user's password
	user.Password = hashedPassword
	err = database.UpdateUser(user, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating password",
		})
		c.Abort()
		return
	}

	log.Warn("Admin '" + userId + "' updated the password for user '" + reqUser + "'")

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Password updated",
		},
	})
}

func apiUpdateUserRole(c *gin.Context) {
	reqUser := c.Param("user_id")

	userId, err := jwttoken.GetUserFromToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token",
		})
		c.Abort()
		return
	}

	if reqUser == userId {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Cannot update your own role using admin api",
		})
		c.Abort()
		return
	}

	// Get user from database
	user, err := database.GetUserByID(reqUser, db.Select("user_id", "account_type"))
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user",
		})
		c.Abort()
		return
	}

	var role struct {
		NewRole string
	}

	// bind the request body to the struct
	err = json.NewDecoder(io.LimitReader(c.Request.Body, 1048576)).Decode(&role)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Invalid request body",
		})
	}

	// check if role is empty
	if role.NewRole == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Role is empty",
		})

		c.Abort()
		return
	}

	role.NewRole = strings.ToLower(role.NewRole)

	// Check if new role in database.GetValidUserTypes
	validRoles := database.GetValidUserTypes()
	valid := false
	for _, validRole := range validRoles {
		if role.NewRole == validRole {
			valid = true
		}
	}
	if !valid {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Invalid role",
		})

		c.Abort()
		return
	}

	// update the user's password
	user.AccountType = role.NewRole
	err = database.UpdateUser(user, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating role",
		})
		c.Abort()
		return
	}

	log.Warn("Admin '" + userId + "' updated the role for user '" + reqUser + "' to '" + role.NewRole + "'")

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Role updated",
		},
	})
}
