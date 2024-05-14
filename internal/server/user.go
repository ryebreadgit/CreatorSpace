package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/api"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"gorm.io/gorm"
)

var lastHealthCheck time.Time
var lastHealthStatus int
var lastHealthMsg string

func get_account(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		// Get the user's subscriptions
		subs, err := database.GetPlaylistByUserID(user.UserID, "Subscriptions", db)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Error getting subscriptions",
			})

			c.Abort()
			return
		}

		// Get the name and channel_id from the creator table
		var creators []database.Creator
		for _, sub := range subs {
			creator, err := database.GetCreator(sub, db.Select("name", "channel_id"))
			if err != nil {
				// Try to grab this via a video if one exists
				vid, err := database.GetCreatorVideos(sub, db.Select("channel_title", "channel_id").Limit(1))
				if err != nil {
					// If we can't find it, just use the sub id
					creator = database.Creator{
						Name:      "Missing Creator",
						ChannelID: sub,
					}
				} else {
					creator = database.Creator{
						Name:      vid[0].ChannelTitle,
						ChannelID: vid[0].ChannelID,
					}
				}
			}
			creators = append(creators, creator)
		}

		// Sort creator by name
		sort.Slice(creators, func(i, j int) bool {
			return strings.ToLower(creators[i].Name) < strings.ToLower(creators[j].Name)
		})

		c.HTML(http.StatusOK, "account.tmpl", gin.H{
			"User":          user,
			"Subscriptions": creators,
			"PageTitle":     "Account",
		})
	}
}

func health_check(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only run the health check every 5 minutes to avoid spamming the database
		if time.Since(lastHealthCheck) < 5*time.Minute {
			ret := gin.H{"ret": lastHealthStatus, "err": lastHealthMsg}
			if lastHealthStatus == 200 {
				// Drop the err key and set the data key
				ret["data"] = lastHealthMsg
				delete(ret, "err")
			}
			c.JSON(lastHealthStatus, ret)
			return
		}

		lastHealthCheck = time.Now()

		// Check if the database is up
		err := db.Exec("SELECT 1").Error
		if err != nil {
			lastHealthMsg = "Database is down"
			lastHealthStatus = 500
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Database is down",
			})
			return
		}

		// Check if yt-dlp is available
		ver := api.GetYTDLPVersion()
		if ver == "" || ver == "unknown" {
			lastHealthMsg = "yt-dlp is not available"
			lastHealthStatus = 500
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "yt-dlp is not available",
			})
			return
		}

		// Set the last health check time
		lastHealthStatus = 200
		lastHealthMsg = "OK"
		c.JSON(200, gin.H{
			"ret":  200,
			"data": "OK",
		})
	}
}
