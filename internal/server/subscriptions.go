package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

func page_subscriptions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		cookie, err := c.Cookie("jwt-token")
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Error getting token",
			})
			return
		}

		// parse jwt token
		parsedToken, err := jwttoken.ParseToken(cookie, settings.JwtSecret)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Error parsing token",
			})
			return
		}

		claims := parsedToken.Claims.(jwt.MapClaims)
		user := claims["user_id"].(string)

		page := c.Query("page")

		if page == "" {
			page = "1"
		}

		pageInt, err := strconv.Atoi(page)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Page not found",
			})
			return
		}

		// check filter
		var filterQuery string
		filter := c.Query("filter")
		if filter == "" {
			filter = "all"
		}

		filterList := getFilterList()

		if filter != "" {
			// check if filter is valid
			if _, ok := filterList[filter]; !ok {
				// Give invalid filter error
				c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
					"ret": 404,
					"err": "Invalid filter type",
				})
				return
			}
			filterQuery = filterList[filter]
		}

		// check sort
		var sortQuery string
		sort := c.Query("sort")
		if sort == "" {
			sort = "newest"
		}

		sortList := map[string]string{
			"newest":     "published_at DESC, created_at DESC, id DESC",
			"oldest":     "published_at ASC, created_at ASC, id ASC",
			"mostviews":  "CAST(views AS int) DESC, created_at DESC, id DESC",
			"leastviews": "CAST(views AS int) ASC, created_at ASC, id ASC",
			"mostlikes":  "CAST(likes AS int) DESC, created_at DESC, id DESC",
			"leastlikes": "CAST(likes AS int) ASC, created_at ASC, id ASC",
		}

		if sort != "" {
			// check if sort is valid
			if _, ok := sortList[sort]; !ok {
				// Give invalid sort error
				c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
					"ret": 404,
					"err": "Invalid sort type",
				})
				return
			}
			sortQuery = sortList[sort]
		}

		vidargs := db.Select("title", "video_id", "likes", "views", "channel_title", "channel_id", "published_at", "description", "length", "video_type", "availability").Limit(20).Offset((pageInt - 1) * 20)

		// Get the user's subscriptions
		userSubscriptions, err := database.GetPlaylistByUserID(user, "Subscriptions", db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "No subscriptions found",
			})
			return
		}

		// Add subscriptions to query
		if len(userSubscriptions) > 0 {
			for _, sub := range userSubscriptions {
				if filterQuery != "" {
					vidargs = vidargs.Or(filterQuery).Where("channel_id = ?", sub)
				} else {
					vidargs = vidargs.Or("channel_id = ?", sub)
				}
			}
		} else {
			// No subscriptions found
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "No subscriptions found",
			})
			return
		}

		if sortQuery != "" {
			// when casting we may get non-int values, so we need to filter those out
			if sort == "mostviews" || sort == "leastviews" {
				vidargs = vidargs.Where("views != ''")
				vidargs = vidargs.Where("views NOT like '%[^0-9]%'")
			}
			if sort == "mostlikes" || sort == "leastlikes" {
				vidargs = vidargs.Where("likes != ''").Where("likes != 'None'")
				vidargs = vidargs.Where("likes NOT like '%[^0-9]%'")
			}
			vidargs = vidargs.Order(sortQuery)
		}

		videos, err := database.GetAllVideos(vidargs)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": http.StatusInternalServerError,
				"err": fmt.Sprintf("Error getting videos: %v", err),
			})
			return
		}

		if len(videos) == 0 {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "No videos found",
			})
			return
		}

		// check if there is a next page, even if there's 20 videos, there might not be a next page

		nextPageFound := false
		if len(videos) == 20 {
			tmp, err := database.GetAllVideos(db.Select("video_id").Order("published_at DESC").Limit(1).Offset(pageInt * 20))
			if err == nil && len(tmp) > 0 {
				nextPageFound = true
			}
		}

		// truncate description

		for i, video := range videos {
			if len(video.Description) > 250 {
				videos[i].Description = video.Description[:247] + "..."
			}
		}

		// get user watched videos, checking user from jwt token

		// get jwt token from cookie

		// get watched videos from database
		watchedVideos, err := database.GetPlaylistByUserID(user, "Completed Videos", db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get watched videos",
			})
			return
		}

		if len(watchedVideos) > 0 {
			for i, video := range videos {
				for _, watchedVideo := range watchedVideos {
					if video.VideoID == watchedVideo {
						videos[i].Watched = true
					}
				}

			}
		}

		// get video progress
		allProg, err := database.GetAllVideoProgress(user, db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get video progress",
			})
			return
		}

		if len(allProg) > 0 {
			for i, video := range videos {
				for _, prog := range allProg {
					if video.VideoID == prog.VideoID {
						// string to float
						tempProg, err := strconv.ParseFloat(prog.Progress, 64)
						if err != nil {
							c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
								"ret": 404,
								"err": "Unable to parse video progress",
							})
							return
						}

						secLength := general.TimeConversion(video.Length)

						// get video length as float
						if secLength == 0 {
							// try to convert to int
							secLength, err = strconv.Atoi(video.Length)
							if err != nil {
								c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
									"ret": 404,
									"err": "Unable to parse video length",
								})
								return
							}
						}
						// calculate percentage as string using fmt
						videos[i].Progress = fmt.Sprintf("%.2f", (tempProg/float64(secLength))*100)
					}
				}
			}
		}

		// Get all sponsorblock from db here action_type = 'full'
		sponsorArgs := db.Select("video_id", "category", "action_type")
		for _, video := range videos {
			sponsorArgs = sponsorArgs.Or("video_id = ?", video.VideoID)
		}
		sponsorArgs = sponsorArgs.Where("action_type = 'full'")
		sponsorblock, err := database.GetAllVideoSponsoBlock(sponsorArgs)
		if err == nil {
			for i, video := range videos {
				for _, sponsor := range sponsorblock {
					if sponsor.ActionType != "full" {
						continue
					}
					if video.VideoID == sponsor.VideoID {
						videos[i].SponsorTag = sponsor.Category
					}
				}
			}
		}

		// Round views adding K for thousands and M for millions
		for i, video := range videos {
			videos[i].Views = general.FormatViews(video.Views)
			videos[i].Length = general.FormatDuration(video.Length)
		}

		ret := gin.H{
			"Videos":     videos,
			"UserID":     user,
			"ServerPath": settings.ServerPath,
		}

		if nextPageFound {
			ret["NextPage"] = pageInt + 1
		}

		if page != "1" {
			ret["PrevPage"] = pageInt - 1
		}

		if filter != "" {
			ret["Filter"] = filter
		}
		if sort != "" {
			ret["Sort"] = sort
		}

		c.HTML(http.StatusOK, "subscriptions.tmpl", ret)
	}
}
