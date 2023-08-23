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

func page_library(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		filterList := getFilterList()
		allFilters, ok := c.GetQueryArray("filter")
		if !ok {
			allFilters = []string{"all"}
		}
		watchFilter := 0

		for _, filter := range allFilters {
			if filter == "watched" {
				watchFilter = 1
				continue
			} else if filter == "notwatched" {
				watchFilter = 2
				continue
			}
			if filter != "" {
				// check if filter is valid
				if _, ok := filterList[filter]; !ok {
					// Give invalid filter error
					c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
						"ret": 404,
						"err": "Invalid filter type: " + filter,
					})
					return
				}
				if filterQuery == "" {
					filterQuery = filterList[filter]
				} else {
					filterQuery = filterQuery + " AND " + filterList[filter]
				}
			}
		}

		// check sort
		var sortQuery string
		sort := c.Query("sort")
		if sort == "" {
			sort = "newest"
		}

		sortList := getSortList()

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

		// get user from jwt, parse it and get the user from the database
		token, err := jwttoken.GetToken(c)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get token",
			})
			return
		}

		parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to parse token",
			})
			return
		}

		// get user_id from parsed token using jwt package
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get claims",
			})
			return
		}

		user := claims["user_id"].(string)

		// get watched videos from database
		watchedVideos, err := database.GetPlaylistByUserID(user, "Completed Videos", db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get watched videos",
			})
			return
		}

		vididquery := db.Select("video_id")
		if filterQuery != "" {
			vididquery = vididquery.Where(filterQuery)
		}
		if sortQuery != "" {
			if sort == "mostviews" || sort == "leastviews" {
				vididquery = vididquery.Where("views != ''")
				vididquery = vididquery.Where("views NOT like '%[^0-9]%'")
			}
			if sort == "mostlikes" || sort == "leastlikes" {
				vididquery = vididquery.Where("likes != ''").Where("likes != 'None'")
				vididquery = vididquery.Where("likes NOT like '%[^0-9]%'")
			}
			vididquery = vididquery.Order(sortQuery)
		}

		// vididquery.Limit(2500).Offset((pageInt - 1) * 25
		vidIds, err := database.GetAllVideos(vididquery)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": http.StatusInternalServerError,
				"err": fmt.Sprintf("Error getting videos: %v", err),
			})
			return
		}

		subVideoIDs := make([]string, len(vidIds))
		for i, video := range vidIds {
			subVideoIDs[i] = video.VideoID
		}

		vidargs := db.Select("title", "video_id", "views", "length", "published_at", "availability", "channel_id", "channel_title").Limit(25).Offset((pageInt - 1) * 25)

		if watchFilter == 1 {
			// only show videos in watchedVideos
			if len(watchedVideos) > 0 {
				vidargs = vidargs.Where("video_id IN (?)", intersection(subVideoIDs, watchedVideos))
			} else {
				// No watched videos found
				c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
					"ret": 404,
					"err": "No watched videos found",
				})
				return
			}
		} else if watchFilter == 2 {
			// remove videos in watchedVideos
			if len(watchedVideos) > 0 {
				vidargs = vidargs.Where("video_id IN (?)", difference(subVideoIDs, watchedVideos))
			} else {
				// All videos are unwatched
				vidargs = vidargs.Where("video_id IN (?)", subVideoIDs)
			}
		} else {
			vidargs = vidargs.Where("video_id IN (?)", subVideoIDs)
		}

		if filterQuery != "" {
			vidargs = vidargs.Where(filterQuery)
		}
		if sortQuery != "" {
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
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "No videos found",
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

		// check if there is a next page, even if there's 25 videos, there might not be a next page
		nextPageFound := false
		if len(videos) >= 25 {
			tmp, err := database.GetAllVideos(vidargs.Select("video_id").Limit(1).Offset(pageInt * 25))
			if err == nil && len(tmp) > 0 {
				nextPageFound = true
			}
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
				"err": "Unable to get all video progress",
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
								"err": "Unable to parse video progress for '" + video.Title + "'",
							})
							return
						}

						// convert video length
						secLength := general.TimeConversion(video.Length)

						// get video length as float
						if secLength == 0 {
							// try to convert to int
							secLength, err = strconv.Atoi(video.Length)
							if err != nil {
								c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
									"ret": 404,
									"err": "Unable to parse video length for '" + video.Title + "'",
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

		// format views and length
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

		ret["Filter"] = allFilters[0]
		if sort != "" {
			ret["Sort"] = sort
		}

		c.HTML(http.StatusOK, "library.tmpl", ret)
	}
}
