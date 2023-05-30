package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/tasking"
)

func getDownloadPage(c *gin.Context) {
	// get video_id and video_type from url
	videoid := c.Param("video_id")
	vidtype := c.Param("video_type")

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
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video type"})
		return
	}

	var data database.YouTubeVideoInfoStruct
	var err error

	// get youtube-dl metadata
	if vidtype == "video" {
		data, err = tasking.GetYouTubeMetadata(url, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ret": http.StatusInternalServerError, "err": err.Error()})
			return
		}
	} else {
		data.Title = videoid
		data.Thumbnail = "https://i.ytimg.com/vi/" + videoid + "/hqdefault.jpg"
		data.Description = "Playlist"
	}

	c.HTML(http.StatusOK, "download-confirm.tmpl", gin.H{
		"VideoName":   data.Title,
		"Thumbnail":   data.Thumbnail,
		"Description": data.Description,
		"Platform":    "YouTube",
		"ID":          videoid,
		"Type":        vidtype,
	})

}
