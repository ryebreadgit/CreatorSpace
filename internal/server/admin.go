package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"gorm.io/gorm"
)

func page_user_management(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		// Get all users
		users, err := database.GetUsers(db.Order("id asc"))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": 500,
				"err": err,
			})
			return
		}

		// For each user get their role
		for i, user := range users {
			role, err := database.GetUserByID(user.UserID, db)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
					"ret": 500,
					"err": err,
				})
				return
			}
			users[i].AccountType = role.AccountType
		}

		c.HTML(http.StatusOK, "user-management.tmpl", gin.H{
			"User":      user,
			"Users":     users,
			"PageTitle": "User Management",
		})
	}
}

func page_library_management(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		// Get all creator info
		creators, err := database.GetAllCreators(db.Order("LOWER(platform) desc, LOWER(name) asc"))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": 500,
				"err": err,
			})
			return
		}

		var creatorIDs []string
		var variousCreators int

		// Get video count for each creator
		for i := range creators {
			if creators[i].ChannelID == "000" {
				variousCreators = i
			}

			// Get video count
			vids, err := database.GetAllVideos(db.Where("channel_id = ?", creators[i].ChannelID).Select("id"))
			if err != nil {
				creators[i].VideoCount = 0
			}

			creators[i].VideoCount = len(vids)

			creatorIDs = append(creatorIDs, creators[i].ChannelID)
		}

		// Get video count for videos where channel_id is not in creatorIDs
		vids, err := database.GetAllVideos(db.Where("channel_id NOT IN ?", creatorIDs).Select("id"))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": 500,
				"err": err,
			})
			return
		}

		creators[variousCreators].VideoCount = len(vids)

		// Get the video

		c.HTML(http.StatusOK, "library-management.tmpl", gin.H{
			"Creators":   creators,
			"User":       user,
			"BaseYTPath": settings.BaseYouTubePath,
			"PageTitle":  "Library Management",
		})
	}
}
