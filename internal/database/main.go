package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func selectDB(settings Settings) (*gorm.DB, error) {
	if settings.DatabaseType == "SQLite3" {
		db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{SkipDefaultTransaction: true, PrepareStmt: true})
		if err != nil {
			// return error
			return nil, err
		}
		return db, nil
	}
	if settings.DatabaseType == "Postgres" {
		dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v TimeZone=%v", settings.DatabaseHost, settings.DatabaseUser, settings.DatabasePass, settings.DatabaseName, settings.DatabasePort, settings.DatabaseSSLMode, settings.DatabaseTimeZone)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: true, PrepareStmt: true})
		if err != nil {
			// return error
			return nil, err
		}
		return db, nil
	}
	return nil, errors.New("no valid database type specified")
}

func getLocalSettings() (Settings, error) {
	// open ./settings.json file and load into Settings struct
	var settings Settings
	// open file
	file, err := os.Open("./config/settings.json")
	if err != nil {
		return settings, err
	}
	// close file
	defer file.Close()
	// decode json
	data, err := io.ReadAll(file)
	if err != nil {
		return settings, err
	}
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return settings, err
	}
	_ = os.MkdirAll(settings.BaseYouTubePath, os.ModePerm)
	return settings, nil
}

func GetDatabase() (*gorm.DB, error) {
	set, err := getLocalSettings()
	if err != nil {
		return nil, err
	}
	db, err := selectDB(set)
	if err != nil {
		// return error
		return nil, err
	}
	db.AutoMigrate(&Creator{})
	db.AutoMigrate(&Video{})
	db.AutoMigrate(&SponsorBlock{})
	db.AutoMigrate(&Comment{})
	db.AutoMigrate(&Settings{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Tasking{})
	db.AutoMigrate(&DownloadQueue{})
	db.AutoMigrate(&Playlist{})

	// insert tasks if they don't exist
	var task []Tasking
	// append tasks
	// task = append(task, Tasking{TaskName: "SyncLocalWithDB", Interval: 20, Epoch: 0}) // Move to fixed function, should only be one time
	task = append(task, Tasking{TaskName: "GetMissingVideoIDs", Interval: 60, Epoch: 0})
	task = append(task, Tasking{TaskName: "GetMissingVideoIDsQuick", Interval: 5, Epoch: 0})
	task = append(task, Tasking{TaskName: "UpdateYouTubeMetadata", Interval: 180, Epoch: 0})
	task = append(task, Tasking{TaskName: "DownloadYouTubeVideo", Interval: 5, Epoch: 0})
	task = append(task, Tasking{TaskName: "SystemCleanup", Interval: 360, Epoch: 0})

	// for each task check if it exists in database and if not insert it
	var dbcount int64
	for _, t := range task {
		db.Model(&Tasking{}).Where("task_name = ?", t.TaskName).Count(&dbcount)
		if dbcount == 0 {
			db.Create(&t)
		}
	}

	// If "Various Creators" doesn't exist, create it with channel_id = 000
	dbcount = 0
	db.Model(&Creator{}).Where("channel_id = ?", "000").Count(&dbcount)
	if dbcount == 0 {
		// Create "Various Creators" creator
		var creator Creator
		creator.ChannelID = "000"
		creator.Name = "Various Creators"
		creator.Platform = "creatorspace"

		// add creator to database
		err := db.Create(&creator).Error
		if err != nil {
			return nil, err
		}
	}

	defaultSettings := Settings{
		BaseYouTubePath: "./downloads/youtube/",
		BaseTwitchPath:  "./downloads/twitch/",
		DatabasePath:    "./database.db",
		DatabaseType:    "sqlite",
		JwtSecret:       "change-me",
	}

	// check if settings exist in database
	var count int64
	db.Model(&Settings{}).Count(&count)
	if count == 0 {
		// insert settings
		db.Create(&defaultSettings)
	}

	err = SetSettings(set, db)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func runImportMetadata(creator string, settings *Settings, db *gorm.DB, errs chan error, done chan bool) {
	err := ImportMetadata(creator, settings, db)
	if err != nil && err.Error() != "video already exists" {
		errs <- err
	}
	// done
	done <- true
}

func runImportTwitchMetadata(creator string, settings *Settings, db *gorm.DB, errs chan error, done chan bool) {
	err := ImportTwitchMetadata(creator, settings, db)
	if err != nil && err.Error() != "video already exists" {
		errs <- err
	}
	// done
	done <- true
}

func ScanLibraryAndAddToDatabase(settings *Settings, db *gorm.DB) error {
	// scan library and add to database
	// get all files from youtube base directory
	// For each creator folder in youtube base directory: get creator metadata from $creator\$creator.jso, get all video json files from $creator\metadata\metadata, get all comments from $creator\metadata\comments, get all sponsorblock from $creator\metadata\sponsorblock and add to database using functions in import.go

	// get all creators

	ytpaths, err := os.ReadDir(settings.BaseYouTubePath)
	if err != nil {
		return err
	}

	// append twitch creators to paths
	twitchPaths, err := os.ReadDir(settings.BaseTwitchPath)
	if err != nil {
		return err
	}

	var paths []string

	// append each to paths adding base path before
	for _, path := range ytpaths {
		fullPath := filepath.Join(settings.BaseYouTubePath, path.Name())
		paths = append(paths, fullPath)
	}

	for _, path := range twitchPaths {
		fullPath := filepath.Join(settings.BaseTwitchPath, path.Name())
		paths = append(paths, fullPath)
	}
	// for each creator folder get metadata and add to database in goroutine 3 creators at a time
	// create a channel to send creators to
	creators := make(chan string)
	// create a channel to receive errors
	errs := make(chan error)
	// create a channel to receive done signals
	done := make(chan bool)
	// create a waitgroup
	var wg sync.WaitGroup
	// create a goroutine to receive creators and add to database
	go func() {
		for {
			select {
			case creator := <-creators:
				// if in base youtube path, import youtube metadata
				if strings.Contains(creator, settings.BaseYouTubePath) {
					creatorName := strings.Replace(creator, settings.BaseYouTubePath, "", 1)
					creatorName = strings.Trim(creatorName, "/")

					go runImportMetadata(creatorName, settings, db, errs, done)
				} else if strings.Contains(creator, settings.BaseTwitchPath) {
					// if in base twitch path, import twitch metadata
					creatorName := strings.Replace(creator, settings.BaseTwitchPath, "", 1)
					go runImportTwitchMetadata(creatorName, settings, db, errs, done)
				} else {
					// done
					// print error
					log.Errorf("creator %s not found in base youtube path or base twitch path", creator)
					done <- true
				}
			}
		}
	}()
	// create a goroutine to receive done signals and decrement waitgroup
	go func() {
		for {
			select {
			case <-done:
				wg.Done()
			}
		}
	}()
	// create a goroutine to receive errors and return
	go func() {
		for {
			select {
			case err := <-errs:
				// print liobrary and error
				log.Errorf("error importing metadata: %s", err)
			}
		}
	}()

	count := 0
	// for each creator folder
	for _, path := range paths {
		if strings.Contains(path, "- Extras -") {
			continue
		}
		// add to waitgroup
		wg.Add(1)
		count++
		// send to creators channel
		creators <- path

		if count == 3 {
			// wait for 3 creators to be added
			wg.Wait()
			count = 0
		}
	}
	wg.Wait()
	return nil
}
