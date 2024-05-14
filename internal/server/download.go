package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/tasking"
	//twitterscraper "github.com/n0madic/twitter-scraper"
)

func getDownloadPage(c *gin.Context) {
	// get video_id and video_type from url
	videoid := c.Param("video_id")
	vidtype := c.Param("video_type")

	userData, exists := c.Get("user")
	if !exists {
		// Redirect to login
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		c.Abort()
		return
	}

	user := userData.(database.User)

	// get youtube metadata for video id
	var url string
	switch vidtype {
	case "video":
		url = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoid)
	case "playlist":
		url = fmt.Sprintf("https://www.youtube.com/playlist?list=%s", videoid)
	case "channel":
		url = fmt.Sprintf("https://www.youtube.com/channel/%s/videos", videoid)
	case "shorts":
		url = fmt.Sprintf("https://www.youtube.com/user/%s/shorts", videoid)
	case "streams":
		url = fmt.Sprintf("https://www.youtube.com/channel/%s/streams", videoid)
	case "twitter":
		url = fmt.Sprintf("https://twitter.com/%s", videoid)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video type"})
		return
	}

	var data database.YouTubeVideoInfoStruct
	var err error

	// Check if already in download queue
	_, err = database.GetDownloadQueueItem(videoid, vidtype, db)
	if err == nil {
		// Video already in download queue
		c.HTML(http.StatusConflict, "error.tmpl", gin.H{
			"ret": http.StatusConflict,
			"err": "Video already in download queue",
		})
		c.Abort()
		return
	}

	// get youtube-dl metadata
	if vidtype == "video" {
		data, err = tasking.GetYouTubeMetadata(url, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}
	} else if vidtype == "channel" || vidtype == "shorts" || vidtype == "streams" {
		creatorMetadata, err := tasking.GetCreatorMetadata(fmt.Sprintf("https://www.youtube.com/channel/%v/about", videoid))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}
		// capitalize first letter of video type
		data.Title = creatorMetadata.Uploader + " - " + strings.ToUpper(vidtype[:1]) + strings.ToLower(vidtype[1:])
		data.Description = creatorMetadata.Description
		// Get uncropped thumbnail and banner
		for _, thumb := range creatorMetadata.Thumbnails {
			if thumb.ID == "avatar_uncropped" {
				data.Thumbnail = thumb.URL
				break
			}
		}

		if data.Thumbnail == "" && len(creatorMetadata.Thumbnails) > 0 {
			data.Thumbnail = creatorMetadata.Thumbnails[0].URL
		}
		/*
			} else if vidtype == "twitter" {

				scraper := twitterscraper.New()
				err := scraper.LoginOpenAccount()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
					return
				}

				profile, err := scraper.GetProfile(videoid)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
					return
				}

				data.Title = profile.Name
				data.Thumbnail = strings.ReplaceAll(profile.Avatar, "_normal", "")
				data.Description = profile.Biography
		*/
	} else {
		data.Title = videoid
		data.Thumbnail = "https://i.ytimg.com/vi/" + videoid + "/hqdefault.jpg"
		data.Description = "Download '" + videoid + "'" + " from YouTube as a " + vidtype
	}

	c.HTML(http.StatusOK, "download-confirm.tmpl", gin.H{
		"VideoName":   data.Title,
		"Thumbnail":   data.Thumbnail,
		"Description": data.Description,
		"ID":          videoid,
		"Type":        vidtype,
		"ServerPath":  settings.ServerPath,
		"User":        user,
		"PageTitle":   "Confirm Download",
	})

}
