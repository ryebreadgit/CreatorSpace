package tasking

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/webp"
	"gorm.io/gorm"

	"github.com/corona10/goimagehash"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
)

// Function to check if a video is available, unavailable, or private

func GetYouTubeMetadata(url string, comments bool) (database.YouTubeVideoInfoStruct, error) {
	// Construct the command to run yt-dlp and get video info in JSON format
	metadataArgs := []string{
		"--skip-download",
		"--dump-json",
		url,
	}
	if comments {
		metadataArgs = append(metadataArgs, "--write-comments")
		metadataArgs = append(metadataArgs, "--extractor-args", "youtube:comment_sort=top")
	}
	cmd := exec.Command("yt-dlp", metadataArgs...)
	out, err := cmd.Output()
	if err != nil {
		// check if video is private. Check stderr for "This video is private"
		errstr := strings.ToLower(string(err.(*exec.ExitError).Stderr))
		// check if just a warning
		if strings.Contains(errstr, "warning") {
			// check if data in stdout is valid
			var info database.YouTubeVideoInfoStruct
			err = json.Unmarshal(out, &info)
			if err != nil {
				fmt.Println("Error unmarshalling JSON data: ", err)
				return database.YouTubeVideoInfoStruct{}, err
			}
			return info, nil
		} else if strings.Contains(errstr, "private video") {
			return database.YouTubeVideoInfoStruct{}, fmt.Errorf("private video")
		} else if strings.Contains(errstr, "unavailable") || strings.Contains(errstr, "this video has been removed") {
			return database.YouTubeVideoInfoStruct{}, fmt.Errorf("unavailable video")
		} else {
			fmt.Println("Error getting video metadata: ", err)
			return database.YouTubeVideoInfoStruct{}, err
		}
	}
	// Parse the JSON data to check if the video is private or available
	var info database.YouTubeVideoInfoStruct
	err = json.Unmarshal(out, &info)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data: ", err)
		return database.YouTubeVideoInfoStruct{}, err
	}
	return info, nil
}

func downloadThumbnail(thumbUrl string, oldThumbnailPath string, videoID string) (string, error) {
	// Download the new thumbnail to ./data/tmp/$videoID.jpg. Use net/http to download the thumbnail
	res, err := http.Get(thumbUrl)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	// Set extension based on content type
	var extension string
	switch res.Header.Get("Content-Type") {
	case "image/jpeg":
		extension = "jpg"
	case "image/png":
		extension = "png"
	case "image/gif":
		extension = "gif"
	case "image/webp":
		extension = "webp"
	default:
		return "", fmt.Errorf("unknown content type: %v", res.Header.Get("Content-Type"))
	}

	// Save the thumbnail to disk
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	tmpimg := fmt.Sprintf("./data/tmp/%v.%v", videoID, extension)
	err = os.WriteFile(tmpimg, data, 0644)
	if err != nil {
		return "", err
	}

	// Compare the thumbnails

	if oldThumbnailPath != "" {
		img1, err := loadImage(tmpimg)
		if err != nil {
			_ = os.Remove(tmpimg)
			return "", err
		}
		img2, err := loadImage(oldThumbnailPath)
		if err == nil {
			hash1, err := goimagehash.PerceptionHash(img1)
			if err != nil {
				_ = os.Remove(tmpimg)
				fmt.Printf("Error comparing hash1 hashes: %v\n", err)
			}

			hash2, err := goimagehash.PerceptionHash(img2)
			if err != nil {
				_ = os.Remove(tmpimg)
				fmt.Printf("Error comparing hash2 hashes: %v\n", err)
			}

			// Compare the hash codes
			distance, err := hash1.Distance(hash2)
			if err != nil {
				_ = os.Remove(tmpimg)
				fmt.Printf("Error comparing hash distance: %v\n", err)
			}

			if distance == 0 {
				_ = os.Remove(tmpimg)
				return "", fmt.Errorf("thumbnails are the same")
			}
		}
	}

	// If the images are different, save the new thumbnail to the disk. Move the old thumbnail to video.ThumbnailPath.00n where n increments if the file already exists
	var thumbnailNum int
	var newThumbPath string
	for {
		// Check if the file exists
		baseThumb := filepath.Base(oldThumbnailPath)

		// Remove ALL extensions
		for {
			ext := filepath.Ext(baseThumb)
			// I know this looks dumb, I tried to just kill all extensions but it, on rare occasions, also include the . in the filename. And sometimes not. So here we are.
			if ext == ".json" || ext == ".mp4" || ext == ".ts" || ext == ".jpg" || ext == ".png" || ext == ".webp" || strings.Contains(ext, ".00") || strings.Contains(ext, ".01") || strings.Contains(ext, ".02") {
				// remove extension
				baseThumb = strings.TrimSuffix(baseThumb, ext)
			} else {
				break
			}
		}

		newThumbPath, err = general.SanitizeFilePath(fmt.Sprintf("%v/%v.%03d.%v", filepath.Dir(oldThumbnailPath), strings.TrimSuffix(baseThumb, filepath.Ext(oldThumbnailPath)), thumbnailNum, extension))
		if err != nil {
			return "", err
		}

		// Create parent directories if they don't exist
		err = os.MkdirAll(filepath.Dir(newThumbPath), 0755)
		if err != nil {
			return "", err
		}

		_, err = os.Stat(newThumbPath)
		if err != nil && os.IsNotExist(err) {
			// If the file doesn't exist, save the file
			err = os.WriteFile(newThumbPath, data, 0644)
			if err != nil {
				fmt.Printf("Error saving thumbnail '%v' for video '%v' due to the following error: %v\n", newThumbPath, videoID, err.Error())
				return "", err
			}
			break
		} else {
			thumbnailNum++
		}
	}

	// Update the video thumbnail path in the database. Remove the youtube path from the path
	newThumbPath = strings.TrimPrefix(newThumbPath, settings.BaseYouTubePath)
	newThumbPath = strings.ReplaceAll(newThumbPath, "//", "/")
	newThumbPath = strings.ReplaceAll(newThumbPath, "\\", "/")
	return newThumbPath, nil
}

