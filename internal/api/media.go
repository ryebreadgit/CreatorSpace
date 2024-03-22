package api

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	log "github.com/sirupsen/logrus"
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func apiMedia(c *gin.Context) {

	video := c.Param("video_id")
	vidData, err := database.GetVideo(video, db)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	var basePath string

	// Check if twitch or youtube
	if vidData.VideoType == "Twitch" {
		basePath = settings.BaseTwitchPath
	} else if vidData.VideoType == "Twitter" {
		basePath = settings.BaseTwitterPath
	} else {
		basePath = settings.BaseYouTubePath
	}

	filePath := fmt.Sprintf("%v/%v", basePath, vidData.FilePath)

	filePath = strings.ReplaceAll(filePath, "//", "/")

	if _, err := os.Stat(filePath); err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": fmt.Sprintf("file not found: %v", filePath)})
		return
	}

	mimeType := vidData.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(filePath))
		if mimeType == "" {
			mimeType = "video/mp4"
		}
	}

	// Check if Twitch
	if vidData.VideoType == "Twitch" {
		// redirect to /api/watch/transcoding/:video_id
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/api/watch/%v/manifest.m3u8", video))
		c.Abort()
		return
	} else {
		if err := StreamDirect(c, filePath, mimeType); err != nil {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}
		return
	}
}

func readImageFromDisk(filepath string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Read the file
	bytes := make([]byte, stat.Size())
	_, err = file.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func convertThumbnail(filepath string) ([]byte, error) {
	// Use ffmpeg to convert thumbnails to smaller size. We use ffmpeg to limit dependancies as this is already required for transcoding.

	cmd := exec.Command("ffmpeg", "-i", filepath, "-vf", "scale=320:-1", "-vframes", "1", "-f", "image2pipe", "-vcodec", "png", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	errBytes, err := io.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, errBytes)
	}

	return bytes, nil
}

// getImageData reads the image data from Redis if available, otherwise reads from the file system and caches it in Redis
// Returns []byte data, error, and a bool indicating if the data was read from Redis
func getImageData(filePath string, compress bool) ([]byte, error, bool) {
	// Try to get the image data from Redis
	var imageData []byte
	if rdb == nil {
		imageData, err := readImageFromDisk(filePath)
		if err != nil {
			return nil, err, false
		}
		return imageData, nil, false
	}

	redisKey := filePath
	if compress {
		redisKey = fmt.Sprintf("%v:compress", filePath)
	}

	imageData, err := rdb.Get(ctx, redisKey).Bytes()

	// If data is not found in Redis
	if err == redis.Nil {
		keytime := time.Hour * 24
		// Read the image from the file system
		if compress {
			imageData, err = convertThumbnail(filePath)
			keytime = time.Hour * 48 // Keep compressed thumbnails in cache for 2 days
		} else {
			imageData, err = readImageFromDisk(filePath)
		}
		if err != nil {
			log.Warnf("Error converting thumbnail '%v': %v", filePath, err.Error())
			imageData, err = readImageFromDisk(filePath)
			if err != nil {
				return nil, err, false
			}
		}

		// Cache the image data in Redis with an expiration time of 24 hours
		err = rdb.Set(ctx, redisKey, imageData, keytime).Err()
		if err != nil {
			return nil, err, false
		}

	} else if err != nil {
		// Read the image from the file system directly
		if compress {
			imageData, err = convertThumbnail(filePath)
		} else {
			imageData, err = readImageFromDisk(filePath)
		}
		if err != nil {
			return nil, err, false
		}
	}
	return imageData, nil, true
}

