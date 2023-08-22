package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
)

// use jwt token to authenticate user
func apiUserLogin(c *gin.Context) (string, error) {
	var userdata database.User
	err := c.ShouldBind(&userdata)
	if err != nil {
		return "", err
	}

	username := userdata.Username
	password := userdata.Password
	// get user from db
	data, err := database.LoginUser(username, password, db)
	if err != nil || data == (database.User{}) {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid username or password",
		})
		c.Abort()
		return "", err
	}

	// check account type

	switch data.AccountType {
	case "disabled":
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "User's account has been disabled, please contact the administrator.",
		})
		c.Abort()
		return "", err
	case "banned":
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "User's account has been banned, please contact the administrator.",
		})
		c.Abort()
		return "", err
	}

	// create jwt token
	token, err := jwttoken.CreateToken(data.UserID)
	if err != nil {
		return "", err
	}

	// set cookie
	jwttoken.SetToken(c, token)

	return token, nil

}

// delete jwt token from cookies
func apiUserLogout(c *gin.Context) (string, error) {
	jwttoken.DeleteToken(c)
	return "logged out", nil
}

// apiUserSignup create a new user. Use SignupUser function from database package
func apiUserSignup(c *gin.Context) (string, error) {
	// get user from db
	var user = database.User{AccountType: "user"}
	err := c.ShouldBind(&user)
	if err != nil {
		return "", err
	}

	if !settings.OpenRegister {
		return "", fmt.Errorf("registration is closed")
	}

	err = database.SignupUser(user, db)
	if err != nil {
		return "", err
	}

	// Login as our new user

	data, err := database.LoginUser(c.Param("username"), c.Param("password"), db)
	if err != nil {
		return "", err
	}

	// create jwt token

	tokenString, err := jwttoken.CreateToken(data.UserID)
	if err != nil {
		return "", err
	}

	// set cookie
	jwttoken.SetToken(c, tokenString)

	return tokenString, nil

}
