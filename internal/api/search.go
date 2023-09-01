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
	filter := c.Query("filter")
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
		var err error
		limit, err = strconv.Atoi(c.Query("limit"))
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

	// Get all video titles lowercase
	var titles []string

	dbquery := db.Model(&database.Video{})
	if filter != "" {
		dbquery = dbquery.Where("channel_id = ?", filter)
	}

	err := dbquery.Pluck("LOWER(title)", &titles).Error
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

		var vid database.Video
		err := db.Where("LOWER(title) = ?", match.Target).First(&vid).Error
		if err != nil {
			continue
		}

		videos = append(videos, vid)
	}

	// Remove duplicates from videos
	for i := 0; i < len(videos); i++ {
		for j := i + 1; j < len(videos); j++ {
			if videos[i].VideoID == videos[j].VideoID {
				videos = append(videos[:j], videos[j+1:]...)
				j--
			}
		}
	}

	// Truncate to limit
	if len(videos) > limit {
		videos = videos[:limit]
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
