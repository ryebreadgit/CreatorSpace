package database

import (
	"errors"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func ifVideoExists(id string, db *gorm.DB) bool {
	// check if id exists
	var v Video

	if err := db.Select("video_id").Where("video_id = ?", id).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("Video_id record #{id} does not exist")
			return false
		} else {
			log.Errorf("Error, unable to get video_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if v.VideoID != "" {
		return true
	} else {
		return false
	}
}

func ifChannelExists(id string, db *gorm.DB) bool {
	// check if id exists
	var c Creator

	if err := db.Select("channel_id").Where("channel_id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("channel_id record #{id} does not exist")
		} else {
			log.Errorf("Error, unable to get channel_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if c.ChannelID != "" {
		return true
	} else {
		return false
	}
}

func ifCommentExists(id string, db *gorm.DB) bool {
	// check if id exists
	var co Comment

	if err := db.Select("comment_id").Where("comment_id = ?", id).First(&co).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("comment_id record #{id} does not exist")
		} else {
			log.Errorf("Error, unable to get comment_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if co.CommentID != "" {
		return true
	} else {
		return false
	}
}

func ifSponsorBlockExists(id string, db *gorm.DB) bool {
	// check if id exists
	var s SponsorBlock

	if err := db.Select("segment_id").Where("segment_id = ?", id).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("segment_id record #{id} does not exist")
			return false
		} else {
			log.Errorf("Error, unable to get segment_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if s.SegmentID != "" {
		return true
	} else {
		return false
	}
}

func ifTaskExists(name string, db *gorm.DB) bool {
	// check if id exists
	var s Tasking

	if err := db.Select("task_name").Where("task_name = ?", name).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("name record #{id} does not exist")
			return false
		} else {
			log.Errorf("Error, unable to get name record #{id} record due to error: %v", err)
			return true
		}
	}
	if s.TaskName != "" {
		return true
	} else {
		return false
	}
}

func ifPlaylistExists(id string, db *gorm.DB) bool {
	// check if id exists
	var p Playlist

	if err := db.Select("playlist_id").Where("playlist_id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("playlist_id record #{id} does not exist")
			return false
		} else {
			log.Errorf("Error, unable to get playlist_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if p.PlaylistID != "" {
		return true
	} else {
		return false
	}
}

// Use bcrypt to hash a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// Use bcrypt to compare a password and a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func RandomString(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func ifCommentsExist(file_path string, db *gorm.DB) bool {
	// check if id exists, limit to 1
	var co Comment

	// try to get the comment_id from the file_path, Example name is "${video_name} ($video_id).jsonl"

	vidName := filepath.Base(file_path)

	// remove the extension
	vidName = strings.TrimSuffix(vidName, filepath.Ext(vidName))
	// get video_id by getting last instance of (
	vidId := vidName[strings.LastIndex(vidName, "(")+1:]
	// remove up to the last )
	vidId = vidId[:strings.LastIndex(vidId, ")")]

	// check for video_id
	if err := db.Select("comment_id").Where("video_id = ?", vidId).First(&co).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("comment_id record #{id} does not exist, checking file_path")
		} else {
			log.Debugf("Error, unable to get comment_id record #{id} record due to error: %v", err)
		}
	}

	// check if we got a comment_id
	if co.CommentID != "" {
		return true
	}

	// try the file_path

	if err := db.Select("comment_id, file_path").Where("file_path = ?", file_path).First(&co).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("comment_id record #{id} does not exist")
			return false
		} else {
			log.Errorf("Error, unable to get comment_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if co.CommentID != "" {
		return true
	} else {
		return false
	}
}

func ConvertVote(vote string) (int, error) {
	// check if has K, M, B in it. If so, convert to full number. This will be a float originally, so we need to convert to int
	if vote == "" {
		return 0, nil
	}

	decPresent := false
	// Check if a decimal is present. If so, convert to int
	if strings.Contains(vote, ".") {
		// only allow one decimal place
		if strings.Count(vote, ".") > 1 {
			// truncate to one decimal place
			vote = vote[:strings.Index(vote, ".")+2]
		}
		vote = strings.Replace(vote, ".", "", -1)
		decPresent = true
	}

	// Check if K is present
	if strings.Contains(vote, "K") {
		// If decimal is present, fill in the zeros
		if decPresent {
			vote = strings.Replace(vote, "K", "00", -1)
		} else {
			vote = strings.Replace(vote, "K", "000", -1)
		}
	} else if strings.Contains(vote, "M") {
		// If decimal is present, fill in the zeros
		if decPresent {
			vote = strings.Replace(vote, "M", "00000", -1)
		} else {
			vote = strings.Replace(vote, "M", "000000", -1)
		}
	} else if strings.Contains(vote, "B") {
		// If decimal is present, fill in the zeros
		if decPresent {
			vote = strings.Replace(vote, "B", "00000000", -1)
		} else {
			vote = strings.Replace(vote, "B", "000000000", -1)
		}
	}

	// convert string to int
	vote_int, err := strconv.Atoi(vote)
	if err != nil {
		log.Errorf("Error converting vote to int: %v", err)
	}

	return vote_int, nil

}

func getVideoLength(filePath string) (string, error) {
	// get video length
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	out, err := cmd.Output()
	if err != nil {
		log.Errorf("Error getting video length: %v", err)
		return "", err
	}
	// get in seconds
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return "", err
	}

	return strconv.FormatFloat(seconds, 'f', 0, 64), nil

}

func ifTweetExists(id string, db *gorm.DB) bool {
	// check if id exists
	var t Tweet

	if err := db.Select("tweet_id").Where("tweet_id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("tweet_id record #{id} does not exist")
		} else {
			log.Errorf("Error, unable to get tweet_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if t.TweetID != "" {
		return true
	} else {
		return false
	}
}

func ifDownloadQueueItemExists(id string, vidType string, db *gorm.DB) bool {
	// check if id exists
	var d DownloadQueue

	if err := db.Select("video_id").Where("video_id = ? AND video_type = ?", id, vidType).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("video_id record #{id} does not exist")
		} else {
			log.Errorf("Error, unable to get video_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if d.VideoID != "" {
		return true
	} else {
		return false
	}
}

func ifUserExists(id string, db *gorm.DB) bool {
	// check if id exists
	var u User

	if err := db.Select("user_id").Where("user_id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Debugf("user_id record #{id} does not exist")
		} else {
			log.Errorf("Error, unable to get user_id record #{id} record due to error: %v", err)
			return true
		}
	}
	if u.UserID != "" {
		return true
	} else {
		return false
	}
}

func GetValidUserTypes() []string {
	return []string{"admin", "user", "api"}
}
