package tasking

import (
	"fmt"
	"path/filepath"
	"strings"

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
				return err
			}
			continue
		}

		// download the video using downloadYouTubeVideo
		vidUrl := "https://www.youtube.com/watch?v=" + video.VideoID
		outputDir := filepath.Dir(video.DownloadPath)
		config := "./config/youtube-video-default.conf"

		filePath, err := downloadYouTubeVideo(vidUrl, outputDir, video.VideoID, config)
		if err != nil {
			return err
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
			log.Error("Error adding video to database: ", err)
			return err
		}

		// update video metadata
		err = updateVideoMetadata(video.VideoID)
		if err != nil {
			log.Error("Error updating video metadata: ", err)
			return err
		}
	}
	return nil
}
