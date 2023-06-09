package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

func page_creators_creator(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		creatorid := c.Param("creator")
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

		filterList := map[string]string{
			"all":         "",
			"public":      "availability = 'available'",
			"live":        "availability = 'live' OR video_type = 'Twitch'",
			"notlive":     "availability != 'live' AND video_type != 'Twitch'",
			"twitch":      "video_type = 'Twitch'",
			"unlisted":    "availability = 'unlisted'",
			"private":     "availability = 'private' OR availability = 'unavailable'",
			"unavailable": "availability = 'unavailable' OR availability = 'private' OR availability = 'unlisted'",
		}

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
			"newest":     "published_at DESC, id DESC",
			"oldest":     "published_at ASC, id ASC",
			"mostviews":  "CAST(views AS int) DESC, id DESC",
			"leastviews": "CAST(views AS int) ASC, id ASC",
			"mostlikes":  "CAST(likes AS int) DESC, id DESC",
			"leastlikes": "CAST(likes AS int) ASC, id ASC",
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

		vidargs := db.Select("title", "video_id", "views", "length", "published_at", "availability").Limit(20).Offset((pageInt - 1) * 20)
		if filterQuery != "" {
			vidargs = vidargs.Where(filterQuery)
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

		// If channel id is 000, then get videos from all creators that are not in the creator table. Get all creators, then add .Where statement to vidargs get videos from all creators that are not in the creator table
		if creatorid == "000" {
			creators, err := database.GetAllCreators(db.Select("channel_id"))
			if err != nil {
				c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
					"ret": 404,
					"err": err.Error(),
				})
				return
			}

			var creatorIDs []string
			for _, creator := range creators {
				creatorIDs = append(creatorIDs, creator.ChannelID)
			}

			vidargs = vidargs.Where("channel_id NOT IN ?", creatorIDs)
		} else {
			vidargs = vidargs.Where("channel_id = ?", creatorid)
		}

		videos, err := database.GetCreatorVideos(creatorid, vidargs)
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

		// check if there is a next page, even if there's 20 videos, there might not be a next page
		nextPageFound := false
		if len(videos) == 20 {
			tmp, err := database.GetCreatorVideos(creatorid, db.Select("video_id").Order("published_at DESC").Limit(1).Offset(pageInt*20))
			if err == nil && len(tmp) > 0 {
				nextPageFound = true
			}
		}

		creator, err := database.GetCreator(creatorid, db.Select("name", "channel_id", "description", "subscribers", "platform"))
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Creator not found",
			})
			return
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

		if len(watchedVideos) > 0 {
			for i, video := range videos {
				for _, watchedVideo := range watchedVideos {
					if video.VideoID == watchedVideo {
						videos[i].Watched = true
					}
				}

			}
		}

		subscribed := false

		// Get user subscriptions and check if user is subscribed to creator
		subscriptions, err := database.GetPlaylistByUserID(user, "Subscriptions", db)
		if err != nil {
			subscriptions = []string{}
		}

		for _, subscription := range subscriptions {
			if subscription == creatorid {
				subscribed = true
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

		// format views and length
		for i, video := range videos {
			videos[i].Views = general.FormatViews(video.Views)
			videos[i].Length = general.FormatDuration(video.Length)
		}

		// Get video count
		videoCount, err := database.GetCreatorVideoCount(creatorid, db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get video count",
			})
			return
		}

		// format videoCount
		creator.VideoCount = videoCount

		// format subscribers
		creator.Subscribers = general.FormatViews(creator.Subscribers)

		linkedaccs, err := general.ParseLinkedAccounts(creator.LinkedAccounts)
		if err != nil {
			linkedaccs = []general.LinkedAccountsStruct{}
		}

		ret := gin.H{
			"Videos":         videos,
			"Creator":        creator,
			"LinkedAccounts": linkedaccs,
			"UserID":         user,
			"Subscribed":     subscribed,
			"ServerPath":     settings.ServerPath,
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

		c.HTML(http.StatusOK, "creator.tmpl", ret)
	}
}

func page_creators(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		creators, err := database.GetAllCreators(db.Select("name", "channel_id", "linked_accounts"))

		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Creator not found",
			})
			return
		}

		var baseFiles []database.Creator
		for _, creator := range creators {
			sendCr := database.Creator{}
			sendCr.Name = creator.Name
			sendCr.ChannelID = creator.ChannelID
			baseFiles = append(baseFiles, sendCr)
		}

		// sort creators by name, ignore case
		sort.Slice(baseFiles, func(i, j int) bool {
			return strings.ToLower(baseFiles[i].Name) < strings.ToLower(baseFiles[j].Name)
		})

		c.HTML(http.StatusOK, "creators.tmpl", gin.H{
			"files": baseFiles,
		})
	}
}
