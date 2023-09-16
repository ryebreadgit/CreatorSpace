package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
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

		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		// get watched videos from database
		watchedVideos, err := database.GetPlaylistByUserID(user.UserID, "Completed Videos", db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get watched videos",
			})
			return
		}

		var creatorIDs []string

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

			for _, creator := range creators {
				creatorIDs = append(creatorIDs, creator.ChannelID)
			}
		}

		vididquery := db.Select("video_id").Order("published_at DESC")
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

		if len(creatorIDs) == 0 {
			vididquery = vididquery.Where("channel_id = ?", creatorid)
		} else {
			vididquery = vididquery.Where("channel_id NOT IN ?", creatorIDs)
		}

		// vididquery.Limit(250).Offset((pageInt - 1) * 25)
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

		vidargs := db.Select("title", "video_id", "views", "length", "published_at", "availability", "channel_id", "channel_title").Limit(20).Offset((pageInt - 1) * 20)

		if len(creatorIDs) == 0 {
			vidargs = vidargs.Where("channel_id = ?", creatorid)
		} else {
			vidargs = vidargs.Where("channel_id NOT IN ?", creatorIDs)
		}

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
			tmp, err := database.GetCreatorVideos(creatorid, vidargs.Select("video_id").Limit(1).Offset(pageInt*20))
			if err == nil && len(tmp) > 0 {
				nextPageFound = true
			}
		}

		creator, err := database.GetCreator(creatorid, db.Select("name", "channel_id", "description", "subscribers", "platform"))
		if err != nil {
			// As we've found videos we know there's a creator, so we can just use the first video to get the creator info and set the rest to Various Creators
			creator.Name = videos[0].ChannelTitle
			creator.ChannelID = videos[0].ChannelID
			creator.Description = "A Channel from Various Creators [Generated by CreatorSpace]"
			if videos[0].VideoType == "Twitch" {
				creator.Platform = "Twitch"
			} else if videos[0].VideoType == "Twitter" {
				creator.Platform = "Twitter"
			} else {
				creator.Platform = "YouTube"
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

		subscribed := false

		// Get user subscriptions and check if user is subscribed to creator
		subscriptions, err := database.GetPlaylistByUserID(user.UserID, "Subscriptions", db)
		if err != nil {
			subscriptions = []string{}
		}

		for _, subscription := range subscriptions {
			if subscription == creatorid {
				subscribed = true
			}
		}

		// get video progress
		allProg, err := database.GetAllVideoProgress(user.UserID, db)
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
			"User":           user,
			"Subscribed":     subscribed,
			"ServerPath":     settings.ServerPath,
			"PageTitle":      creator.Name,
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

		c.HTML(http.StatusOK, "creator.tmpl", ret)
	}
}

func page_creators(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		creators, err := database.GetAllCreators(db.Select("name", "channel_id", "linked_accounts").Order("LOWER(platform) DESC, LOWER(name) ASC"))

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

		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		c.HTML(http.StatusOK, "creators.tmpl", gin.H{
			"files":     baseFiles,
			"User":      user,
			"PageTitle": "Creators",
		})
	}
}