func getVideoThumbnailPath(videoID string) (string, error) {
	vidData, err := database.GetVideo(videoID, db)
	if err != nil {
		return "", err
	}

	if vidData.VideoID == "" {
		return "", errors.New("video not found")
	}

	var basePath string
	if vidData.VideoType == "Twitch" {
		basePath = settings.BaseTwitchPath
	} else if vidData.VideoType == "Twitter" {
		basePath = settings.BaseTwitterPath
	} else {
		basePath = settings.BaseYouTubePath
	}
	thumbPath := fmt.Sprintf("%s/%s", basePath, vidData.ThumbnailPath)
	// replace double slashes
	thumbPath = filepath.Clean(thumbPath)
	redirect := false

	dat, err := os.Stat(thumbPath)
	if err != nil || dat == nil {
		redirect = true
		if !os.IsNotExist(err) { // If the error is not that the file does not exist, log the error
			log.Errorf("Unable to get thumbnail %v due to error: %v", thumbPath, err)
		}
	}

	// Check if file is a directory
	if err == nil && dat.IsDir() {
		// redirect to default thumbnail
		redirect = true
	}

	if redirect {
		if vidData.VideoType == "Twitch" {
			return "/assets/img/defaults/posts/twitch_post.svg", nil
		} else if vidData.VideoType == "Twitter" {
			return "/assets/img/defaults/posts/twitch_post.svg", nil
		} else {
			return "/assets/img/defaults/posts/youtube_post.svg", nil
		}
	}

	return thumbPath, nil
}

func getThumbnail(c *gin.Context, thumbPath string, compress bool) ([]byte, string, error, bool) {

	// If starts with /assets/ then redirect to that path
	if strings.HasPrefix(thumbPath, "/assets/") {
		c.Redirect(http.StatusMovedPermanently, thumbPath)
		c.Abort()
		return nil, "redirect", nil, false
	}

	// Read thumbnail from path
	thumbnail, err, red := getImageData(thumbPath, compress)
	if err != nil {
		log.Errorf("Error getting thumbnail '%v': %v", thumbPath, err)
		return nil, "", err, false
	}

	// get mimetype from extension

	mimetype := mime.TypeByExtension(filepath.Ext(thumbPath))

	return thumbnail, mimetype, nil, red
}

func getCreatorThumbnailPath(creatorID string) (string, error) {
	creatorData, err := database.GetCreator(creatorID, db)
	if err != nil {
		creatorData = database.Creator{}
	}

	var basePath string
	if creatorData.Platform == "Twitch" {
		basePath = settings.BaseTwitchPath
	} else if creatorData.Platform == "Twitter" {
		basePath = settings.BaseTwitterPath
	} else {
		basePath = settings.BaseYouTubePath
	}
	thumbPath := fmt.Sprintf("%s/%s", basePath, creatorData.ThumbnailPath)
	thumbPath, err = general.SanitizeFilePath(thumbPath)
	if err != nil {
		return "", err
	}

	// Get absolute path
	thumbPath, err = filepath.Abs(thumbPath)
	if err != nil {
		return "", err
	}

	// check if thumbnail exists or is empty
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) || creatorData.ThumbnailPath == "" {
		// redirect to default thumbnail
		if creatorData.Platform == "Twitch" {
			return "/assets/img/defaults/avatars/twitch_avatar.svg", nil
		} else if creatorData.Platform == "Twitter" {
			return "/assets/img/defaults/avatars/twitter_avatar.svg", nil
		} else {
			return "/assets/img/defaults/avatars/youtube_avatar.svg", nil
		}
	}

	return thumbPath, nil
}

