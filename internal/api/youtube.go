package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
)

func apiCreators(c *gin.Context) (string, error) {

	channelid := c.Param("channelid")
	db, err := database.GetDatabase()
	if err != nil {
		return "", err
	}

	if channelid == "" {
		return "", errors.New("no channelid specified")
	}

	channel, err := database.GetCreator(channelid, db)
	if err != nil {
		return "", err
	}

	emptchnl := database.Creator{}

	if channel == emptchnl {
		return "", errors.New("channel not found")
	}

	// convert vidData to json string
	ret, err := json.MarshalIndent(channel, "", "  ")
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func apiVideoMetadata(c *gin.Context) (string, error) {

	videoid := c.Param("video_id")
	vidData, err := database.GetVideo(videoid, db)
	if err != nil {
		return "", err
	}

	// convert vidData to json string
	ret, err := json.MarshalIndent(vidData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func apiVideoSponsorblock(c *gin.Context) (string, error) {

	videoid := c.Param("video_id")
	vidspon, err := database.GetVideoSponsorBlock(videoid, db)
	if err != nil {
		return "", err
	}

	if vidspon == nil {
		return "", errors.New("video not found")
	}

	metadata, err := os.ReadFile(fmt.Sprintf(vidspon[0].FilePath)) // TODO fix this

	if err != nil {
		return "", err
	}

	return string(metadata), nil
}

func apiVideoComments(c *gin.Context) (string, error) {

	videoid := c.Param("video_id")
	comms, err := database.GetVideoComments(videoid, db)
	if err != nil {
		return "", err
	}

	var ret string
	for _, comm := range comms {
		ret += comm.MetadataJson + "\n"
	}
	return ret, nil
}

func apiAllVideoComments(c *gin.Context) {

	video := c.Param("video_id")
	// get video from database
	vidData, err := database.GetVideo(video, db)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	commentPath := fmt.Sprintf("%v/%v", settings.BaseYouTubePath, vidData.CommentsPath)

	// read and send file in 5mb chunks to prevent loading entire file into memory
	file, err := os.Open(commentPath)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	// get file info
	fileInfo, err := file.Stat()
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}
	fileSize := fileInfo.Size()

	// send file in 5mb chunks, use range header to determine which chunk to send
	// prevent loading entire file into memory

	const maxChunkSize int64 = 5 * 1024 * 1024

	rangeHeader := c.GetHeader("Range")
	if rangeHeader == "" {
		// Stream file in 5mb chunks
		c.DataFromReader(http.StatusOK, fileSize, "application/jsonl+json", file, map[string]string{
			"Content-Range": fmt.Sprintf("bytes %v-%v/%v", 0, maxChunkSize, fileSize),
		})
		return
	}

	start, end, err := parseRangeHeader(rangeHeader, fileSize, maxChunkSize)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	if start >= fileSize || end >= fileSize {
		c.AbortWithStatusJSON(416, gin.H{"ret": 416, "err": "range out of bounds"})
		return
	}

	// check if data is too large to send in one chunk
	if end-start > 5*1024*1024 {
		end = start + 5*1024*1024
	}

	// seek to start of chunk
	_, err = file.Seek(start, 0)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	// send chunk
	c.DataFromReader(http.StatusPartialContent, fileSize, "application/jsonl+json", file, map[string]string{
		"Content-Range": fmt.Sprintf("bytes %v-%v/%v", start, end, fileSize),
	})
}

func getVideoSubtitles(c *gin.Context) {

	videoid := c.Param("video_id")
	lang := c.Param("lang")
	vidsubs, err := database.GetVideoSubtitles(videoid, db)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	if vidsubs == nil {
		c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": "video not found"})
		return
	}

	for _, sub := range vidsubs {
		if sub.Language == lang {
			metadata, err := os.ReadFile(filepath.Join(settings.BaseYouTubePath, sub.FilePath))

			if err != nil {
				c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
				return
			}

			c.String(http.StatusOK, string(metadata))
			return
		}
	}

	c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": "subtitle not found"})
}
