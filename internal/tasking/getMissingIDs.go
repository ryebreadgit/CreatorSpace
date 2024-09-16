package tasking

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func fetchChannelVideoIDs(channelID string, vidtype string, limit int) ([]string, error) {
	if limit > 0 {
		log.Debugf("Fetching video IDs for %v-%v (quick)", channelID, vidtype)
	} else {
		log.Debugf("Fetching video IDs for %v-%v (full)", channelID, vidtype)
	}
	var addr string
	if vidtype == "playlist" {
		addr = fmt.Sprintf("https://www.youtube.com/playlist?list=%v", channelID)
	} else if vidtype == "channel" {
		addr = fmt.Sprintf("https://www.youtube.com/channel/%v/videos", channelID)
	} else if vidtype == "shorts" {
		addr = fmt.Sprintf("https://www.youtube.com/channel/%v/shorts", channelID)
	} else if vidtype == "streams" {
		addr = fmt.Sprintf("https://www.youtube.com/channel/%v/streams", channelID)
	} else if vidtype == "live" {
		addr = fmt.Sprintf("https://www.youtube.com/channel/%v/live", channelID)
	} else {
		return nil, fmt.Errorf("invalid video type")
	}

	// Check if channel is already in database. If not, get creator json from yt-dlp and insert into db
	_, err := database.GetCreator(channelID, db)
	if err != nil {
		// If vid type is not playlist, get creator json from yt-dlp and insert into db
		if vidtype != "playlist" && vidtype != "live" {
			_, err = getNewCreator(channelID)
			if err != nil {
				return nil, err
			}
		}
	}

	ytargs := []string{"--config-locations", "./config/youtube-video-fetch_id.conf", addr}
	if limit > 0 {
		ytargs = append(ytargs, "--playlist-end", fmt.Sprintf("%v", limit))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*15) // 15 minute timeout (This is a long timeout because yt-dlp can take a long time to fetch video IDs)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", ytargs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if the error is a context timeout
		if osErr, ok := err.(*os.PathError); ok && osErr.Err == context.DeadlineExceeded {
			return nil, fmt.Errorf("yt-dlp command timed out after 15 minutes on fetching video ids for '%v'", channelID)
		}

		// Check if yt-dlp returned data in stdout. If so, try to continue with that data.
		if len(stdout.Bytes()) > 0 {
			log.Debugf("yt-dlp command failed on '%v', but returned data in stdout. Continuing with that data.\n", channelID)
		} else {
			return nil, fmt.Errorf("yt-dlp command failed on '%v': %v, stderr: %s", channelID, err, stderr.String())
		}
	}

	videoIDs := make([]string, 0)
	output := stdout.String()
	// output is $id\n$id\n$id\n
	for _, id := range bytes.Split([]byte(output), []byte{'\n'}) {
		if len(id) > 0 {
			videoIDs = append(videoIDs, string(id))
		}
	}

	return videoIDs, nil
}

func getMissingVideoIDs(settings *database.Settings, limit int, db *gorm.DB) error {
	// get all channels
	dlqueue, err := database.GetDownloadQueue(db)
	if err != nil {
		return err
	}

	var reterr error

	// create channels for communication between goroutines
	workChan := make(chan fetchWorkItem, 100)
	videoIDChan := make(chan videoWorkItem, 100)
	errChan := make(chan error, 100)

	// create workers to fetch video IDs
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go fetchWorker(workChan, videoIDChan, &wg, errChan)
	}

	// create goroutine to process video IDs
	go processVideoIDs(videoIDChan, limit, settings, db, errChan)

	// send fetch work items to the workers
	for _, queueitem := range dlqueue {
		// Check if video type is channel, playlist, shorts, or live. Skip others.
		allowedTypes := []string{"channel", "playlist", "shorts", "live"}
		if !general.StringInSlice(allowedTypes, queueitem.VideoType) {
			continue
		}

		var channel database.Creator
		channelID := queueitem.VideoID
		if queueitem.VideoType == "playlist" {
			// If this is a playlist then the channel will be various creators for now.
			channel = database.Creator{
				ChannelID: "000",
				Name:      "Various Creators",
			}
		} else {
			// get channel info based on channel ID
			channel, err = database.GetCreator(channelID, db)
			if err != nil {
				// If the channel is not in the database, then create it
				channel, err = getNewCreator(channelID)
				if err != nil {
					errChan <- err
					continue
				}
			}
		}

		// send fetch work item to the workChan channel
		workChan <- fetchWorkItem{
			channelID: channelID,
			channel:   channel,
			videoType: queueitem.VideoType, // set the videoType field
			limit:     limit,
		}
	}

	// close the workChan channel to signal that no more work items will be sent
	close(workChan)

	// wait for all workers to complete
	wg.Wait()

	// close the videoIDChan channel to signal that no more IDs will be sent
	close(videoIDChan)

	// close the errChan channel to signal that no more errors will be sent
	close(errChan)

	// check for errors
	for err := range errChan {
		reterr = fmt.Errorf("%v\n%w", err, reterr)
	}

	return reterr
}