// downloadYouTubeVideo downloads a YouTube video using yt-dlp. Pass through the video url, the output directory, yt-dlp config, and the output filename
func downloadYouTubeVideo(url string, outputDir string, videoid string, config string) (string, error) {
	// Construct the command to run yt-dlp and download the video
	outputLoc := fmt.Sprintf("%v/%v", outputDir, videoid)
	outputLoc, err := general.SanitizeFilePath(outputLoc)
	if err != nil {
		return "", err
	}

	tmpPath := fmt.Sprintf("./data/tmp/%v", videoid)

	cmd := exec.Command("yt-dlp", "--config-location", config, "-o", tmpPath, url)

	// Create a pipe for capturing the output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Read the output in a separate goroutine
	done := make(chan error)
	go func() {
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			done <- err
		}
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			done <- err
		}
		done <- nil
	}()

	// Wait for the goroutine to finish reading the output
	if err := <-done; err != nil {
		return "", fmt.Errorf("error reading output: %v", err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Error downloading video '%v' due to the following error: %v\n", url, err)
		// Check if any files were downloaded and delete them
		fif, err2 := general.StringInFolder(videoid, "./data/tmp/")
		if err2 != nil {
			return "", err2
		}
		if fif != nil {
			err2 = os.RemoveAll(tmpPath)
			if err2 != nil {
				return "", err2
			}
		}
		return "", err
	}

	// Make the output directory if it doesn't exist
	err = os.MkdirAll(filepath.Dir(outputLoc), 0755)
	if err != nil {
		return "", err
	}

	// Check outputDir for files with the video ID in the name
	fif, err := general.StringInFolder(videoid, filepath.Dir(tmpPath))
	if err != nil {
		return "", err
	}

	// If there are no files, return an error
	if fif == nil {
		return "", fmt.Errorf("no files found in output directory")
	}

	fifName := fif[0]

	// If there is more than one file, check for a file with a video extension
	if len(fif) > 1 {
		for _, file := range fif {
			extension := filepath.Ext(file)
			if extension == ".mp4" || extension == ".ts" || extension == ".webm" || extension == ".mkv" {
				fifName = file
			} else {
				// Delete the file
				err = os.Remove(file)
				if err != nil {
					return "", err
				}
			}
		}
	}

	// Quick metadata query to get video title for output filename
	met, err := GetYouTubeMetadata(url, false)
	if err != nil {
		return "", err
	}

	// Set outputFilename
	outputFilename := fmt.Sprintf("%v (%v)", met.Title, videoid)
	outputFilename, err = general.SanitizeFileName(outputFilename)
	if err != nil {
		return "", err
	}

	// If there is only one file, change the base name to the output filename but keep the extension
	extension := filepath.Ext(fifName)
	newFilename := fmt.Sprintf("%v/%v%v", outputDir, outputFilename, extension)
	newFilename, err = general.SanitizeFilePath(newFilename)
	if err != nil {
		return "", err
	}

	// Move the file
	err = general.Move(fifName, newFilename)
	if err != nil {
		return "", err
	}

	return newFilename, nil
}

