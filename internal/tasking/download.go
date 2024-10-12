package tasking

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// download videos in queue and update database
func downloadYouTubeVideos(settings *database.Settings, db *gorm.DB) error {
	// get all videos in queue
	videos, err := database.GetDownloadQueue(db.Where("source = ?", "youtube").Where("video_type = ?", "video").Or("video_type = ?", "short").Where("approved = ?", true).Order("created_at desc"))
	if err != nil {
		return err
	}
	dlerrs := []error{}
	// loop over all videos and download them
	for _, video := range videos {
		// skip any non-youtube videos. Type must be video
		if (video.VideoType != "video" && video.VideoType != "short") || !video.Approved {
			continue
		}
		// if the video is already downloaded, skip it and remove from queue
		// check the db for the video
		_, err := database.GetVideo(video.VideoID, db)
		if err == nil {
			// video is already downloaded, remove from queue
			err = database.RemoveFromDownloadQueue(video.VideoID, video.VideoType, db)
			if err != nil {
				log.Errorf("Error removing video id '%v' from download queue: %v", video.VideoID, err)
				dlerrs = append(dlerrs, err)
				continue
			}
			continue
		}

		// download the video using downloadYouTubeVideo
		vidUrl := "https://www.youtube.com/watch?v=" + video.VideoID
		outputDir := filepath.Dir(video.DownloadPath)
		config := "./config/youtube-video-default.conf"

		filePath, err := downloadYouTubeVideo(vidUrl, outputDir, video.VideoID, config)
		if err != nil {
			_ = database.DeleteVideo(video.VideoID, db) // Remove video from Videos table if it exists
			if strings.Contains(err.Error(), "rate limited") || strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "sign in to confirm you’re not a bot") {
				log.Warnf("Rate limited, skipping video id '%v' and sleeping for 5 minutes", video.VideoID)
				time.Sleep(5 * time.Minute)
				continue
			}
			dlerrs = append(dlerrs, err)
			log.Errorf("Error downloading video id '%v': %v", video.VideoID, err)
			continue
		}

		filePath = strings.ReplaceAll(filePath, settings.BaseYouTubePath, "")

		baseName := strings.ReplaceAll(filepath.Base(filePath), filepath.Ext(filePath), "")

		thumbnailPath := fmt.Sprintf("%v/../metadata/thumbnails/%v.jpg", outputDir, baseName)
		thumbnailPath = strings.ReplaceAll(thumbnailPath, settings.BaseYouTubePath, "")
		thumbnailPath = filepath.Clean(thumbnailPath)

		metadataPath := fmt.Sprintf("%v/../metadata/metadata/%v.json", outputDir, baseName)
		metadataPath = strings.ReplaceAll(metadataPath, settings.BaseYouTubePath, "")
		metadataPath = filepath.Clean(metadataPath)

		vidData := database.Video{
			VideoID:       video.VideoID,
			FilePath:      filePath,
			ThumbnailPath: thumbnailPath,
			MetadataPath:  metadataPath,
			VideoType:     video.VideoType,
		}

		// Add the video to the database
		err = database.InsertVideo(vidData, db)
		if err != nil {
			log.Errorf("Error inserting video id '%v' into database: %v", video.VideoID, err)
			dlerrs = append(dlerrs, err)
			continue
		}

		// update video metadata
		err = updateVideoMetadata(video.VideoID)
		if err != nil {
			_ = database.DeleteVideo(video.VideoID, db) // Remove video from Videos table if it exists
			if strings.Contains(err.Error(), "rate limited") || strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "sign in to confirm you’re not a bot") {
				log.Warnf("Rate limited, skipping video id '%v' and sleeping for 5 minutes", video.VideoID)
				time.Sleep(5 * time.Minute)
				continue
			}
			log.Errorf("Error updating video metadata for video id '%v': %v", video.VideoID, err)
			dlerrs = append(dlerrs, err)
			continue
		}

		log.Info("Successfully downloaded video: ", video.VideoID)
	}
	if len(dlerrs) > 0 {
		return fmt.Errorf("errors downloading videos: %v", dlerrs)
	}
	return nil
}
