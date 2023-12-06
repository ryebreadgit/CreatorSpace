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

	query = strings.Trim(strings.ToLower(query), " ")
	// Replace special characters with spaces
	// Literal search
	literalSearch := db.Where("LOWER(title) LIKE ?", "%"+query+"%")
	if filter != "" {
		literalSearch = literalSearch.Where("channel_id = ?", filter)
	}

	err := literalSearch.Limit(limit).Find(&videos).Error
	if err != nil {
		c.JSON(500, gin.H{
			"res": 500,
			"err": err.Error(),
		})
		return
	}

	// Sort videos by relevance
	sort.Slice(videos, func(i, j int) bool {
		return fuzzy.LevenshteinDistance(query, strings.ToLower(videos[i].Title)) < fuzzy.LevenshteinDistance(query, strings.ToLower(videos[j].Title))
	})

	var videoMap = make(map[string]bool)

	for _, video := range videos {
		videoMap[video.VideoID] = true
	}

	// Fuzzy search

	// Get all video titles lowercase
	var titles []string

	dbquery := db.Model(&database.Video{})
	if filter != "" {
		dbquery = dbquery.Where("channel_id = ?", filter)
	}

	err = dbquery.Pluck("LOWER(title)", &titles).Error
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

	// Only keep top 100 matches
	if len(matches) > 100 {
		matches = matches[:100]
	}

	// Append matches up to the limit
	for _, match := range matches {
		if len(videos) >= limit+20 { // Arbitrary number to so that we can sort the last half of videos by relevance. This mixes the literal search and fuzzy search results better.
			break
		}

		if match.Distance > threshold {
			continue
		}

		var vid database.Video
		err := db.Where("LOWER(title) = ?", match.Target).First(&vid).Error
		if err != nil {
			continue
		}

		// Check if video is already in videos
		if _, ok := videoMap[vid.VideoID]; ok {
			continue
		}
		videoMap[vid.VideoID] = true
		videos = append(videos, vid)
	}

	// Sort only the last half of videos by relevance. Keep first half's sorting order.
	sort.Slice(videos[len(videos)/2:], func(i, j int) bool {
		return fuzzy.LevenshteinDistance(query, strings.ToLower(videos[i].Title)) < fuzzy.LevenshteinDistance(query, strings.ToLower(videos[j].Title))
	})

	// Truncate to limit
	if len(videos) > limit {
		videos = videos[:limit]
	}

	// Append creators to the bottom

	var creators []database.Creator

	err = db.Limit(limit).Where("LOWER(name) LIKE ?", "%"+query+"%").Find(&creators).Error
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
