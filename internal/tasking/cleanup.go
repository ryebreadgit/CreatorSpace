package tasking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	log "github.com/sirupsen/logrus"
)

func correctUserProgress() error {
	// get all users
	users, err := database.GetUsers(db)
	if err != nil {
		return fmt.Errorf("error getting all users: %v", err)
	}

	videoIds := []string{}
	viddata, err := database.GetAllVideos(db.Select("video_id"))
	if err != nil {
		return fmt.Errorf("error getting all videos: %v", err)
	}
	for _, v := range viddata {
		videoIds = append(videoIds, v.VideoID)
	}

	// for each user, get the completeed videos
	for _, u := range users {
		// get all completed videos
		completedVideos, err := database.GetPlaylistByUserID(u.UserID, "Completed Videos", db)
		if err != nil {
			// print error and continue
			log.Errorf("error getting completed videos for user %s: %v\n", u.Username, err)
			continue
		}

		// Get all video progress
		allProg, err := database.GetAllVideoProgress(u.UserID, db)
		if err != nil {
			// print error and continue
			log.Errorf("error getting all video progress for user %s: %v", u.Username, err)
			continue
		}

		newProg := []database.ProgressToken{}

		// for each video progress, check if it's in the completed videos
		for _, prog := range allProg {
			// if it's completed delete it from the video progress
			if !contains(completedVideos, prog.VideoID) {
				newProg = append(newProg, prog)
			}
		}

		// Check if the newProg is different from the old one
		if len(newProg) != len(allProg) {
			log.Infof("Cleaned up %d videos for user progress %s\n", len(allProg)-len(newProg), u.Username)
			// if it is, update the video progress
			newProgJson, err := json.Marshal(newProg)
			if err != nil {
				return fmt.Errorf("error marshalling new video progress for user %s: %v", u.Username, err)
			}

			err = database.UpdateUserProgress(u.UserID, string(newProgJson), db)
			if err != nil {
				return fmt.Errorf("error updating video progress for user %s: %v", u.Username, err)
			}
		}

		var tempCompletedVideos []string

		// Check if any videos in completed videos are not in the video ids
		for _, v := range completedVideos {
			if contains(videoIds, v) {
				tempCompletedVideos = append(tempCompletedVideos, v)
			}
		}

		// Remove any duplicates
		for i := 0; i < len(tempCompletedVideos); i++ {
			for j := i + 1; j < len(tempCompletedVideos); j++ {
				if tempCompletedVideos[i] == tempCompletedVideos[j] {
					tempCompletedVideos = append(tempCompletedVideos[:j], tempCompletedVideos[j+1:]...)
					j--
				}
			}
		}

		// Check if the tempCompletedVideos is different from the old one
		if len(tempCompletedVideos) != len(completedVideos) {
			log.Infof("Cleaned up %d videos for completed videos %s\n", len(completedVideos)-len(tempCompletedVideos), u.Username)
			// if it is, update the completed videos
			completedJson, err := json.Marshal(tempCompletedVideos)
			if err != nil {
				return fmt.Errorf("error marshalling new completed videos for user %s: %v", u.Username, err)
			}
			err = database.UpdatePlaylistByUserId(u.UserID, "Completed Videos", string(completedJson), db)
			if err != nil {
				return fmt.Errorf("error updating completed videos for user %s: %v", u.Username, err)
			}
		}
	}

	return nil
}

