package tasking

import (
	"encoding/json"
	"fmt"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
)

func CorrectUserProgress() error {
	// get all users
	users, err := database.GetUsers(db)
	if err != nil {
		return fmt.Errorf("error getting all users: %v", err)
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