func apiThumbnail(c *gin.Context) {

	var err error
	var compress bool = false
	var thumbPath string

	if c.Params.ByName("video_id") != "" {
		vidId := c.Param("video_id")
		thumbPath, err = getVideoThumbnailPath(vidId)
		if err != nil {
			// Default to the YouTube video thumbnail
			log.Errorf("Error getting video thumbnail '%v': %v", vidId, err)
			thumbPath = "/assets/img/defaults/posts/youtube_post.svg"
		}
	} else if c.Params.ByName("creator") != "" {
		creator_id := c.Param("creator")
		thumbPath, err = getCreatorThumbnailPath(creator_id)
		if err != nil {
			// Default to the YouTube creator thumbnail
			log.Errorf("Error getting creator thumbnail '%v': %v", creator_id, err)
			thumbPath = "/assets/img/defaults/avatars/youtube_avatar.svg"
		}
	} else {
		c.AbortWithStatusJSON(400, gin.H{"ret": 400, "err": "invalid request"})
		return
	}

	if c.Query("compress") == "true" {
		compress = true
	}

	// get if modified since header
	if match := c.Request.Header.Get("If-None-Match"); match != "" {
		// set etag from redis if set
		var e string

		// get etag from file
		f, err := os.Open(thumbPath)
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}
		defer f.Close()
		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}
		e = fmt.Sprintf("%x", h.Sum(nil))
		if strings.Contains(match, e) {
			c.AbortWithStatus(http.StatusNotModified)
			return
		}
	}

	thumbnail, mimetype, err, red := getThumbnail(c, thumbPath, compress)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	if mimetype == "redirect" {
		c.Abort()
		return
	}

	if red {
		c.Header("X-Cache", "HIT")
	} else {
		c.Header("X-Cache", "MISS")
	}

	// set header to cache for 14 days
	c.Header("Cache-Control", "public, max-age=1209600")
	// set etag header
	c.Header("ETag", fmt.Sprintf("%x", md5.Sum(thumbnail)))
	// set last modified header
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	c.Data(200, mimetype, thumbnail)
}

func apiCreatorBanner(c *gin.Context) {

	creatorid := c.Param("creator")
	creatorData, err := database.GetCreator(creatorid, db)
	if err != nil {
		// return with default banner
		c.Redirect(http.StatusTemporaryRedirect, "/assets/img/defaults/banners/youtube_banner.svg")
		c.Abort()
		return
	}

	var basePath string
	if creatorData.Platform == "Twitch" {
		basePath = settings.BaseTwitchPath
	} else if creatorData.Platform == "Twitter" {
		basePath = settings.BaseTwitterPath
	} else {
		basePath = settings.BaseYouTubePath
	}

	bannerPath := fmt.Sprintf("%v/%v", basePath, creatorData.BannerPath)

	// check if banner path exists or is a directory
	if _, err := os.Stat(bannerPath); os.IsNotExist(err) || creatorData.BannerPath == "" {
		// redirect to default banner
		if creatorData.Platform == "Twitch" {
			c.Redirect(http.StatusTemporaryRedirect, "/assets/img/defaults/banners/twitch_banner.svg")
			c.Abort()
			return
		} else if creatorData.Platform == "Twitter" {
			c.Redirect(http.StatusTemporaryRedirect, "/assets/img/defaults/banners/twitter_banner.svg")
			c.Abort()
			return
		} else {
			c.Redirect(http.StatusTemporaryRedirect, "/assets/img/defaults/banners/youtube_banner.svg")
			c.Abort()
			return
		}
	}

	// open banner file
	banner, err := os.ReadFile(bannerPath)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	// get mimetype from extension
	mimetype := mime.TypeByExtension(filepath.Ext(creatorData.BannerPath))

	// Set header to cache for 3 days
	c.Header("Cache-Control", "public, max-age=259200")
	// set etag header
	c.Header("ETag", fmt.Sprintf("%x", md5.Sum(banner)))
	// set last modified header
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	c.Data(http.StatusOK, mimetype, banner)
}