func correctVariousUsers() error {
	// Get all videos where the path contains "/Various Creators/"
	vids, err := database.GetAllVideos(db.Where("file_path LIKE ?", "%/Various Creators/%"))
	if err != nil {
		return fmt.Errorf("error getting all videos: %v", err)
	}

	// Get all creators
	creators, err := database.GetAllCreators(db)
	if err != nil {
		return fmt.Errorf("error getting all creators: %v", err)
	}

	var errs []error

	// Check if the video is in the creator ids
	for _, c := range creators {
		for _, v := range vids {
			// Ignore the "Various Creators" creator
			if c.ChannelID == "000" {
				continue
			}
			if v.ChannelID == c.ChannelID {
				err := updateVariousUserVideo(c, v)
				if err != nil {
					errs = append(errs, err)
				}
				log.Debugf("Updated %s to be under correct creator: %s (%s)\n", v.VideoID, c.Name, c.ChannelID)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors encountered: %v", errs)
	}
	return nil
}

func updateVariousUserVideo(c database.Creator, v database.Video) error {
	tmpslc := strings.Split(c.FilePath, "/")
	tmpname := tmpslc[len(tmpslc)-2]

	cname, err := general.SanitizeFileName(tmpname)
	if err != nil {
		return fmt.Errorf("error sanitizing creator name: %v", err)
	}

	// Replace "Various Creators" with the creator name
	var newVidPath, newMetaPath, newThumbPath, newCommentsPath, newSubData string
	newVidPath = strings.ReplaceAll(v.FilePath, "/Various Creators/", fmt.Sprintf("/%s/", cname))
	if v.ThumbnailPath != "" {
		newThumbPath = strings.ReplaceAll(v.ThumbnailPath, "/Various Creators/", fmt.Sprintf("/%s/", cname))
	}
	if v.MetadataPath != "" {
		newMetaPath = strings.ReplaceAll(v.MetadataPath, "/Various Creators/", fmt.Sprintf("/%s/", cname))
	}
	if v.CommentsPath != "" {
		newCommentsPath = strings.ReplaceAll(v.CommentsPath, "/Various Creators/", fmt.Sprintf("/%s/", cname))
	}
	if v.SubtitlePath != "" {
		newSubData = strings.ReplaceAll(v.SubtitlePath, "/Various Creators/", fmt.Sprintf("/%s/", cname))
	}

	// Move the files

	// Check if the video file exists
	if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.FilePath)); err == nil {
		// Make folders as necessary
		err = os.MkdirAll(filepath.Dir(filepath.Join(settings.BaseYouTubePath, newVidPath)), 0755)
		if err != nil {
			return fmt.Errorf("error making folders for %s: %v", filepath.Join(settings.BaseYouTubePath, newVidPath), err)
		}
		// Move the video file
		err = os.Rename(filepath.Join(settings.BaseYouTubePath, v.FilePath), filepath.Join(settings.BaseYouTubePath, newVidPath))
		if err != nil {
			return fmt.Errorf("error moving video file %s to %s: %v", filepath.Join(settings.BaseYouTubePath, v.FilePath), filepath.Join(settings.BaseYouTubePath, newVidPath), err)
		}

		// Update the video path
		v.FilePath = newVidPath
	}

	// Check if the thumbnail file exists
	if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.ThumbnailPath)); err == nil {
		// Make folders as necessary
		err = os.MkdirAll(filepath.Dir(filepath.Join(settings.BaseYouTubePath, newVidPath)), 0755)
		if err != nil {
			return fmt.Errorf("error making folders for %s: %v", filepath.Join(settings.BaseYouTubePath, newVidPath), err)
		}
		// Move the thumbnail file
		err = os.Rename(filepath.Join(settings.BaseYouTubePath, v.ThumbnailPath), filepath.Join(settings.BaseYouTubePath, newThumbPath))
		if err != nil {
			return fmt.Errorf("error moving thumbnail file %s to %s: %v", filepath.Join(settings.BaseYouTubePath, v.ThumbnailPath), filepath.Join(settings.BaseYouTubePath, newThumbPath), err)
		}

		// Update the thumbnail path
		v.ThumbnailPath = newThumbPath
	}

	// Check if the metadata file exists
	if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.MetadataPath)); err == nil {
		// Make folders as necessary
		err = os.MkdirAll(filepath.Dir(filepath.Join(settings.BaseYouTubePath, newVidPath)), 0755)
		if err != nil {
			return fmt.Errorf("error making folders for %s: %v", filepath.Join(settings.BaseYouTubePath, newVidPath), err)
		}
		// Move the metadata file
		err = os.Rename(filepath.Join(settings.BaseYouTubePath, v.MetadataPath), filepath.Join(settings.BaseYouTubePath, newMetaPath))
		if err != nil {
			return fmt.Errorf("error moving metadata file %s to %s: %v", filepath.Join(settings.BaseYouTubePath, v.MetadataPath), filepath.Join(settings.BaseYouTubePath, newMetaPath), err)
		}

		// Update the metadata path
		v.MetadataPath = newMetaPath
	}

	// Check if the comments file exists
	if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.CommentsPath)); err == nil {
		// Make folders as necessary
		err = os.MkdirAll(filepath.Dir(filepath.Join(settings.BaseYouTubePath, newVidPath)), 0755)
		if err != nil {
			return fmt.Errorf("error making folders for %s: %v", filepath.Join(settings.BaseYouTubePath, newVidPath), err)
		}
		// Move the comments file
		err = os.Rename(filepath.Join(settings.BaseYouTubePath, v.CommentsPath), filepath.Join(settings.BaseYouTubePath, newCommentsPath))
		if err != nil {
			return fmt.Errorf("error moving comments file %s to %s: %v", filepath.Join(settings.BaseYouTubePath, v.CommentsPath), filepath.Join(settings.BaseYouTubePath, newCommentsPath), err)
		}

		// Update the comments path
		v.CommentsPath = newCommentsPath
	}

	// Check if subtitles string is empty
	if v.SubtitlePath != "" {
		// Parse json to subtitles
		var subtitles []database.VidSubtitle
		err = json.Unmarshal([]byte(v.SubtitlePath), &subtitles)
		if err != nil {
			return fmt.Errorf("error unmarshalling subtitles: %v", err)
		}

		var newSubs []database.VidSubtitle
		err = json.Unmarshal([]byte(newSubData), &newSubs)
		if err != nil {
			return fmt.Errorf("error unmarshalling new subtitles: %v", err)
		}

		// Get the parent folder of the first subtitle file
		subParent := filepath.Dir(filepath.Join(settings.BaseYouTubePath, subtitles[0].FilePath))
		newSubParent := filepath.Dir(filepath.Join(settings.BaseYouTubePath, newSubs[0].FilePath))

		// Make folders as necessary
		err = os.MkdirAll(newSubParent, 0755)
		if err != nil {
			return fmt.Errorf("error making folders for %s: %v", newSubParent, err)
		}

		// Move the sub parent folder to the new sub parent folder
		err = os.Rename(subParent, newSubParent)
		if err != nil {
			return fmt.Errorf("error moving subtitle folder %s to %s: %v", subParent, newSubParent, err)
		}

		// Update the subtitle path
		v.SubtitlePath = newSubData

	}

	// Update the video in the database
	err = database.UpdateVideo(v, db)
	if err != nil {
		return fmt.Errorf("error updating video %s: %v", v.VideoID, err)
	}
	return nil
}
