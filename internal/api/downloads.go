package api

import (
	"errors"
	"fmt"

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
	if vidtype == "video" || vidtype == "playlist" {
		var cname string
		var vidurl string
		if vidtype == "video" {
			vidurl = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoid)
		} else if vidtype == "playlist" {
			vidurl = fmt.Sprintf("https://www.youtube.com/playlist?list=%s", videoid)
		}
		metadata, err := tasking.GetYouTubeMetadata(vidurl, false)
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}

		creator, err := database.GetCreator(metadata.ChannelID, db)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		} else if err == nil {
			cname, err = general.SanitizeFileName(creator.Name)
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

	// Add video to download queue
	err = database.InsertDownloadQueueItem(downloadItem, db)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	c.JSON(200, gin.H{"ret": 200, "data": "Video added to download queue"})
}
