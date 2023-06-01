package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
)

func SetupDatabase() {
	router := gin.Default()

	router.LoadHTMLFiles("templates/setup.tmpl")

	router.GET("/", func(c *gin.Context) {
		// redirect to setup
		c.Redirect(http.StatusTemporaryRedirect, "/setup")
	})

	router.GET("/setup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "setup.tmpl", nil)
	})

	router.POST("/submit-setup", func(c *gin.Context) {
		openRegister, _ := strconv.ParseBool(c.PostForm("OpenRegister"))
		redisDB, _ := strconv.Atoi(c.PostForm("RedisDB"))

		settings := database.Settings{
			BaseYouTubePath:  c.PostForm("BaseYouTubePath"),
			BaseTwitchPath:   c.PostForm("BaseTwitchPath"),
			DatabaseType:     c.PostForm("DatabaseType"),
			DatabasePath:     c.PostForm("DatabasePath"),
			DatabaseHost:     c.PostForm("DatabaseHost"),
			DatabasePort:     c.PostForm("DatabasePort"),
			DatabaseUser:     c.PostForm("DatabaseUser"),
			DatabasePass:     c.PostForm("DatabasePass"),
			DatabaseName:     c.PostForm("DatabaseName"),
			DatabaseSSLMode:  c.PostForm("DatabaseSSLMode"),
			DatabaseTimeZone: c.PostForm("DatabaseTimeZone"),
			RedisAddress:     c.PostForm("RedisAddress"),
			RedisPassword:    c.PostForm("RedisPassword"),
			RedisDB:          redisDB,
			JwtSecret:        c.PostForm("JwtSecret"),
			OpenRegister:     openRegister,
		}

		// Remove trailing slash from BaseYouTubePath and BaseTwitchPath
		if settings.BaseYouTubePath[len(settings.BaseYouTubePath)-1:] == "/" || settings.BaseYouTubePath[len(settings.BaseYouTubePath)-1:] == "\\" {
			settings.BaseYouTubePath = settings.BaseYouTubePath[:len(settings.BaseYouTubePath)-1]
		}

		if settings.BaseTwitchPath[len(settings.BaseTwitchPath)-1:] == "/" || settings.BaseTwitchPath[len(settings.BaseTwitchPath)-1:] == "\\" {
			settings.BaseTwitchPath = settings.BaseTwitchPath[:len(settings.BaseTwitchPath)-1]
		}

		var err error
		// Sanitize BaseYouTubePath and BaseTwitchPath
		settings.BaseYouTubePath, err = general.SanitizeFilePath(settings.BaseYouTubePath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		settings.BaseTwitchPath, err = general.SanitizeFilePath(settings.BaseTwitchPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		// save settings to settings.json and restart the program
		f, err := os.Create("settings.json")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		defer f.Close()

		// convert settings to json
		settingsJson, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		// write settings to file
		_, err = f.Write(settingsJson)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		user := database.User{
			Username:    c.PostForm("Username"),
			Password:    c.PostForm("Password"),
			AccountType: "admin",
		}

		// Write user to ./newuser.txt
		f, err = os.Create("newuser.txt")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		defer f.Close()

		// convert user to json
		userJson, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		// write user to file
		_, err = f.Write(userJson)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}

		// restart the program

		general.RestartSelf()

		c.Redirect(http.StatusTemporaryRedirect, "/")

	})

	router.Run(":8080")
}
