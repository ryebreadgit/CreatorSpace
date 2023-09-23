package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/api"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	log "github.com/sirupsen/logrus"
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
	r := gin.New()
	r.Use(customRecovery())
	r.Use(errorHandlingMiddleware())
	r.Use(loggingMiddleware())
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
	r.Use(setUserMiddleware())
	// reroute only 401 errors to login page

	r.GET("/home", func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)
		c.HTML(http.StatusOK, "home.tmpl", gin.H{
			"User":      user,
			"PageTitle": "Home",
		})
	})

	r.GET("/account", get_account(db))

	r.GET("/creators/:creator", page_creators_creator(db))

	r.GET("/library", page_library(db))

	r.GET("/creators", page_creators(db))

	r.GET("/subscriptions", page_subscriptions(db))

	r.GET("/download", func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)
		c.HTML(http.StatusOK, "download.tmpl", gin.H{
			"User":      user,
			"PageTitle": "Download",
		})
	})

	r.GET("/download/:video_type/:video_id/", getDownloadPage)

	r.GET("/watch/:video_id", page_watch(db))

	r.Use(isAdminMiddleware())
	r.GET("/user-management", page_user_management(db))
	r.GET("/library-management", page_library_management(db))

	// share all files in static folder to /assets

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

func errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute next handlers in the chain
		c.Next()

		// If there are errors after executing all handlers, log them
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				if strings.Contains(e.Error(), "broken pipe") || strings.Contains(e.Error(), "connection reset by peer") {
					log.Debugf("Gin error: %s", e.Error())
				} else {
					log.Warnf("Gin error: %s", e.Error())
				}
			}
			c.Abort() // Abort the context to prevent other handlers from executing
		}
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		// Execute next handlers in the chain
		c.Next()

		// Check if client has closed before setting latency
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		// Calculate latency using timer
		latency := time.Since(start)

		logData := gin.H{
			"ip":         c.ClientIP(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"user_agent": c.Request.UserAgent(),
		}

		user, err := jwttoken.GetUserFromToken(c)
		if err == nil {
			logData["user"] = user
		}

		// If there is an error, add it to log data
		if len(c.Errors) > 0 {
			logData["error"] = c.Errors.String()
		}

		// To json
		logDataJson, err := json.Marshal(logData)
		if err != nil {
			log.Errorf("Error marshalling log data: %s", err)
		}

		if c.Writer.Status() == 401 {
			log.Warnf("Access Unauthorized: %s", logDataJson)
		} else if c.Writer.Status() >= 500 {
			log.Errorf("Access Error: %s", logDataJson)
		} else if c.Writer.Status() >= 400 {
			log.Debugf("Access Error: %s", logDataJson)
		} else if c.Writer.Status() >= 300 {
			log.Debugf("Access Redirect: %s", logDataJson)
		} else {
			log.Debugf("Access: %s", logDataJson)
		}
	}
}

func customRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); ok {
					if strings.Contains(e.Error(), "broken pipe") || strings.Contains(e.Error(), "connection reset by peer") {
						// Simply return if it's a broken pipe error
						log.Debugf("Gin error: %s", e.Error())
						return
					}
				}
				// If it's any other error, use the default gin recovery behavior
				log.Errorf("Gin panic recovered: %s\n\t%s", err, debug.Stack())
				gin.DefaultErrorWriter.Write([]byte(fmt.Sprintf("[Recovery] panic recovered:\n%s\n%s", err, debug.Stack())))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
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