func updateVideoMetadata(videoID string) error {
	//fmt.Printf("Checking metadata updates for video '%v'\n", videoID) // Change to debug
	// Get the video from the database
	video, err := database.GetVideo(videoID, db)
	if err != nil {
		return err
	}
	// Get the video metadata from YouTube
	info, err := GetYouTubeMetadata(fmt.Sprintf("https://www.youtube.com/watch?v=%v", videoID), false)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unavailable video") {
			video.Availability = "unavailable"
			err = database.UpdateVideo(video, db)
			if err != nil {
				return err
			}
			return nil
		} else if strings.Contains(strings.ToLower(err.Error()), "private video") {
			video.Availability = "private"
			err = database.UpdateVideo(video, db)
			if err != nil {
				return err
			}
			return nil
		} else {
			fmt.Println("Error getting video metadata: ", err)
			return err
		}
	}
	// create a copy of our original video
	originalVideo := video
	// Update the video metadata in the database
	video.Title = info.Title
	video.Description = info.Description
	video.Length = fmt.Sprintf("%v", info.Duration)
	if info.UploadDate != "" {
		// use general.ParseDate to parse the date
		if info.ReleaseTimestamp != 0 {
			// Timestamp will be a unix timestamp, so convert to a 2023-03-16T00:00:00Z format
			video.PublishedAt = general.EpochToDate(int64(info.ReleaseTimestamp))
			video.Epoch = int64(info.ReleaseTimestamp)
		} else {
			parsedDate := info.UploadDate
			// convert 20230316 to PT2023-03-16T00:00:00Z
			parsedDate = fmt.Sprintf("%v-%v-%vT00:00:00Z", parsedDate[0:4], parsedDate[4:6], parsedDate[6:8])
			video.PublishedAt = parsedDate

			// convert to epoch
			parsedDateEpoch, err := general.DateToEpoch(parsedDate)
			if err != nil {
				fmt.Println("Error converting date to epoch: ", err)
			} else {
				video.Epoch = parsedDateEpoch
			}
		}
	}

	// check video availability
	switch info.Availability {
	case "unlisted":
		video.Availability = "unlisted"
	case "public":
		video.Availability = "available"
	case "private":
		video.Availability = "private"
	case "needs_auth":
		video.AgeRestricted = true
	default:
		video.Availability = "unavailable"
	}
	// if thumbnail path, video path, or metadata path have the youtube default path, trim this
	video.ThumbnailPath = strings.ReplaceAll(video.ThumbnailPath, settings.BaseYouTubePath, "")
	// if the metadata path has the youtube default path, trim this
	video.MetadataPath = strings.ReplaceAll(video.MetadataPath, settings.BaseYouTubePath, "")

	// replace \ with / in the these paths and // with /
	video.ThumbnailPath = strings.ReplaceAll(video.ThumbnailPath, "//", "/")
	video.MetadataPath = strings.ReplaceAll(video.MetadataPath, "//", "/")
	video.FilePath = strings.ReplaceAll(video.FilePath, "//", "/")

	// Do the same with \
	video.ThumbnailPath = strings.ReplaceAll(video.ThumbnailPath, "\\", "/")
	video.MetadataPath = strings.ReplaceAll(video.MetadataPath, "\\", "/")
	video.FilePath = strings.ReplaceAll(video.FilePath, "\\", "/")

	var updateVidMeta bool

	if video.Title != originalVideo.Title {
		fmt.Printf("Video title changed from %v to %v\n", originalVideo.Title, video.Title)
		updateVidMeta = true
	}
	if video.Description != originalVideo.Description {
		fmt.Printf("Video description changed from %v to %v\n", originalVideo.Description, video.Description)
		updateVidMeta = true
	}
	if video.Length != originalVideo.Length {
		fmt.Printf("Video length changed from %v to %v\n", originalVideo.Length, video.Length)
		updateVidMeta = true
	}
	if video.PublishedAt != originalVideo.PublishedAt {
		fmt.Printf("Video published at changed from %v to %v\n", originalVideo.PublishedAt, video.PublishedAt)
		updateVidMeta = true
	}
	if video.ThumbnailPath != originalVideo.ThumbnailPath {
		fmt.Printf("Video thumbnail path changed from %v to %v\n", originalVideo.ThumbnailPath, video.ThumbnailPath)
		updateVidMeta = true
	}
	if video.MetadataPath != originalVideo.MetadataPath {
		fmt.Printf("Video metadata path changed from %v to %v\n", originalVideo.MetadataPath, video.MetadataPath)
		updateVidMeta = true
	}
	if video.Availability != originalVideo.Availability {
		fmt.Printf("Video availability changed from %v to %v\n", originalVideo.Availability, video.Availability)
		updateVidMeta = true
	}
	if video.AgeRestricted != originalVideo.AgeRestricted {
		fmt.Printf("Video age restricted changed from %v to %v\n", originalVideo.AgeRestricted, video.AgeRestricted)
		updateVidMeta = true
	}
	if video.Views != strconv.Itoa(info.ViewCount) {
		video.Views = strconv.Itoa(info.ViewCount)
		// if nothing else has changed, update the video
		if !updateVidMeta {
			err = database.UpdateVideo(video, db)
			if err != nil {
				return err
			}
		}
	}
	if video.ChannelID == "" {
		// update the channel id
		video.ChannelID = info.ChannelID
		updateVidMeta = true
	}
	if video.ChannelTitle == "" {
		// update the channel title
		video.ChannelTitle = info.Uploader
		updateVidMeta = true
	}

	// convert genres to json string
	catJSON, err := json.Marshal(info.Categories)
	if err != nil {
		return err
	}
	// convert tags to json string
	tagsJSON, err := json.Marshal(info.Tags)
	if err != nil {
		return err
	}

	// if the categories are different, update the video
	if video.Categories != string(catJSON) {
		video.Categories = string(catJSON)
		updateVidMeta = true
	}

	// if the tags are different, update the video
	if video.Tags != string(tagsJSON) {
		video.Tags = string(tagsJSON)
		updateVidMeta = true
	}

	// check if commentsPath ends in .jsonl or missing, if so just download the comments and update the comments path
	if strings.HasSuffix(video.CommentsPath, ".jsonl") || (video.CommentsPath == "" && info.CommentCount != 0) {
		// Download comments
		if !updateVidMeta {
			newCommPath, err := downloadComments(video.VideoID)
			if err != nil {
				return err
			}
			// if the comments path is different, update the comments path
			if newCommPath != video.CommentsPath {
				// replace the youtube default path
				video.CommentsPath = strings.ReplaceAll(newCommPath, settings.BaseYouTubePath, "")
				// replace \ with / in the these paths and // with /
				video.CommentsPath = strings.ReplaceAll(video.CommentsPath, "//", "/")
				// Do the same with \
				video.CommentsPath = strings.ReplaceAll(video.CommentsPath, "\\", "/")

				err = database.UpdateVideo(video, db)
				if err != nil {
					return err
				} else {
					return nil
				}
			}
		}
	}

	// if subtitles path is missing but the video has subtitles, download the subtitles
	if video.SubtitlePath == "" && (info.Subtitles.En != nil || info.Subtitles.Es != nil || info.Subtitles.De != nil || info.Subtitles.Fr != nil) {
		// Download subtitles
		if !updateVidMeta {
			// set the input subtitle path to the ${settings.BaseYouTubePath}/${video.FilePath}/../subtitles/${video.FilePath base name without extension}/
			inputSubPath, err := general.SanitizeFilePath(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/subtitles", strings.TrimSuffix(filepath.Base(video.FilePath), filepath.Ext(video.FilePath))))
			if err != nil {
				return err
			}

			newSubs, err := downloadSubtitles(inputSubPath, &info)
			if err != nil {
				return err
			}
			// convert newSubs to json
			newSubsJSON, err := json.Marshal(newSubs)
			if err != nil {
				return err
			}
			// if the subtitles path is different, update the subtitles path
			if string(newSubsJSON) != video.SubtitlePath {
				// replace the youtube default path
				video.SubtitlePath = strings.ReplaceAll(string(newSubsJSON), settings.BaseYouTubePath, "")

				// replace \ with / in the these paths and // with /
				video.SubtitlePath = strings.ReplaceAll(video.SubtitlePath, "//", "/")
				// Do the same with \
				video.SubtitlePath = strings.ReplaceAll(video.SubtitlePath, "\\", "/")

				err = database.UpdateVideo(video, db)
				if err != nil {
					return err
				}
			}
		}
	}

	// Check db for sponsor segments for this video. If none are found, download them
	if !updateVidMeta {
		// get sponsor segments from db
		sponsorSegments, err := database.GetVideoSponsorBlock(video.VideoID, db)
		if err != nil {
			return err
		}
		// Set pubTime to the video's published time in time.Time. "2023-05-25T00:00:00Z" is the input
		pubTime, err := time.Parse("2006-01-02T15:04:05Z", video.PublishedAt)
		if err != nil {
			return err
		}

		sponsorBlockPath, err := general.SanitizeFilePath(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/sponsorblock", strings.TrimSuffix(filepath.Base(video.FilePath), filepath.Ext(video.FilePath))+".json"))
		if err != nil {
			return err
		}

		if time.Since(pubTime) < 30*24*time.Hour || len(sponsorSegments) == 0 {
			// Download sponsor segments
			_, err = downloadSponsorBlockSegments(video.VideoID, sponsorBlockPath)
			if err != nil {
				fmt.Printf("Error downloading sponsor block segments for %v: %v\n", video.VideoID, err) // don't return error, just print it
			}
		}

		// If over 7 days old and not updated, update comments and set updated to true
		if time.Since(pubTime) > 7*24*time.Hour && !video.Updated {
			// Download comments
			newCommPath, err := downloadComments(video.VideoID)
			if err != nil {
				return err
			}
			// Update video comments path and set updated to true
			video.CommentsPath = strings.ReplaceAll(newCommPath, settings.BaseYouTubePath, "")

			err = database.UpdateVideo(video, db)
			if err != nil {
				return err
			}
			fmt.Printf("Updated video comments for %v\n", video.VideoID)
			// Also update sponsor segments at this time to ensure they are up to date
			_, err = downloadSponsorBlockSegments(video.VideoID, sponsorBlockPath)
			if err != nil {
				fmt.Printf("Error downloading sponsor block segments for %v: %v\n", video.VideoID, err) // don't return error, just print it
			}
			video.Updated = true
		}
	}

	// Check if the video has changed, if so update the database
	if updateVidMeta {
		fmt.Printf("Updating video metadata for %v\n", video.Title)

		// Make metadata folder for the video filepath.Dir(video.FilePath)../metadata/metadata/

		err = os.MkdirAll(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/metadata"), 0755)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/thumbnails"), 0755)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/subtitles"), 0755)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/sponsorblock"), 0755)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/comments"), 0755)
		if err != nil {
			return err
		}

		// Update video subtitles if they are missing
		if video.SubtitlePath == "" && (info.Subtitles.En != nil || info.Subtitles.EnUS != nil || info.Subtitles.EnGB != nil || info.Subtitles.De != nil || info.Subtitles.Es != nil || info.Subtitles.Fr != nil) {
			// Download subtitles
			// set the input subtitle path to the ${settings.BaseYouTubePath}/${video.FilePath}/../subtitles/${video.FilePath base name without extension}/
			inputSubPath, err := general.SanitizeFilePath(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/subtitles", strings.TrimSuffix(filepath.Base(video.FilePath), filepath.Ext(video.FilePath))))
			if err != nil {
				return err
			}

			newSubs, err := downloadSubtitles(inputSubPath, &info)
			if err != nil {
				return err
			}
			// convert newSubs to json
			newSubsJSON, err := json.Marshal(newSubs)
			if err != nil {
				return err
			}
			// set subtitles path to the new subtitles path
			video.SubtitlePath = strings.ReplaceAll(string(newSubsJSON), settings.BaseYouTubePath, "")
		}

		sponsorBlockPath, err := general.SanitizeFilePath(filepath.Join(settings.BaseYouTubePath, filepath.Dir(video.FilePath), "../metadata/sponsorblock", strings.TrimSuffix(filepath.Base(video.FilePath), filepath.Ext(video.FilePath))+".json"))
		if err != nil {
			return err
		}

		// Download sponsorblock segments
		_, err = downloadSponsorBlockSegments(video.VideoID, sponsorBlockPath)
		if err != nil {
			fmt.Printf("Error downloading sponsorblock segments for %v: %v\n", video.Title, err) // Change to log error
		}

		// Download comments
		newCommPath, err := downloadComments(video.VideoID)
		if err == nil {
			// if the comments path is different, update the comments path
			if newCommPath != video.CommentsPath {
				video.CommentsPath = strings.ReplaceAll(newCommPath, settings.BaseYouTubePath, "")
			}
		} // We'll just ignore the error here. If the comments fail to download, we'll just leave the old comments path

		video.Views = strconv.Itoa(info.ViewCount)

		// save info to the disk. Save this to the video.MetadataPath location. Rename the old file to video.MetadataPath.00n where n increments if the file already exists
		var metadataNum int
		for {
			// Check if the file exists

			baseMeta := filepath.Base(video.ThumbnailPath)
			// Remove ALL extensions
			for {
				ext := filepath.Ext(baseMeta)
				// I know this looks dumb, I tried to just kill all extensions but it, on rare occasions, also include the . in the filename. And sometimes not. So here we are.
				if ext == ".json" || ext == ".mp4" || ext == ".ts" || ext == ".jpg" || ext == ".png" || ext == ".webp" || strings.Contains(ext, ".00") || strings.Contains(ext, ".01") || strings.Contains(ext, ".02") {
					// remove extension
					baseMeta = strings.TrimSuffix(baseMeta, ext)
				} else {
					break
				}
			}

			newPath := fmt.Sprintf("%v/%v/%v.%03d%v", settings.BaseYouTubePath, filepath.Dir(video.MetadataPath), baseMeta, metadataNum, filepath.Ext(video.MetadataPath))

			_, err := os.Stat(newPath)
			if err != nil {
				// If the file doesn't exist, save the file
				// convert info to JSON
				// Drop unnecessary fields
				info.Formats = nil
				info.Thumbnails = nil
				info.Comments = nil
				info.RequestedFormats = nil
				info.Filename = ""
				info.Filename0 = ""

				data, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return err
				}

				newPath = filepath.Clean(newPath)

				// Create the parent directory if it doesn't exist
				err = os.MkdirAll(filepath.Dir(newPath), 0755)
				if err != nil {
					return err
				}

				err = os.WriteFile(newPath, []byte(data), 0644)
				if err != nil {
					return err
				}
				// update the video metadata path and remove the base youtube path
				newPath = strings.ReplaceAll(video.MetadataPath, settings.BaseYouTubePath, "")

				video.MetadataPath = newPath

				break
			}
			metadataNum++
		}
		// if the channel id or channel title is empty, update the channel id and channel title
		if video.ChannelID == "" {
			video.ChannelID = info.ChannelID
		}
		if video.ChannelTitle == "" {
			video.ChannelTitle = info.Uploader
		}

		// if still empty, error
		if video.ChannelID == "" {
			return fmt.Errorf("channel id is empty")
		} else if video.ChannelTitle == "" {
			return fmt.Errorf("channel title is empty")
		}

		err = database.UpdateVideo(video, db)
		if err != nil {
			return err
		}

		fmt.Printf("Updated video metadata for: %v\n", video.VideoID)
	} //else {
	//fmt.Printf("Video %v has not changed\n", videoID)  // Change to debug
	//}

	var oldThumbPath string

	if video.ThumbnailPath != "" {
		oldThumbPath = fmt.Sprintf("%v/%v", settings.BaseYouTubePath, video.ThumbnailPath)
	} else {
		oldThumbPath = ""
	}

	// Download thumbnail
	newThumbPath, err := downloadThumbnail(info.Thumbnail, oldThumbPath, video.VideoID)
	if err != nil && !strings.Contains(err.Error(), "thumbnails are the same") {
		return err
	}

	// If video is older than a week old, set updated to true
	if video.PublishedAt != "" {
		// Set pubTime to the video's published time in time.Time. "2023-05-25T00:00:00Z" is the input
		pubTime, err := time.Parse("2006-01-02T15:04:05Z", video.PublishedAt)
		if err != nil {
			return err
		}
		if time.Since(pubTime) > 7*24*time.Hour && !video.Updated {
			video.Updated = true
		} else if !video.Updated {
			// Ensure it's not null
			video.Updated = false
		}
	}

	video.ThumbnailPath = strings.ReplaceAll(newThumbPath, settings.BaseYouTubePath, "")
	err2 := database.UpdateVideo(video, db)
	if err2 != nil {
		return err2
	}

	if err == nil {
		fmt.Printf("Updated video thumbnail for: %v\n", video.VideoID)
	}

	return nil
}

