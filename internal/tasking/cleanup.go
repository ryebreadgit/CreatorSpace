package tasking

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charlievieth/fastwalk"
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
	var basePath string
	var vcName string

	// Get all creators
	creators, err := database.GetAllCreators(db)
	if err != nil {
		return fmt.Errorf("error getting all creators: %v", err)
	}
	for _, c := range creators {
		if c.ChannelID == "000" {
			basePath = filepath.Join(settings.BaseYouTubePath, c.Name)
			vcName = c.Name
		}
	}

	if basePath == "" {
		return fmt.Errorf("error getting base path for Various Creators")
	}

	// Get all videos where the path contains "/Various Creators/"
	vids, err := database.GetAllVideos(db.Where("file_path LIKE ?", fmt.Sprintf("%%/%s/%%", vcName)))
	if err != nil {
		return fmt.Errorf("error getting all videos: %v", err)
	}

	videoPath := filepath.Join(basePath, "videos")
	metadataPath := filepath.Join(basePath, "metadata", "metadata")
	commentsPath := filepath.Join(basePath, "metadata", "comments")
	subtitlePath := filepath.Join(basePath, "metadata", "subtitles")
	thumbnailPath := filepath.Join(basePath, "metadata", "thumbnails")
	sponsorblockPath := filepath.Join(basePath, "metadata", "sponsorblock")

	var errs []error

	// Check if the video is in the creator ids
	for _, c := range creators {
		for _, v := range vids {
			// Ignore the "Various Creators" creator
			if c.ChannelID == "000" {
				continue
			}
			if v.ChannelID == c.ChannelID {
				err := updateVariousUserVideo(c, v, vcName, videoPath, metadataPath, commentsPath, subtitlePath, thumbnailPath, sponsorblockPath)
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

func getFilesByVidId(vidId string, path string) ([]string, error) {
	var files []string

	conf := &fastwalk.Config{
		Follow: true,
	}

	// Iterate through all files in the path
	err := fastwalk.Walk(conf, path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error walking path %s: %v", path, err)
		}
		if d.IsDir() {
			return nil
		}
		if strings.Contains(p, vidId) {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking path %s: %v", path, err)
	}
	return files, nil
}

func updateVariousUserVideo(c database.Creator, v database.Video, oldName string, videoPath string, metadataPath string, commentsPath string, subtitlePath string, thumbnailPath string, sponsorblockPath string) error {
	vidId := v.VideoID
	var files []string

	cname, err := general.SanitizeFileName(c.Name)
	if err != nil {
		return fmt.Errorf("error sanitizing creator name: %v", err)
	}

	// Get all files for the video
	files, err = getFilesByVidId(vidId, videoPath)
	if err != nil {
		return fmt.Errorf("error getting video files for %s: %v", vidId, err)
	}

	tmpf, err := getFilesByVidId(vidId, metadataPath)
	if err != nil {
		return fmt.Errorf("error getting metadata files for %s: %v", vidId, err)
	}
	files = append(files, tmpf...)

	tmpf, err = getFilesByVidId(vidId, commentsPath)
	if err != nil {
		return fmt.Errorf("error getting comments files for %s: %v", vidId, err)
	}
	files = append(files, tmpf...)

	tmpf, err = getFilesByVidId(vidId, subtitlePath)
	if err != nil {
		return fmt.Errorf("error getting subtitle files for %s: %v", vidId, err)
	}
	files = append(files, tmpf...)

	tmpf, err = getFilesByVidId(vidId, thumbnailPath)
	if err != nil {
		return fmt.Errorf("error getting thumbnail files for %s: %v", vidId, err)
	}
	files = append(files, tmpf...)

	tmpf, err = getFilesByVidId(vidId, sponsorblockPath)
	if err != nil {
		return fmt.Errorf("error getting sponsorblock files for %s: %v", vidId, err)
	}
	files = append(files, tmpf...)

	var moveErrs []error

	// Move each file to the new creator folder, replacing "Various Creators" with cname in the path
	for _, f := range files {
		newPath := strings.ReplaceAll(f, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))
		err = os.MkdirAll(filepath.Dir(newPath), 0755)
		if err != nil {
			moveErrs = append(moveErrs, fmt.Errorf("error making folders for %s: %v", newPath, err))
			continue
		}
		err = os.Rename(f, newPath)
		if err != nil {
			moveErrs = append(moveErrs, fmt.Errorf("error moving file %s to %s: %v", f, newPath, err))
		}
	}

	if len(moveErrs) > 0 {
		errsStr := "\n\t"
		for _, e := range moveErrs {
			errsStr += e.Error() + "\n\t"
		}
		log.Warnf("errors encountered while moving files for cleanup of %v: %v", vidId, errsStr)
	}

	// Update the video in the database
	v.FilePath = strings.ReplaceAll(v.FilePath, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))
	v.MetadataPath = strings.ReplaceAll(v.MetadataPath, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))
	v.CommentsPath = strings.ReplaceAll(v.CommentsPath, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))
	v.SubtitlePath = strings.ReplaceAll(v.SubtitlePath, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))
	v.ThumbnailPath = strings.ReplaceAll(v.ThumbnailPath, fmt.Sprintf("/%s/", oldName), fmt.Sprintf("/%s/", cname))

	// Check if paths exist
	if v.FilePath != "" {
		if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.FilePath)); err != nil {
			return fmt.Errorf("error checking video file %s: %v", filepath.Join(settings.BaseYouTubePath, v.FilePath), err)
		}
	}
	if v.MetadataPath != "" {
		if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.MetadataPath)); err != nil {
			return fmt.Errorf("error checking metadata file %s: %v", filepath.Join(settings.BaseYouTubePath, v.MetadataPath), err)
		}
	}
	if v.CommentsPath != "" {
		if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.CommentsPath)); err != nil {
			return fmt.Errorf("error checking comments file %s: %v", filepath.Join(settings.BaseYouTubePath, v.CommentsPath), err)
		}
	}
	if v.ThumbnailPath != "" {
		if _, err := os.Stat(filepath.Join(settings.BaseYouTubePath, v.ThumbnailPath)); err != nil {
			return fmt.Errorf("error checking thumbnail file %s: %v", filepath.Join(settings.BaseYouTubePath, v.ThumbnailPath), err)
		}
	}

	err = database.UpdateVideo(v, db)
	if err != nil {
		return fmt.Errorf("error updating video %s: %v", v.VideoID, err)
	}
	return nil
}
