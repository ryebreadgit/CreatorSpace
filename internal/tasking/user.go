package tasking

import (
	"encoding/json"
	"fmt"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
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
			fmt.Printf("error getting completed videos for user %s: %v\n", u.Username, err)
			continue
		}

		// Get all video progress
		allProg, err := database.GetAllVideoProgress(u.UserID, db)
		if err != nil {
			// print error and continue
			fmt.Printf("error getting all video progress for user %s: %v", u.Username, err)
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
			fmt.Printf("Cleaned up %d videos for user progress %s\n", len(allProg)-len(newProg), u.Username)
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
			fmt.Printf("Cleaned up %d videos for completed videos %s\n", len(completedVideos)-len(tempCompletedVideos), u.Username)
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
