package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