// function to update metadata for all videos in the database
func updateAllVideoMetadata() error {
	// Get all videos from the database
	videos, err := database.GetAllVideos(db.Select("video_id").Where("video_type != ?", "Twitch").Order("created_at desc"))
	if err != nil {
		return err
	}
	var ret error
	// For each video, update the metadata
	for _, video := range videos {
		// if video id is an int, skip it as this is a twitch video
		_, err := strconv.Atoi(video.VideoID)
		if err == nil {
			continue
		}
		err = updateVideoMetadata(video.VideoID)
		if err != nil {
			ret = err
			fmt.Printf("Error updating video %v: %v\n", video.VideoID, err)
			continue
		}
		// if ./data/tmp/$videoID.* exists, delete it
		files, err := os.ReadDir("./data/tmp")
		if err != nil {
			return err
		}
		for _, file := range files {
			if strings.HasPrefix(file.Name(), video.VideoID) {
				err = os.Remove(fmt.Sprintf("./data/tmp/%v", file.Name()))
				if err != nil {
					return err
				}
			}
		}
	}
	return ret
}

func GetCreatorMetadata(creatorID string) (database.YoutubePlaylistStruct, error) {
	// Get creator json from https://www.youtube.com/channel/$creatorID/about from yt-dlp. Export to stdout and unmarshal into a Creator struct
	cmd := exec.Command("yt-dlp", "--dump-single-json", fmt.Sprintf("https://www.youtube.com/channel/%v/about", creatorID))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return database.YoutubePlaylistStruct{}, err
	}

	var data database.YoutubePlaylistStruct
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		return database.YoutubePlaylistStruct{}, err
	}
	return data, nil
}

