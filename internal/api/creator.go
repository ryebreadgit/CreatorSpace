package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
)

func apiGetCreator(c *gin.Context) {
	// Get the creator id from the url
	creatorID := c.Param("creator_id")

	// Get the creator from the database
	creator, err := database.GetCreator(creatorID, db)
	if err != nil {
		c.JSON(500, gin.H{"ret": 500, "err": err.Error()})
		return
	}

	// Remove info
	creator.FilePath = ""

	// Return the creator
	c.JSON(200, gin.H{"ret": 200, "creator": creator})
}

func apiGetCreatorVideos(c *gin.Context) {
	// Get the creator id from the url
	creatorID := c.Param("creator_id")
	if creatorID == "" {
		c.JSON(400, gin.H{"ret": 400, "err": "No creator id provided"})
		return
	}
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

	// Get videos from the database 20 at a time
	videos, err := database.GetAllVideos(db.Where("channel_id = ?", creatorID).Order("published_at desc").Limit(20).Offset((pageInt - 1) * 20))
	if err != nil {
		c.JSON(500, gin.H{"ret": 500, "err": err.Error()})
		return
	}

	if len(videos) == 0 {
		c.JSON(404, gin.H{"ret": 404, "err": "No videos found"})
		return
	}

	for i := range videos {
		videos[i].CommentsPath = ""
		videos[i].FilePath = ""
		videos[i].ThumbnailPath = ""
	}

	// Return the videos
	c.JSON(200, gin.H{"ret": 200, "data": videos})

}