type fetchWorkItem struct {
	channelID string
	channel   database.Creator
	videoType string
	limit     int
}

type videoWorkItem struct {
	videoID   string
	videoType string
	channelID string
}

func fetchWorker(workChan chan fetchWorkItem, videoIDChan chan videoWorkItem, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()
	for workItem := range workChan {
		// fetch video IDs
		videoIDs, err := fetchChannelVideoIDs(workItem.channelID, workItem.videoType, workItem.limit)
		if err != nil {
			errChan <- err
			continue
		}
		// send each video ID through the videoIDChan channel
		for _, videoID := range videoIDs {
			videoWorkItem := videoWorkItem{
				videoID:   videoID,
				videoType: workItem.videoType,
				channelID: workItem.channelID,
			}
			videoIDChan <- videoWorkItem
		}
	}
}

func processVideoIDs(videoIDChan chan videoWorkItem, limit int, settings *database.Settings, db *gorm.DB, errChan chan error) {
	for videoID := range videoIDChan {
		// check if in database
		_, err := database.GetVideo(videoID.videoID, db)
		if err != nil {

			var vidType string
			var folderName string
			// determine the vidType and folderName based on the video type
			switch videoID.videoType {
			case "channel":
				folderName = "videos"
				vidType = "video"
			case "playlist":
				folderName = "videos"
				vidType = "video"
			case "shorts":
				folderName = "shorts"
				vidType = "short"
			case "streams":
				folderName = "streams"
				vidType = "stream"
			case "live":
				folderName = "streams"
				vidType = "stream"
			}
			// check if in download queue
			_, err := database.GetDownloadQueueItem(videoID.videoID, videoID.videoType, db)
			if err != nil {
				// add download queue item
				var item database.DownloadQueue
				item.VideoID = videoID.videoID
				item.VideoType = vidType
				item.Approved = true
				item.Downloaded = false
				item.Source = "youtube"

				item.Requester = getRequester(limit)

				channel, err := database.GetCreator(videoID.channelID, db)
				if err != nil {
					if videoID.videoType == "playlist" {
						// If the channel is not in the database, then instead use chanel id 000 and Various Creators as the channel name
						channel = database.Creator{
							Name: "Various Creators",
						}
					} else {
						// If the channel is not in the database, then create it
						channel, err = getNewCreator(videoID.channelID)
						if err != nil {
							errChan <- err
							continue
						}
					}
				}

				chanName, err := general.SanitizeFileName(channel.Name)
				if err != nil {
					log.Errorf("Error sanitizing channel name %v: %v\n", channel.Name, err)
					errChan <- err
					continue
				}

				if channel.FilePath != "" {

					// Pull name from filepath if possible

					tmpslc := strings.Split(channel.FilePath, "/")
					tmpname := tmpslc[len(tmpslc)-2]

					chanName, err = general.SanitizeFileName(tmpname)
					if err != nil {
						log.Errorf("Error sanitizing channel name %v: %v\n", channel.Name, err)
						errChan <- err
						continue
					}
				}

				item.DownloadPath = fmt.Sprintf("%v/%v/%v/%v", settings.BaseYouTubePath, chanName, folderName, videoID.videoID)

				// Make the download path parent directories if they don't exist
				err = os.MkdirAll(filepath.Dir(item.DownloadPath), 0755)
				if err != nil {
					log.Errorf("Error making download path %v: %v\n", item.DownloadPath, err)
					errChan <- err
					continue
				}

				err = database.InsertDownloadQueueItem(item, db)
				if err != nil {
					if strings.Contains(err.Error(), "item already exists") {
						continue
					}
					log.Errorf("Error inserting download queue item for video %v: %v\n", videoID.videoID, err)
					errChan <- err
					continue
				}
			}
		}
	}
}

func getRequester(limit int) string {
	if limit > 0 {
		return "system-quick"
	} else {
		return "system-full"
	}
}
