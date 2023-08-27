package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"github.com/ryebreadgit/CreatorSpace/internal/tasking"
	"gorm.io/gorm"
)

// Add video to download queue
func apiDownloadVideo(c *gin.Context) {
	// get video id and type from request
	var downloadItem database.DownloadQueue
	videoid := c.Param("video_id")
	vidtype := c.Param("video_type")

	// get user id from jwt token
	userid, err := jwttoken.GetUserFromToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": "Invalid token"})
		return
	}

	// Check if user is admin
	userData, err := database.GetUserByID(userid, db.Select("account_type"))
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	// Check if video exists
	_, err = database.GetVideo(videoid, db)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	} else if err == nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": "Video already exists"})
		return
	}

	// Check if video is already in download queue
	_, err = database.GetDownloadQueueItem(videoid, vidtype, db)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	} else if err == nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": "Video already in download queue"})
		return
	}

	downloadItem.VideoID = videoid
	downloadItem.VideoType = vidtype
	downloadItem.Requester = userid

	if userData.AccountType == "admin" {
		downloadItem.Approved = true
	}

	// If the video type is video get metadata. If the video creator is not in the database, set the creator id 000.
	if vidtype == "video" {
		var cname string
		vidurl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoid)
		metadata, err := tasking.GetYouTubeMetadata(vidurl, false)
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}

		// Check width and height, if vertical and duration is 60 seconds or less, set video type to short
		if metadata.Width < metadata.Height && metadata.Duration <= 60 {
			downloadItem.VideoType = "short"
		}

		creator, err := database.GetCreator(metadata.ChannelID, db)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		} else if err == nil {
			// creator.FilePath = "/${creator_name}/${creator_name}.json", pull the creator name from json name.
			tmpslc := strings.Split(creator.FilePath, "/")
			tmpname := tmpslc[len(tmpslc)-2]

			cname, err = general.SanitizeFileName(tmpname)
			if err != nil {
				c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
				return
			}
		} else {
			cname = "Various Creators"
		}
		downloadItem.Source = "youtube"
		downloadItem.DownloadPath = fmt.Sprintf("%s/%s/videos/%s", settings.BaseYouTubePath, cname, videoid)
	}

	if vidtype == "twitter" {
		downloadItem.Source = "twitter"
		downloadItem.DownloadPath = fmt.Sprintf("%s/%s", settings.BaseTwitterPath, videoid)
	}

	// Add video to download queue
	err = database.InsertDownloadQueueItem(downloadItem, db)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	c.JSON(200, gin.H{"ret": 200, "data": "Video added to download queue"})
}
