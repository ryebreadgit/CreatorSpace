package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

func get_account(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// parse the jwt token and get the user id
		user, err := jwttoken.GetUserFromToken(c)
		if err != nil {
			c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})
			return
		}

		// Get the user's subscriptions
		subs, err := database.GetPlaylistByUserID(user, "Subscriptions", db)
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
		})
	}
}
