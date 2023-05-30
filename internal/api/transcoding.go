package api

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
)

// Transcode video into HLS chunks. This is done with

func generateHLSManifestFile(videoFilePath string, transcodingFolder string, manifestFilePath string) error {
	if err := os.MkdirAll(transcodingFolder, os.ModePerm); err != nil {
		return err
	}

	// Transcode first chunk
	if err := transcodeChunk(videoFilePath, fmt.Sprintf("%s/chunk000.ts", transcodingFolder), 0, 10*time.Second); err != nil {
		return err
	}

	// Generate initial m3u8 file with first chunk
	if err := generateHLSManifest(transcodingFolder, []string{fmt.Sprintf("%s/chunk000.ts", transcodingFolder)}, manifestFilePath); err != nil {
		log.Printf("Failed to generate HLS manifest file: %v", err)
	}

	return nil
}

func transcodeChunkByIndex(videoFilePath string, transcodingFolder string, chunkIndex int) error {
	chunkPath := fmt.Sprintf("%s/chunk%03d.ts", transcodingFolder, chunkIndex)
	startTime := chunkIndex * int(10*time.Second)
	if err := transcodeChunk(videoFilePath, chunkPath, startTime, 10*time.Second); err != nil {
		return err
	}

	return nil
}

func initialManifest(targetDuration float64) string {
	return fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-TARGETDURATION:%.6f\n#EXT-X-MEDIA-SEQUENCE:0\n", targetDuration)
}

