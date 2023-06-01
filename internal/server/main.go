package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Masterminds/sprig/v3"
	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/api"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

var settings *database.Settings
var db *gorm.DB

func Run() {

	db, err := database.GetDatabase()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	r := gin.Default()
	// load html templates

	r.SetFuncMap(sprig.FuncMap())

	api.Routes(r)

	// 404
	r.NoRoute(func(c *gin.Context) {
		if c.Writer.Status() == 401 {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		} else {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Page not found",
			})
		}
	})

	r.GET("/", func(c *gin.Context) {
		if jwttoken.IsLoggedIn(c) {
			c.Redirect(http.StatusTemporaryRedirect, "/home")
			c.Abort()
			return
		} else {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}
	})

	r.Static("/assets", "./static")

	r.GET("/favicon.ico", func(c *gin.Context) {
		// check if cached
		if c.Writer.Header().Get("Cache-Control") == "" {
			c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		c.File("./static/img/favicon.ico")
	})

	r.LoadHTMLGlob("./templates/*.tmpl")

	r.GET("/login", page_login(db))

	r.GET("/logout", page_logout(db))

	r.GET("/register", page_register(db))

	// require jwt token for all below routes
	r.Use(notLoggedInMiddleware(db))
	r.Use(jwttoken.JwtMiddleware())
	// reroute only 401 errors to login page

	r.GET("/home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.tmpl", gin.H{})
	})

	r.GET("/account", func(c *gin.Context) {
		// parse the jwt token and get the user id
		user, err := jwttoken.GetUserFromToken(c)
		if err != nil {
			c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})
			return
		}
		c.HTML(http.StatusOK, "account.tmpl", gin.H{
			"User": user,
		})
	})

	r.GET("/creators/:creator", page_creators_creator(db))

	r.GET("/library", page_library(db))

	r.GET("/creators", page_creators(db))

	r.GET("/subscriptions", page_subscriptions(db))

	r.GET("/download", func(c *gin.Context) {
		c.HTML(http.StatusOK, "download.tmpl", gin.H{})
	})

	r.GET("/download/:video_type/:video_id/", getDownloadPage)

	r.GET("/watch/:video_id", page_watch(db))

	// share all files in static folder to /assets

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

func init() {
	var err error
	// get database
	db, err = database.GetDatabase()
	if err != nil {
		fmt.Printf("Error connecting to database: %s\n", err)
		return
	}

	// get settings
	settings, err = database.GetSettings(db)
	if err != nil {
		fmt.Printf("Error getting settings: %s\n", err)
		return
	}

	// if ./newuser.txt exists, create a new user. Use os

	if _, err := os.Stat("./newuser.txt"); err == nil {
		// Open the file and read the contents to database.User
		f, err := os.Open("./newuser.txt")
		if err != nil {
			fmt.Printf("Error opening file: %s\n", err)
			return
		}
		defer f.Close()

		var userdata database.User
		err = json.NewDecoder(f).Decode(&userdata)
		if err != nil {
			fmt.Printf("Error decoding file: %s\n", err)
			return
		}

		// create user
		err = database.SignupUser(userdata, db)
		if err != nil {
			fmt.Printf("Error creating user: %s\n", err)
			return
		}

		// Close the file
		err = f.Close()
		if err != nil {
			fmt.Printf("Error closing file: %s\n", err)
			return
		}

		// delete file
		err = os.Remove("./newuser.txt")
		if err != nil {
			fmt.Printf("Error deleting file: %s\n", err)
			return
		}

		fmt.Printf("New user created: %v\n", userdata.Username)
	}
}