func GetRecommendations(videoID string, watchedVids []string) ([]database.Video, error) {
	var currentVideo database.Video
	err := db.First(&currentVideo, "video_id = ?", videoID).Error
	if err != nil {
		return nil, err
	}

	var currentCategories, currentTags []string
	json.Unmarshal([]byte(currentVideo.Categories), &currentCategories)
	json.Unmarshal([]byte(currentVideo.Tags), &currentTags)

	var recommendations []database.Video

	// Fetch videos with the same categories or from the same creator
	query := db.Select("title", "description", "video_id", "likes", "views", "channel_title", "channel_id", "published_at", "length", "video_type", "availability", "categories", "tags")
	query = query.Where("video_id != ?", videoID)
	for _, category := range currentCategories {
		query = query.Or("categories LIKE ?", "%"+category+"%")
	}
	query = query.Or("channel_id = ?", currentVideo.ChannelID)
	// set a limit while sorting to allow the most efficient query
	query = query.Order("published_at DESC")
	query = query.Order("views DESC")
	query = query.Order("likes DESC")
	query = query.Limit(3500)
	err = query.Find(&recommendations).Error
	if err != nil {
		return nil, err
	}

	type videoScore struct {
		video database.Video
		score float64
	}

	var scoredRecommendations []videoScore

	// Store the common words in a map
	commonWordsMap := make(map[string]bool)
	commonWords := []string{
		"a", "an", "the", "and", "or", "but", "for", "nor", "so", "yet", "as", "at", "by", "in", "from", "into", "of", "on", "we", "is", "you", "he", "she", "it", "they", "me", "him", "her", "us", "them", "my", "your", "our", "we", "best",
		"to", "with", "that", "this", "these", "those", "his", "hers", "its", "theirs", "mine", "yours", "ours", "theirs", "i", "am", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does", "did", "done", "will", "would", "shall", "should", "may", "might", "must", "can", "could",
		"about", "above", "across", "after", "against", "along", "among", "around", "at", "before", "behind", "below", "beneath", "beside", "between", "beyond", "down", "during", "except", "for", "from", "inside", "into", "near", "off", "on", "onto", "outside", "over", "past", "since", "through", "throughout", "till", "to", "toward", "under", "underneath", "until", "up", "upon", "with", "within", "without",
		// Clickbait terms will be counted as common
		"shocking", "you won't believe", "amazing", "secret", "revealed", "incredible", "ind-blowing", "unbelievable", "must-see", "jaw-dropping", "mind-boggling", "mind-bending", "epic", "awesome", "unreal", "insane", "crazy", "wild", "ridiculous", "hilarious", "funny", "lol", "wtf", "omg", "wow", "fail", "fails", "huge", "free", "crack", "pirate", "piracy",
	}
	for _, word := range commonWords {
		commonWordsMap[word] = true
	}

	// Store the categories and tags of the current video in a map
	currentCategoriesMap := make(map[string]bool)
	currentTagsMap := make(map[string]bool)
	for _, category := range currentCategories {
		currentCategoriesMap[category] = true
	}
	for _, tag := range currentTags {
		currentTagsMap[tag] = true
	}

	// Use a map to store the watched videos
	watchedVidsMap := make(map[string]bool)
	for _, vid := range watchedVids {
		watchedVidsMap[vid] = true
	}

	// Store the words in the current video's title in a map
	vidTitleMap := make(map[string]bool)
	for _, word := range strings.Split(currentVideo.Title, " ") {
		word := strings.ToLower(word)
		if commonWordsMap[word] {
			continue
		}
		vidTitleMap[word] = true
	}

	// Store the words in the current video's description in a map
	vidDescMap := make(map[string]bool)
	currentVideo.Description = strings.ReplaceAll(currentVideo.Description, "\n", " ")
	for _, word := range strings.Split(currentVideo.Description, " ") {
		word := strings.ToLower(word)
		if commonWordsMap[word] {
			continue
		}
		if len(word) <= 8 && strings.Contains(word, ":") { // Ignores timestamps
			_, err := strconv.Atoi(strings.ReplaceAll(word, ":", ""))
			if err == nil {
				continue
			}
		}
		vidDescMap[word] = true
	}

	curViewc, err := strconv.Atoi(currentVideo.Views)
	if err != nil {
		curViewc = 0
	}

	for _, rec := range recommendations {
		var recCategories, recTags []string
		json.Unmarshal([]byte(rec.Categories), &recCategories)
		json.Unmarshal([]byte(rec.Tags), &recTags)

		score := 0.0

		// Check if the recommendation has any of the current video's categories or tags
		for _, cat := range recCategories {
			if currentCategoriesMap[cat] {
				score += 3
			}
		}
		for _, tag := range recTags {
			if currentTagsMap[tag] {
				score += 3
			}
		}

		// If the video is from the same creator, give it a little boost
		if rec.ChannelID == currentVideo.ChannelID {
			score += 2
		}

		// If the video type is the same, give it a little boost
		if rec.VideoType == currentVideo.VideoType {
			score += 4
		}

		// If the video is not available, give it a boost
		if rec.Availability != "available" && currentVideo.Availability != "available" {
			score += 3
		}

		var recTitleMap = make(map[string]bool)
		// split title into words
		for _, word := range strings.Split(rec.Title, " ") {
			recTitleMap[strings.ToLower(word)] = true
		}

		// If the video title has similar words to the current video, add 1 point
		for word := range vidTitleMap {
			if recTitleMap[word] {
				score += 3
			}
		}

		// Do the same for description

		// replace return characters with spaces
		rec.Description = strings.ReplaceAll(rec.Description, "\n", " ")

		var recDescWordsMap = make(map[string]bool)
		// split description into words
		for _, word := range strings.Split(rec.Description, " ") {
			recDescWordsMap[strings.ToLower(word)] = true
		}

		for word := range vidDescMap {
			if recDescWordsMap[word] {
				score += 1
			}
		}

		// Give higher score for videos with more rec.Views
		viewc, err := strconv.Atoi(rec.Views)
		if err != nil {
			viewc = 0
		}

		if viewc > 0 && curViewc > 0 {
			// Give videos with similar views a little boost
			if viewc > curViewc {
				score += math.Log10(float64(viewc)) / 2
			} else {
				score += math.Log10(float64(curViewc)) / 2
			}
		}

		// Give a lower score for older videos
		publishedAt, err := time.Parse("2006-01-02T15:04:05Z", rec.PublishedAt)
		if err != nil {
			// use rec.Epoch
			publishedAt = time.Unix(rec.Epoch, 0)
		}

		score -= math.Log10(float64(time.Since(publishedAt).Hours()))

		// If the video id is in the watched list, minus points
		if watchedVidsMap[rec.VideoID] {
			score -= 12.5
		}

		if score > 0 && rec.VideoID != currentVideo.VideoID {
			scoredRecommendations = append(scoredRecommendations, videoScore{video: rec, score: score})
		}
	}

	// Sort scoredRecommendations by score
	sort.Slice(scoredRecommendations, func(i, j int) bool {
		return scoredRecommendations[i].score > scoredRecommendations[j].score
	})

	// Drop all but top 50
	if len(scoredRecommendations) > 50 {
		scoredRecommendations = scoredRecommendations[:50]
	}

	// Get 10 random reccs and sort by score
	var topReccs []videoScore
	// Shuffle scoredRecommendations
	r.Shuffle(len(scoredRecommendations), func(i, j int) {
		scoredRecommendations[i], scoredRecommendations[j] = scoredRecommendations[j], scoredRecommendations[i]
	})

	topNum := 10

	// if there are less than 10 recommendations, set topNum to the length of scoredRecommendations
	if len(scoredRecommendations) < 10 {
		topNum = len(scoredRecommendations)
	}

	for i := 0; i < topNum; i++ {
		topReccs = append(topReccs, scoredRecommendations[i])
	}

	// Sort topReccs by score
	sort.Slice(topReccs, func(i, j int) bool {
		return topReccs[i].score > topReccs[j].score
	})

	var topRecommendations []database.Video
	for _, recc := range topReccs {
		topRecommendations = append(topRecommendations, recc.video)
	}

	return topRecommendations, nil
}
