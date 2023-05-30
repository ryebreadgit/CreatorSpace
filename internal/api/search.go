package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
)

func apiSearchCreators(c *gin.Context) {
	var creators []database.Creator
	// if limit is not specified, default to 10
	limit := 10
	if c.Query("limit") != "" {
		limit = c.MustGet("limit").(int)
	}
	err := db.Limit(limit).Where("name LIKE ?", c.Query("q")+"%").Find(&creators).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"res":  200,
		"data": creators,
	})
}

func apiSearchVideos(c *gin.Context) {
	var videos []database.Video
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{
			"res": 400,
			"err": "no query specified",
		})
		return
	}
	// if limit is not specified, default to 10
	limit := 10
	if c.Query("limit") != "" {
		limit, err := strconv.Atoi(c.Query("limit"))
		if err != nil {
			c.JSON(400, gin.H{
				"res": 400,
				"err": "invalid limit",
			})
			return
		}

		// if limit is greater than 50, set it to 50
		if limit > 50 {
			limit = 50
		}
	}

	// enable searching for the middle of a title
	query = strings.ToLower(query)

	err := db.Limit(limit).Where("LOWER(title) LIKE ?", "%"+query+"%").Order("likes DESC").Find(&videos).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"res":  200,
		"data": videos,
	})
}