func getNewCreator(creatorID string) (database.Creator, error) {

	// Check if creatorid exists, if so skip
	dbcreator, err := database.GetCreator(creatorID, db)
	if err == nil {
		return dbcreator, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Creator{}, err
	}

	creator := database.Creator{}
	data, err := GetCreatorMetadata(creatorID)
	if err != nil {
		return database.Creator{}, err
	}

	creator.ChannelID = creatorID
	creator.Name = data.Uploader
	creator.Description = data.Description
	creator.Subscribers = strconv.Itoa(data.ChannelFollowerCount)
	creator.Platform = "YouTube"

	creatorName, err := general.SanitizeFileName(creator.Name)
	if err != nil {
		return database.Creator{}, err
	}
	creatorPath := fmt.Sprintf("%v/%v/", settings.BaseYouTubePath, creatorName)

	metaPath := fmt.Sprintf("%v/%v.json", creatorPath, creatorName)
	thumbPath := fmt.Sprintf("%v/avatar.png", creatorPath)
	bannerPath := fmt.Sprintf("%v/banner.png", creatorPath)

	creator.FilePath = strings.ReplaceAll(metaPath, settings.BaseYouTubePath, "")

	var thumbUrl string
	var bannerUrl string
	var setBanner bool
	var setThumb bool
	// Get uncropped thumbnail and banner
	for _, thumb := range data.Thumbnails {
		if thumb.URL == "" || setBanner || setThumb {
			continue
		}
		if thumb.ID == "avatar_uncropped" {
			thumbUrl = thumb.URL
			setThumb = true
		}

		if thumb.ID == "banner_uncropped" {
			bannerUrl = thumb.URL
		}
	}

	// Download thumbnail

	if thumbUrl != "" {
		thmb, err := downloadThumbnail(thumbUrl, thumbPath, "avatar")
		if err != nil {
			fmt.Printf("Error downloading thumbnail for %v: %v\n", creator.Name, err)
		} else {
			creator.ThumbnailPath = strings.ReplaceAll(thmb, settings.BaseYouTubePath, "")
		}
	}

	// Download banner

	if bannerUrl != "" {
		thmb, err := downloadThumbnail(bannerUrl, bannerPath, "banner")
		if err != nil {
			fmt.Printf("Error downloading banner for %v: %v\n", creator.Name, err)
		} else {
			creator.BannerPath = strings.ReplaceAll(thmb, settings.BaseYouTubePath, "")
		}
	}

	// Save data to file as indented json
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return database.Creator{}, err
	}

	// save to metaPath
	err = os.WriteFile(metaPath, jsonData, 0644)
	if err != nil {
		return database.Creator{}, err
	}

	// Insert creator intor database
	err = database.InsertCreator(creator, db)
	if err != nil {
		return database.Creator{}, err
	}

	return creator, nil

}

