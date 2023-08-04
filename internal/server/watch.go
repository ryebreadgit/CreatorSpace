package server

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ryebreadgit/CreatorSpace/internal/api"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

func page_watch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		videoid := c.Param("video_id")
		video, err := database.GetVideo(videoid, db.Select("title", "video_id", "channel_id", "channel_title", "description", "published_at", "length", "likes", "views", "subtitle_path", "video_type"))
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Video not found",
			})
			return
		}

		if video.Length == "" {
			video.Length = "0"
		}

		// format video views
		video.Views = general.FormatViews(video.Views)

		creator, err := database.GetCreator(video.ChannelID, db.Select("name", "channel_id", "description", "subscribers", "platform"))
		if err != nil {
			// set temp creator data
			creator = database.Creator{
				ChannelID: video.ChannelID,
				Name:      video.ChannelTitle,
				Platform:  video.VideoType,
			}
		}

		// format creator subscribers
		creator.Subscribers = general.FormatViews(creator.Subscribers)

		// set creator video count
		creator.VideoCount, err = database.GetCreatorVideoCount(creator.ChannelID, db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Video creator not found",
			})
			return
		}

		// get video sponsorblock

		var sponsorblock []database.SponsorBlock
		sponsorblock, err = database.GetVideoSponsorBlock(videoid, db.Select("segment_start", "segment_end", "category"))
		if err != nil {
			sponsorblock = nil
		}

		// get video comments, only get the first 25.

		comments, err := database.GetVideoComments(videoid, db.Select("comment_id", "author", "author_id", "text", "time_parsed", "votes", "parent_comment_id").Order("votes DESC").Where("parent_comment_id = ?", "").Limit(25))

		if err != nil {
			comments = nil
		}

		// Sort by votes since this is a string and not done by the database
		if comments != nil {
			for i := 0; i < len(comments); i++ {
				for j := 0; j < len(comments)-1; j++ {

					// convert K, M, B and decimals to full number in string

					vote1, err := database.ConvertVote(comments[j].Votes)
					if err != nil {
						// just put it at the bottom
						vote1 = 0
					}
					vote2, err := database.ConvertVote(comments[j+1].Votes)
					if err != nil {
						vote2 = 0
					}
					if vote1 < vote2 {
						comments[j], comments[j+1] = comments[j+1], comments[j]
					}
				}
			}
		}

		// convert TimeParsed from epoch to human readable
		for i, comment := range comments {
			// use time to parse to human readable string
			// float64 to int64
			temptime := time.Unix(int64(comment.TimeParsed), 0).Format("2006-01-02 15:04:05")
			// set the new time in string format
			comments[i].TimeString = temptime

			// convert votes to human readable string
			comments[i].Votes = general.FormatViews(comment.Votes)
		}

		// Move comments by the poster to the top
		if comments != nil {
			for i := 0; i < len(comments); i++ {
				if comments[i].AuthorID == creator.ChannelID {
					temp := comments[i]
					for j := i; j > 0; j-- {
						comments[j], comments[j-1] = comments[j-1], comments[j]
					}
					comments[0] = temp
				}
			}
		}

		mimeType := mime.TypeByExtension(video.FilePath)
		if mimeType == "" {
			mimeType = "video/mp4"
		}
		video.MimeType = mimeType

		// get video progress for current user
		// get jwt token from cookie

		unparsedToken, err := c.Cookie("jwt-token")
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Token not found",
			})
			c.Abort()
			return
		}

		parsedToken, err := jwttoken.ParseToken(unparsedToken, settings.JwtSecret)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Token not found",
			})
			c.Abort()
			return
		}

		// get user_id from token
		claims := parsedToken.Claims.(jwt.MapClaims)
		user := claims["user_id"].(string)

		vidProgress, err := database.GetVideoProgress(videoid, user, db)
		if err != nil {
			vidProgress = "0"
		}

		// get sponsorblockenabled and sponsorblockcategories for current user
		sponsorblockEnabled, sponsorblockCategories, err := database.GetSponsorBlockSettings(user, db)
		if err != nil {
			sponsorblockEnabled = false
			sponsorblockCategories = ""
		}

		var subtitles []database.VidSubtitle

		// Unmarshal subtitles
		if video.SubtitlePath != "" {
			subtitles, err = database.GetVideoSubtitles(videoid, db)
			if err != nil {
				subtitles = nil
			}
		}

		// get watched videos from database
		watchedVideos, err := database.GetPlaylistByUserID(user, "Completed Videos", db)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
				"ret": 404,
				"err": "Unable to get watched videos",
			})
			return
		}

		// Get video recommendations
		recs, err := api.GetRecommendations(videoid, watchedVideos)
		if err != nil {
			recs = nil
		}

		if len(watchedVideos) > 0 {
			for i, video := range recs {
				for _, watchedVideo := range watchedVideos {
					if video.VideoID == watchedVideo {
						recs[i].Watched = true
					}
				}
			}

			// Check if current video is in watched videos
			for _, watchedVideo := range watchedVideos {
				if videoid == watchedVideo {
					video.Watched = true
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
			for i, video := range recs {
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
						recs[i].Progress = fmt.Sprintf("%.2f", (tempProg/float64(secLength))*100)
					}
				}
			}
		}

		// Round views adding K for thousands and M for millions
		for i, video := range recs {
			recs[i].Views = general.FormatViews(video.Views)
			recs[i].Length = general.FormatDuration(video.Length)
		}

		c.HTML(http.StatusOK, "watch.tmpl", gin.H{
			"Video":                  video,
			"Creator":                creator,
			"SponsorBlock":           sponsorblock,
			"SponsorBlockEnabled":    sponsorblockEnabled,
			"SponsorBlockCategories": sponsorblockCategories,
			"Comments":               comments,
			"Progress":               vidProgress,
			"UserID":                 user,
			"Subtitles":              subtitles,
			"Recommendations":        recs,
			"ServerPath":             settings.ServerPath,
		})
	}
}
