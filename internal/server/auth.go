package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

func page_login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		openreg := settings.OpenRegister
		c.HTML(http.StatusOK, "login.tmpl", gin.H{
			"OpenRegister": openreg,
		})
	}
}

func page_logout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwttoken.DeleteToken(c)
		c.HTML(http.StatusOK, "logout.tmpl", gin.H{})
	}
}

func page_register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settings.OpenRegister {
			c.HTML(http.StatusOK, "register.tmpl", gin.H{})
		} else {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}
	}
}

func notLoggedInMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !jwttoken.IsLoggedIn(c) {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		} else {
			c.Next()
		}
	}
}

func setUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user id from jwt token
		userid, err := jwttoken.GetUserFromToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Get user from database
		user, err := database.GetUserByID(userid, db)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Remove the password from the user
		user.Password = ""

		c.Set("user", user)

		// Continue with the next handler in the chain
		c.Next()
	}
}