func loadImage(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if strings.HasSuffix(filename, ".webp") {
		img, err := webp.Decode(file)
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// download youtube comments
func downloadComments(videoID string) (string, error) {
	fmt.Printf("Downloading comments for video %v\n", videoID)
	// Make sure video exists
	video, err := database.GetVideo(videoID, db)
	if err != nil {
		return "", err
	}

	// Get comment path. If one doesn't exist, create it (${settings.BaseYouTubePath}/${video.FilePath}/../metadata/comments/)
	// make the file name the file name of ${video.FilePath} and replace the extension with .json
	commentPath, err := general.SanitizeFilePath(fmt.Sprintf("%v/%v/../metadata/comments/%v", settings.BaseYouTubePath, filepath.Dir(video.FilePath), strings.TrimSuffix(filepath.Base(video.FilePath), filepath.Ext(video.FilePath))+".json"))
	if err != nil {
		return "", err
	}

	// Create the directory if it doesn't exist
	err = os.MkdirAll(filepath.Dir(commentPath), 0755)
	if err != nil {
		return "", err
	}

	// Get the metadata and include comments
	vidUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%v", videoID)
	data, err := GetYouTubeMetadata(vidUrl, true)
	if err != nil && !strings.Contains(err.Error(), "warning") {
		// if the availability status is available, change it to unavailable and update the database
		if video.Availability == "available" {
			video.Availability = "unavailable"
			err = database.UpdateVideo(video, db)
			if err != nil {
				return "", err
			}
		}
		return "", err
	} else if err != nil && strings.Contains(err.Error(), "warning") {
		// return nothing for now
		return "", nil
	}

	// if comments are empty, return
	if len(data.Comments) == 0 {
		return "", nil
	}

	// Convert the comments to json
	jsonData, err := json.MarshalIndent(data.Comments, "", "  ")
	if err != nil {
		return "", err
	}

	// Write the comments to the file
	err = os.WriteFile(commentPath, jsonData, 0644)
	if err != nil {
		return "", err
	}

	// Sort the comments by likes
	sort.Slice(data.Comments, func(i, j int) bool {
		return data.Comments[i].LikeCount > data.Comments[j].LikeCount
	})

	sanitizedCommentPath := strings.ReplaceAll(commentPath, settings.BaseYouTubePath, "")

	// Check the first 20 comments. Skip any replies. If any of the comments are not in the database, add them
	var i int
	for _, comment := range data.Comments {
		if comment.Parent != "root" {
			continue
		}

		var newComment database.Comment
		newComment.CommentID = comment.ID
		newComment.VideoID = videoID
		newComment.Text = comment.Text
		newComment.Author = comment.Author
		newComment.Votes = strconv.Itoa(comment.LikeCount)
		newComment.TimeParsed = float64(comment.Timestamp)
		newComment.TimeString = comment.TimeText
		newComment.FilePath = sanitizedCommentPath

		_, err := database.GetComment(comment.ID, db)
		if err != nil {
			// Comment doesn't exist in the database, add it
			err = database.InsertComment(newComment, db)
			if err != nil {
				return "", err
			}
			fmt.Printf("Added comment '%v' to video id '%v' in database\n", comment.ID, videoID) // Make Debug Logging
		} else {
			// Update the comment
			err = database.UpdateComment(newComment, db)
			if err != nil {
				return "", err
			}
			fmt.Printf("Updated comment '%v' in video id '%v' in database\n", comment.ID, videoID) // Make Debug Logging
		}
		i++ // Increment i even if we skip the comment, we just want 20 comments overall
		if i >= 20 {
			break
		}
	}

	return sanitizedCommentPath, nil
}

func subDownload(subFile string, url string, ext string) error {

	// Create the directory if it doesn't exist
	err := os.MkdirAll(filepath.Dir(subFile), 0755)

	if err != nil {
		return err
	}
	// Make the request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check response

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// save file as-is
	err = os.WriteFile(subFile, body, 0644)
	if err != nil {
		return err
	}

	// Unmarshal the json to interface
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		// if invalid json, return
		return nil
	}

	// Marshal the json to string
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write the json to the file
	err = os.WriteFile(subFile, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil

}

func _dlSubtitle(subData []database.YouTubeApiVideoInfoStructSub, lang string, subtitlePath string) (string, error) {
	var retSub string
	for _, sub := range subData {
		if sub.URL == "" {
			continue
		}
		// if the language is not set, set it to unknown
		if lang == "" {
			lang = "und"
		}
		// Set the subFile path to ${settings.YouTubeBasePath}/${subtitlePath}/${subtitlePath last path name}.{lang}.{videoData.Subtitles.De.Ext}
		subFile, err := general.SanitizeFilePath(fmt.Sprintf("%v/%v.%v.%v", subtitlePath, filepath.Base(subtitlePath), lang, sub.Ext))
		if err != nil {
			return "", err
		}

		err = subDownload(subFile, sub.URL, sub.Ext)
		if err != nil {
			return "", err
		}
		// if vtt set retSub to subFile. This overrides previous languages.
		if sub.Ext == "vtt" {
			retSub = strings.ReplaceAll(subFile, settings.BaseYouTubePath, "")
		}
	}
	return retSub, nil
}

// download subtitles from database.YoutubeVideo.Subtitles
func downloadSubtitles(subtitlePath string, videoData *database.YouTubeVideoInfoStruct) ([]database.VidSubtitle, error) {
	var subList []database.VidSubtitle

	if videoData.Subtitles.En != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.En, "en", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "en", LanguageText: "English", FilePath: retSub})
	}

	if videoData.Subtitles.EnUS != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.EnUS, "en-US", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "en-US", LanguageText: "English (US)", FilePath: retSub})
	}

	if videoData.Subtitles.EnGB != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.EnGB, "en-GB", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "en-GB", LanguageText: "English (UK)", FilePath: retSub})
	}

	if videoData.Subtitles.Es != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.Es, "es", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "es", LanguageText: "Español", FilePath: retSub})
	}

	if videoData.Subtitles.De != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.De, "de", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "de", LanguageText: "Deutsch", FilePath: retSub})
	}

	if videoData.Subtitles.Fr != nil {
		// Download the subtitles
		retSub, err := _dlSubtitle(videoData.Subtitles.Fr, "fr", subtitlePath)
		if err != nil {
			return subList, err
		}
		subList = append(subList, database.VidSubtitle{Language: "fr", LanguageText: "Français", FilePath: retSub})
	}

	// if sublist isn't empty, log the download
	if len(subList) > 0 {
		fmt.Printf("Downloaded subtitles for video id '%v' to '%v'\n", videoData.ID, subtitlePath)
	}
	return subList, nil
}

