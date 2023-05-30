package general

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func TimeConversion(durationStr string) int {

	// check if time is in the correct format
	if !strings.HasPrefix(durationStr, "PT") {
		return 0
	}

	// Remove the leading "PT" from the duration string if it exists
	durationStr = strings.TrimPrefix(durationStr, "PT")

	// if minutes are not present, then the duration is in seconds
	if !strings.Contains(durationStr, "M") {
		secondsStr := strings.TrimSuffix(durationStr, "S")
		seconds, _ := strconv.Atoi(secondsStr)
		return seconds
	}

	// if hours are not present, then the duration is in minutes and seconds
	if !strings.Contains(durationStr, "H") {

		components := strings.Split(durationStr, "M")
		minutes, _ := strconv.Atoi(components[0])
		secondsStr := strings.TrimSuffix(components[1], "S")
		seconds, _ := strconv.Atoi(secondsStr)

		// Calculate the total duration in seconds
		totalSeconds := (minutes * 60) + seconds

		return totalSeconds
	}

	// if hours are present, then the duration is in hours, minutes and seconds
	components := strings.Split(durationStr, "H")
	hours, _ := strconv.Atoi(components[0])
	components = strings.Split(components[1], "M")
	minutes, _ := strconv.Atoi(components[0])
	secondsStr := strings.TrimSuffix(components[1], "S")
	seconds, _ := strconv.Atoi(secondsStr)

	totalSeconds := (hours * 3600) + (minutes * 60) + seconds
	return totalSeconds
}

// RoundViews rounds views to K, M, or B, allowing for 1 decimal place. Take string input and return string output.
func FormatViews(views string) string {
	// Convert views to int
	viewsInt, _ := strconv.Atoi(views)

	// Round views
	if viewsInt >= 1000000000 {
		return strconv.FormatFloat(float64(viewsInt)/1000000000, 'f', 2, 64) + "B"
	} else if viewsInt >= 1000000 {
		return strconv.FormatFloat(float64(viewsInt)/1000000, 'f', 2, 64) + "M"
	} else if viewsInt >= 1000 {
		return strconv.FormatFloat(float64(viewsInt)/1000, 'f', 2, 64) + "K"
	} else {
		return views
	}
}

// FormatDuration takes a duration in seconds and returns a string in the format of HH:MM:SS. If the duration is less than 1 hour, the hours are omitted. Drop the leading 0 from the hours if it exists.
func FormatDuration(duration string) string {
	// Convert PT23M14S to seconds
	durationInt := TimeConversion(duration)
	if durationInt == 0 {
		// Convert duration to int
		durationInt, _ = strconv.Atoi(duration)
	}

	var ret string

	// Format duration
	if durationInt >= 3600 {
		ret = time.Unix(int64(durationInt), 0).UTC().Format("15:04:05")
	} else {
		ret = time.Unix(int64(durationInt), 0).UTC().Format("04:05")
	}

	// Drop the leading 0 from if it exists
	ret = strings.TrimPrefix(ret, "0")

	return ret
}

// Parse api data from metadataLink := fmt.Sprintf("https://www.youtube.com/get_video_info?video_id=%s", videoID)
func ParseYouTubePublicMetadata(data string) (map[string]string, error) {
	// parse the response data
	metadata := make(map[string]string)
	for _, pair := range strings.Split(data, "&") {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) == 2 {
			metadata[keyValue[0]] = keyValue[1]
		}
	}

	return metadata, nil
}

// parse LinkedAccountsStruct from json string
func ParseLinkedAccounts(data string) ([]LinkedAccountsStruct, error) {
	var linkedAccounts []LinkedAccountsStruct

	// parse the response data
	err := json.Unmarshal([]byte(data), &linkedAccounts)
	if err != nil {
		return linkedAccounts, err
	}

	return linkedAccounts, nil
}

func EpochToDate(epoch int64) string {
	return time.Unix(epoch, 0).Format("2006-01-02T15:04:05Z")
}

func DateToEpoch(date string) (int64, error) {
	// check if PT is present at the beginning, trim if so
	date = strings.TrimPrefix(date, "PT")
	t, err := time.Parse("2006-01-02T15:04:05Z", date)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// Check for any files in a folder that contain a string
func StringInFolder(check string, folderpath string) ([]string, error) {
	var files []string
	err := filepath.Walk(folderpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, check) {
			// append the absolute path to the files slice with filepath.Abs(path)
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, absPath)
		}
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, nil
}

// Check if string in slice
func StringInSlice(list []string, a string) bool {
	for _, b := range list {
		alower := strings.ToLower(a)
		blower := strings.ToLower(b)
		if blower == alower {
			return true
		}
	}
	return false
}

func Move(source, destination string) error {
	err := os.Rename(source, destination)
	if err != nil && strings.Contains(err.Error(), "invalid cross-device link") {
		return moveCrossDevice(source, destination)
	}
	return err
}

func moveCrossDevice(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	dst, err := os.Create(destination)
	if err != nil {
		src.Close()
		return err
	}
	_, err = io.Copy(dst, src)
	src.Close()
	dst.Close()
	if err != nil {
		return err
	}
	fi, err := os.Stat(source)
	if err != nil {
		os.Remove(destination)
		return err
	}
	err = os.Chmod(destination, fi.Mode())
	if err != nil {
		os.Remove(destination)
		return err
	}
	os.Remove(source)
	return nil
}

func RestartSelf() {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to restart: %s", err)
	}

	// Ensure that we don't return before the restart has had a chance to take effect.
	os.Exit(0)
}
