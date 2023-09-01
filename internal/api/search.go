package api

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/fuzzysearch/fuzzy"
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

	query = strings.ToLower(query)

	err := db.Limit(limit).Where("LOWER(title) LIKE ?", "%"+query+"%").Order("views DESC").Find(&videos).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}

	curTitles := make(map[string]bool)

	for _, vid := range videos {
		curTitles[vid.Title] = true
	}

	// Get all video titles lowercase
	var titles []string

	err = db.Model(&database.Video{}).Pluck("LOWER(title)", &titles).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}

	// Fuzzy search titles
	matches := fuzzy.RankFind(strings.ToLower(query), titles)
	sort.Sort(matches)

	const threshold = 50

	// Append matches up to the limit
	for i, match := range matches {
		if len(videos) >= limit || i >= limit || match.Distance > threshold {
			break
		}
		if curTitles[match.Target] {
			continue
		}

		var vid database.Video
		err := db.Where("LOWER(title) = ?", match.Target).First(&vid).Error
		if err != nil {
			continue
		}

		videos = append(videos, vid)
	}

	// Append creators to the bottom

	var creators []database.Creator

	err = db.Limit(5).Where("LOWER(name) LIKE ?", "%"+query+"%").Find(&creators).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}

	for _, creator := range creators {
		var creatorVideo database.Video
		creatorVideo.VideoType = "channel"
		creatorVideo.VideoID = creator.ChannelID
		creatorVideo.Title = creator.Name

		videos = append(videos, creatorVideo)
	}

	c.JSON(200, gin.H{
		"res":  200,
		"data": videos,
	})
}