// Download sponsorblock segments for a given video id and download json to given path
func downloadSponsorBlockSegments(videoID string, sponsorBlockPath string) (string, error) {
	// Create the directory if it doesn't exist
	err := os.MkdirAll(filepath.Dir(sponsorBlockPath), 0755)
	if err != nil {
		return "", err
	}

	// Download the sponsorblock segments
	sponsorUrl := fmt.Sprintf("https://sponsor.ajay.app/api/skipSegments?videoID=%v", videoID)
	resp, err := http.Get(sponsorUrl)
	if err != nil {
		return "", err
	}

	// Check status code
	if resp.StatusCode == 404 {
		// No sponsorblock segments found
		return "", nil
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("sponsorblock api returned status code %v", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// unmarshall body into json
	var segments []database.SponsorBlockRawApi
	err = json.Unmarshal(body, &segments)
	if err != nil {
		fmt.Printf("Error unmarshalling sponsorblock segments for video id '%v': %v\n", videoID, err)
		return "", err
	}

	// Get all the sponsorblock segments in the database for the video
	dbSegments, err := database.GetVideoSponsorBlock(videoID, db)
	if err != nil {
		// Just set dbSegments to an empty slice if there is an error
		dbSegments = []database.SponsorBlock{}
		fmt.Printf("Error getting sponsorblock segments for video id '%v': %v\n", videoID, err)
	}

	changes := false

	// Loop through the segments and add them to the database if they don't exist
	for _, segment := range segments {
		// Check if the segment is in the database
		var inDb bool
		for _, dbSegment := range dbSegments {
			if dbSegment.SegmentID == segment.UUID {
				inDb = true
				break
			}
		}

		// If the segment is not in the database, add it
		if !inDb {
			var newSegment database.SponsorBlock

			newSegment.SegmentID = segment.UUID
			newSegment.VideoID = videoID
			newSegment.Category = segment.Category
			newSegment.SegmentStart = segment.Segment[0]
			newSegment.SegmentEnd = segment.Segment[1]
			newSegment.Votes = segment.Votes
			newSegment.SegmentID = segment.UUID
			newSegment.FilePath = strings.ReplaceAll(sponsorBlockPath, settings.BaseYouTubePath, "")
			newSegment.ActionType = segment.ActionType

			err = database.InsertSponsorBlock(newSegment, db)
			if err != nil {
				return "", err
			}
			fmt.Printf("Added sponsorblock segment %v to database for video %v (%v)\n", segment.UUID, videoID, segment.Category)
			changes = true
		}
	}

	// Get all segments in the database for the video, delete any that aren't in our new list
	dbSegments, err = database.GetVideoSponsorBlock(videoID, db)
	if err != nil {
		return "", err
	}

	for _, dbSegment := range dbSegments {
		// Check if the segment is in the new list
		var inNewList bool
		for _, segment := range segments {
			if dbSegment.SegmentID == segment.UUID {
				inNewList = true
				break
			}
		}

		// If the segment is not in the new list, delete it
		if !inNewList {
			err = database.DeleteSponsorBlock(dbSegment, db)
			if err != nil {
				return "", err
			}
			fmt.Printf("Deleted sponsorblock segment %v from database for video %v\n", dbSegment.SegmentID, videoID)
			changes = true
		}
	}

	if changes {
		// Write the segments to the file
		writeData, err := json.MarshalIndent(segments, "", "  ")
		if err != nil {
			return "", err
		}
		err = os.WriteFile(sponsorBlockPath, writeData, 0644)
		if err != nil {
			return "", err
		}
	}

	return sponsorBlockPath, nil
}
