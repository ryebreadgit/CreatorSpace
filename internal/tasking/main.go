package tasking

import (
	"context"
	"fmt"
	"time"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
)

var db *gorm.DB
var settings *database.Settings

func runTask(task *Task, db *gorm.DB) {
	for {
		now := time.Now().Unix()
		if now >= task.Epoch {
			err := database.UpdateTaskEpochLastRanByName(task.Name, now, db)
			if err != nil {
				log.Errorf("Error updating task epoch: %v", err)
			}
			log.Infof("Started running task '%v' at '%v'", task.Name, time.Now().Format("2006-01-02 15:04:05"))

			ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)

			done := make(chan error, 1)
			go func() {
				done <- task.Task(task.Args...)
			}()

			select {
			case err = <-done:
				if err != nil {
					log.Errorf("Error running task '%v': %v", task.Name, err)
				}
			case <-ctx.Done():
				log.Warnf("Task '%v' timed out", task.Name)
			}

			// Explicitly call cancel to release resources
			cancel()

			task.Lock()
			task.Epoch = time.Now().Unix() + int64(task.Interval.Seconds())

			// update epoch
			err = database.UpdateTaskEpochByName(task.Name, task.Epoch, db)
			if err != nil {
				log.Errorf("Error updating task epoch: %v", err)
			}

			log.Infof("Completed running '%v' at '%v', next run at '%v'", task.Name, time.Now().Format("2006-01-02 15:04:05"), time.Unix(task.Epoch, 0).Format("2006-01-02 15:04:05"))
			task.Unlock()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func task_ScanLibraryAndAddToDatabase(args ...interface{}) error {
	settings := args[0].(*database.Settings)
	db := args[1].(*gorm.DB)
	return database.ScanLibraryAndAddToDatabase(settings, db)
}

func task_getMissingVideoIDs(args ...interface{}) error {
	return getMissingVideoIDs(settings, 0, db)
}

func task_getMissingVideoIDsQuick(args ...interface{}) error {
	return getMissingVideoIDs(settings, 3, db)
}

func task_updateAllVideoMetadata(args ...interface{}) error {
	return updateAllVideoMetadata()
}

func task_DownloadYouTubeVideo(args ...interface{}) error {
	return downloadYouTubeVideos(settings, db)
}

/*func task_UpdateTweets(args ...interface{}) error {
	return UpdateTwitterCreators(args[0].(int))
}*/

func task_SystemCleanup(args ...interface{}) error {
	var errs []error
	errs = append(errs, correctVariousUsers())
	errs = append(errs, correctUserProgress())
	if len(errs) > 0 {
		errStr := "\n\t"
		for _, e := range errs {
			if e != nil {
				errStr += e.Error() + "\n\t"
			}
		}
		if errStr != "\n\t" {
			return fmt.Errorf("errors: %v", errStr)
		}
	}
	return nil
}

func InitTasking() {

	tasking, err := database.GetAllTasks(db)
	if err == nil {
		var tasks []*Task

		for _, t := range tasking {

			// if the interval is 0, then the task is disabled
			if t.Interval == 0 {
				continue
			}

			if t.TaskName == "SyncLocalWithDB" {
				// get epoch from database
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_ScanLibraryAndAddToDatabase,
					Timeout:  1 * time.Hour,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "GetMissingVideoIDs" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_getMissingVideoIDs,
					Timeout:  4 * time.Hour,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "UpdateYouTubeMetadata" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_updateAllVideoMetadata,
					Timeout:  4 * time.Hour,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "DownloadYouTubeVideo" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_DownloadYouTubeVideo,
					Timeout:  1 * time.Hour,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "GetMissingVideoIDsQuick" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_getMissingVideoIDsQuick,
					Timeout:  30 * time.Minute,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "SystemCleanup" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_SystemCleanup,
					Timeout:  30 * time.Minute,
					Args:     []interface{}{settings, db},
				})
			}
			/*
				if t.TaskName == "UpdateTweetsQuick" {
					tasks = append(tasks, &Task{
						Name:     t.TaskName,
						Epoch:    t.Epoch,
						Interval: t.Interval * time.Minute,
						Task:     task_UpdateTweets,
						Args:     []interface{}{100},
					})
				}

				if t.TaskName == "UpdateTweets" {
					tasks = append(tasks, &Task{
						Name:     t.TaskName,
						Epoch:    t.Epoch,
						Interval: t.Interval * time.Minute,
						Task:     task_UpdateTweets,
						Args:     []interface{}{5},
					})
				}
			*/
		}

		for _, t := range tasks {
			go runTask(t, db)
		}
		for {
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		log.Errorf("Error getting tasking from database: %v", err)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func init() {
	var err error
	// get database
	db, err = database.GetDatabase()
	if err != nil {
		fmt.Printf("Error connecting to database: %s\n", err)
		return
	}

	// get settings
	settings, err = database.GetSettings(db)
	if err != nil {
		fmt.Printf("Error getting settings: %s\n", err)
		return
	}
}
