package tasking

import (
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
			err = task.Task(task.Args...)
			if err != nil {
				log.Errorf("Error running task: %v", err)
			}
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
func task_CorrectUserProgress(args ...interface{}) error {
	return correctUserProgress()
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
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "GetMissingVideoIDs" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_getMissingVideoIDs,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "UpdateYouTubeMetadata" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_updateAllVideoMetadata,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "DownloadYouTubeVideo" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_DownloadYouTubeVideo,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "GetMissingVideoIDsQuick" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_getMissingVideoIDsQuick,
					Args:     []interface{}{settings, db},
				})
			}

			if t.TaskName == "CorrectUserProgress" {
				tasks = append(tasks, &Task{
					Name:     t.TaskName,
					Epoch:    t.Epoch,
					Interval: t.Interval * time.Minute,
					Task:     task_CorrectUserProgress,
					Args:     []interface{}{settings, db},
				})
			}
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