func generateHLSManifest(transcodingFolder string, chunkPaths []string, manifestFilePath string) error {
	// Open manifest file for writing
	manifestFile, err := os.Create(manifestFilePath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	initData := initialManifest(10.0)

	// Write manifest header
	if _, err := manifestFile.WriteString(initData); err != nil {
		return err
	}

	// Write chunk paths to manifest file
	for _, chunkPath := range chunkPaths {
		chunkDuration := 10.0

		// Get chunk web path /api/media/transcoding/video/${uuid}/${chunkName}.ts, the transcodingFolder name is the uuid
		uuid := strings.Split(transcodingFolder, "/")[len(strings.Split(transcodingFolder, "/"))-1]
		apiPath := fmt.Sprintf("/api/media/transcoding/video/%s/%s", uuid, strings.Split(chunkPath, "/")[len(strings.Split(chunkPath, "/"))-1])

		if _, err := manifestFile.WriteString(fmt.Sprintf("#EXTINF:%f,\n%s\n", chunkDuration, apiPath)); err != nil {
			return err
		}
	}

	return nil
}

func transcodeChunk(videoFilePath string, chunkPath string, chunkNumber int, chunkDuration time.Duration) error {
	startTime := time.Duration(chunkNumber) * chunkDuration

	args := []string{
		"-ss", fmt.Sprintf("%02d:%02d:%02d.%03d", int(startTime.Hours()), int(startTime.Minutes())%60, int(startTime.Seconds())%60, startTime.Milliseconds()%1000),
		"-i", videoFilePath,
		"-t", fmt.Sprintf("%v", chunkDuration),
		"-c", "copy",
		"-bsf:v", "h264_mp4toannexb",
		"-f",
		"mpegts",
		"-y",
		"-loglevel", "error",
		chunkPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func ServeHLSManifest(c *gin.Context) {
	uuid := c.Param("uuid")
	manifestFilePath := fmt.Sprintf("./transcoding/%s/manifest.m3u8", uuid)
	ServeHLSManifestWithFilePath(c, manifestFilePath)
}

func ServeHLSManifestWithFilePath(c *gin.Context, filePath string) {
	c.Header("Content-Type", "application/x-mpegURL")
	c.File(filePath)
}

func ServeVideoChunk(c *gin.Context) {
	chunkName := c.Param("chunk_name")
	uuid := c.Param("uuid")
	transcodingFolder := fmt.Sprintf("./transcoding/%s", uuid)

	// Use regular expression to extract chunk index
	re := regexp.MustCompile(`chunk(\d+)\.ts`)
	matches := re.FindStringSubmatch(chunkName)
	if len(matches) < 2 {
		c.JSON(400, gin.H{"error": "Invalid chunk name"})
		return
	}

	chunkIndex, err := strconv.Atoi(matches[1])
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid chunk index"})
		return
	}

	chunkPath := fmt.Sprintf("%s/chunk%03d.ts", transcodingFolder, chunkIndex)

	// Check if the requested chunk exists
	_, err = os.Stat(chunkPath)
	if os.IsNotExist(err) {
		// Extract chunk index from the chunk name
		chunkIndex, _ := strconv.Atoi(chunkName[5:8])

		// Get the video file path from the database
		videoData, err := database.GetVideo(uuid, db)
		if err != nil {
			c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}

		// Transcode the requested chunk
		videoFilePath := videoData.FilePath
		if err := transcodeChunkByIndex(videoFilePath, transcodingFolder, chunkIndex); err != nil {
			c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}

		// Update the manifest file
		manifestFilePath := fmt.Sprintf("%s/manifest.m3u8", transcodingFolder)
		chunkPaths, err := getExistingChunkPaths(transcodingFolder)
		if err != nil {
			c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}
		if err := generateHLSManifest(transcodingFolder, chunkPaths, manifestFilePath); err != nil {
			c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}
	}

	// Serve the chunk
	c.Header("Content-Type", "video/MP2T")
	c.File(chunkPath)
}

// Assuming you have a function to list existing chunks in the transcoding folder
func getExistingChunkPaths(transcodingFolder string) ([]string, error) {
	// use filepath
	files, err := filepath.Glob(fmt.Sprintf("%s/*.ts", transcodingFolder))
	if err != nil {
		return nil, err
	}

	return files, nil
}

func StartTranscoding(videoFilePath string, transcodingFolder string) error {
	// Create transcoding folder if it does not exist
	if err := os.MkdirAll(transcodingFolder, os.ModePerm); err != nil {
		return err
	}
	// Transcode first chunk
	if err := transcodeChunk(videoFilePath, fmt.Sprintf("%s/chunk000.ts", transcodingFolder), 0, 10*time.Second); err != nil {
		return err
	}

	// Generate initial manifest file
	if err := generateHLSManifestFile(videoFilePath, transcodingFolder, fmt.Sprintf("%s/manifest.m3u8", transcodingFolder)); err != nil {
		return err
	}

	// Transcode remaining chunks and append to manifest file
	lastChunkNumber := 0
	callback := func(chunkPath string) {
		lastChunkNumber++
		manifestFilePath := fmt.Sprintf("%s/manifest.m3u8", transcodingFolder)

		// Append new chunk to manifest file
		if err := generateHLSManifest(transcodingFolder, []string{chunkPath}, manifestFilePath); err != nil {
			log.Printf("Failed to append new chunk to manifest file: %v", err)
		}

		// Update manifest file
		if err := generateHLSManifest(transcodingFolder, []string{chunkPath}, manifestFilePath); err != nil {
			log.Printf("Failed to update manifest file: %v", err)
		}
	}

	// Transcode remaining chunks
	if err := transcodeVideoIntoChunks(videoFilePath, transcodingFolder, callback); err != nil {
		return err
	}

	// Add the #EXT-X-ENDLIST tag when all chunks have been generated
	manifestFilePath := fmt.Sprintf("%s/manifest.m3u8", transcodingFolder)
	file, err := os.OpenFile(manifestFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString("#EXT-X-ENDLIST\n"); err != nil {
		return err
	}

	return nil
}

func streamTranscodedVideo(c *gin.Context) {
	videoID := c.Param("video_id")
	userid, err := jwttoken.GetUserFromToken(c)
	if err != nil {
		c.JSON(401, gin.H{"ret": 401, "err": "Unauthorized"})
		return
	}
	uuid := fmt.Sprintf("%v-%v", userid, videoID)
	transcodingFolder := fmt.Sprintf("./transcoding/%s", uuid)
	manifestFilePath := fmt.Sprintf("%s/manifest.m3u8", transcodingFolder)

	// get video file path
	videoData, err := database.GetVideo(videoID, db)
	if err != nil {
		c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
		return
	}

	var videoFilePath string

	// Check if twitch or youtube
	if videoData.VideoType == "Twitch" {
		videoFilePath = fmt.Sprintf("%v/%v", settings.BaseTwitchPath, videoData.FilePath)
	} else {
		// return error if not twitch
		c.JSON(503, gin.H{"ret": 503, "err": "Video is not a stream"})
		return
	}

	// Start transcoding in a separate goroutine
	go func() {
		if err := StartTranscoding(videoFilePath, transcodingFolder); err != nil {
			fmt.Printf("Failed to start transcoding: %v", err)
		}
	}()

	// Wait for manifest file to be generated
	for {
		if _, err := os.Stat(manifestFilePath); !os.IsNotExist(err) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Serve manifest file
	ServeHLSManifestWithFilePath(c, manifestFilePath)
}

func transcodeVideoIntoChunks(videoFilePath string, transcodingFolder string, callback func(string)) error {
	// Get video duration
	duration, err := getDuration(videoFilePath)
	if err != nil {
		return err
	}

	// Calculate chunk duration
	chunkDuration := 10 * time.Second

	// Transcode video into chunks
	for i := 0; i < int(duration.Seconds()/chunkDuration.Seconds()); i++ {
		chunkPath := fmt.Sprintf("%s/chunk%03d.ts", transcodingFolder, i)
		startTime := i * int(chunkDuration.Seconds())
		if err := transcodeChunk(videoFilePath, chunkPath, startTime, chunkDuration); err != nil {
			return err
		}
		callback(chunkPath)
	}

	return nil
}

func getDuration(filePath string) (time.Duration, error) {
	probeCmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	var out bytes.Buffer
	probeCmd.Stdout = &out

	if err := probeCmd.Run(); err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(out.String())
	// check if string is n/a, if so return 0
	if durationStr == "N/A" {
		return 0, nil
	}
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(duration) * time.Second, nil
}
