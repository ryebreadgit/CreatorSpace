package database

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

func InsertCreator(creator Creator, db *gorm.DB) error {
	// open database and check if creator exists, if not, create it
	if !ifChannelExists(creator.ChannelID, db) {
		db.Create(&creator)
		return nil
	} else {
		// update the database with the new information
		var c Creator
		db.Where("channel_id = ?", creator.ChannelID).First(&c)
		c.Platform = "YouTube"
		db.Save(&c)
	}
	return errors.New("record already exists")
}

func InsertVideo(video Video, db *gorm.DB) error {
	// open database and check if video exists, if not, create it
	if !ifVideoExists(video.VideoID, db) {
		db.Create(&video)
		return nil
	}
	return errors.New("record already exists")
}

func InsertSponsorBlock(sponsorBlock SponsorBlock, db *gorm.DB) error {
	// open database and check if SponsorBlock exists, if not, create it
	if !ifSponsorBlockExists(sponsorBlock.SegmentID, db) {
		db.Create(&sponsorBlock)
		return nil
	}
	return errors.New("record already exists")
}

func InsertComment(comment Comment, db *gorm.DB) error {
	// open database and check if comment exists, if not, create it
	if !ifCommentExists(comment.CommentID, db) {
		db.Create(&comment)
		return nil
	}
	return errors.New("record already exists")
}

func SetSettings(settings Settings, db *gorm.DB) error {
	// open database and check if settings exists, if not, create it
	var s Settings
	db.First(&s)
	if s.ID == 0 {
		db.Create(&settings)
		return nil
	}
	db.Model(&s).Updates(settings)
	return nil
}

func InsertTask(t Tasking, db *gorm.DB) error {
	// open database and check if comment exists, if not, create it
	if !ifTaskExists(t.TaskName, db) {
		db.Create(&t)
		return nil
	}
	return errors.New("record already exists")
}

func SetTaskEpochLastRan(task Tasking, db *gorm.DB) error {
	// open database and check if comment exists, if not, create it
	if !ifTaskExists(task.TaskName, db) {
		return errors.New("record does not exist")
	}
	db.Find(&task) // get the record
	db.Model(&task).Update("epoch_last_ran", task.EpochLastRan)

	return nil
}

func insertUser(user User, db *gorm.DB) error {
	// open database and check if user exists, if not, create it
	data, err := GetUserByName(user.Username, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if data != (User{}) {
		return errors.New("username already exists")
	}

	data, err = GetUserByID(user.UserID, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if data != (User{}) {
		return errors.New("user id already exists")
	}

	db.Create(&user)
	return nil
}

func SignupUser(user User, db *gorm.DB) error {
	// rule out empty fields
	if user.Username == "" || user.Password == "" {
		return errors.New("empty fields")
	}
	// check if username exists
	data, err := GetUserByName(user.Username, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if data != (User{}) {
		return errors.New("username already exists")
	}
	// convert password to hash
	hash, err := HashPassword(user.Password)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	user.Password = hash

	// Set the user id to a random string of 32 characters, check if it exists, if it does, generate a new one
	for {
		randomstring := RandomString(32)
		user.UserID = randomstring
		tempid, err := GetUserByID(user.UserID, db)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return err
		}
		if tempid == (User{}) {
			break
		}
	}

	// set the username to lowercase
	user.Username = strings.ToLower(user.Username)

	// Create default playlists for the user
	defaultPlaylists := []Playlist{
		{
			Name:        "Completed Videos",
			Description: "Videos you have completed",
			VideoIDs:    "[]",
			UserID:      user.UserID,
		},
		{
			Name:        "Progress",
			Description: "Videos you are currently watching",
			VideoIDs:    "[]",
			UserID:      user.UserID,
		},
		{
			Name:        "Watch Later",
			Description: "Videos you want to watch later",
			VideoIDs:    "[]",
			UserID:      user.UserID,
		},
		{
			Name:        "Subscriptions",
			Description: "Videos from your subscriptions",
			VideoIDs:    "[]",
			UserID:      user.UserID,
		},
	}

	for _, playlist := range defaultPlaylists {
		_, err := CreatePlaylist(playlist, db)
		if err != nil {
			return err
		}
	}

	if user.AccountType != "admin" {
		user.AccountType = "user"
	}

	if user.SponsorBlockCategories == "" {
		user.SponsorBlockCategories = "sponsor"
		user.SponsorBlockEnabled = true
	}

	return insertUser(user, db)
}

func UpdateUser(user User, db *gorm.DB) error {
	// open database and check if user exists, if not, create it
	if !ifUserExists(user.UserID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&user).Where("user_id=?", user.UserID).Updates(user).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserProgress(userID string, progressJsonString string, db *gorm.DB) error {

	// Get the user's progress videos playlist.
	playlists, err := GetPlaylistsByUserID(userID, db)
	if err != nil {
		return err
	}

	// check if the user has a progress videos playlist
	progVidPl := Playlist{}
	for _, v := range playlists {
		if v.Name == "Progress" {
			progVidPl = v
			break
		}
	}

	// Check progress video playlist
	if progVidPl.Name == "" {
		// Create one
		progVidPl.Name = "Progress"
		progVidPl.Description = "Videos you are currently watching"
		progVidPl.VideoIDs = progressJsonString
		progVidPl.UserID = userID
		_, err := CreatePlaylist(progVidPl, db)
		if err != nil {
			return err
		}
		return nil
	}

	progVidPl.VideoIDs = progressJsonString

	// Update the playlist
	err = UpdatePlaylist(progVidPl, db)
	if err != nil {
		return err
	}
	return nil
}

func UpdatePlaylistByUserId(userID string, playlistName string, videoIDJson string, db *gorm.DB) error {
	// Get the user's completed videos playlist.
	playlists, err := GetPlaylistsByUserID(userID, db)
	if err != nil {
		return nil
	}

	// check if the user has a completed videos playlist
	compVidPl := Playlist{}
	for _, v := range playlists {
		if v.Name == playlistName {
			compVidPl = v
			break
		}
	}

	// Check completed video playlist
	if compVidPl.Name == "" {
		return errors.New("playlist does not exist")
	}

	compVidPl.VideoIDs = videoIDJson
	// Update the playlist
	err = UpdatePlaylist(compVidPl, db)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserSponsorblockEnabled(userID string, sponsorblockEnabled bool, db *gorm.DB) error {
	// open database and check if user exists, if not, create it
	var user User
	db.First(&user, "user_id = ?", userID)
	if user == (User{}) {
		return errors.New("user does not exist")
	}
	db.Model(&user).Update("sponsor_block_enabled", sponsorblockEnabled)
	return nil
}

func UpdateTaskEpoch(task *Tasking, db *gorm.DB) error {
	// open database and check if task exists. if not, error
	if !ifTaskExists(task.TaskName, db) {
		return errors.New("record does not exist")
	}

	err := db.Model(&task).Update("epoch", task.Epoch).Error
	if err != nil {
		return err
	}

	return nil
}

func UpdateTaskEpochByName(taskName string, epoch int64, db *gorm.DB) error {
	// open database and check if task exists. if not, error
	var task Tasking
	db.First(&task, "task_name = ?", taskName)
	if task == (Tasking{}) {
		return errors.New("record does not exist")
	}

	err := db.Model(&task).Update("epoch", epoch).Error
	if err != nil {
		return err
	}

	return nil
}

func UpdateTaskEpochLastRanByName(taskName string, epoch int64, db *gorm.DB) error {
	// open database and check if task exists. if not, error
	var task Tasking
	db.First(&task, "task_name = ?", taskName)
	if task == (Tasking{}) {
		return errors.New("record does not exist")
	}

	err := db.Model(&task).Update("epoch_last_ran", epoch).Error
	if err != nil {
		return err
	}

	return nil
}

func UpdateVideo(video Video, db *gorm.DB) error {
	// open database and check if video exists, if not, create it
	if !ifVideoExists(video.VideoID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&video).Updates(video).Error
	if err != nil {
		return err
	}
	return nil
}

// Insert download queue item
func InsertDownloadQueueItem(item DownloadQueue, db *gorm.DB) error {
	// Use GetDownloadQueueItem to see if the item already exists
	_, err := GetDownloadQueueItem(item.VideoID, item.VideoType, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// If the item does not exist, insert it
	if err == gorm.ErrRecordNotFound {
		db.Create(&item)
		return nil
	}
	return errors.New("item already exists")
}

func RemoveFromDownloadQueue(videoID string, videotype string, db *gorm.DB) error {
	// Use GetDownloadQueueItem to see if the item already exists
	item, err := GetDownloadQueueItem(videoID, videotype, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if item == (DownloadQueue{}) {
		return errors.New("item does not exist")
	}
	db.Delete(&item)
	return nil
}

func DeleteSponsorBlock(sponsorBlock SponsorBlock, db *gorm.DB) error {
	// open database and check if SponsorBlock exists, if not, create it
	if !ifSponsorBlockExists(sponsorBlock.SegmentID, db) {
		return errors.New("record does not exist")
	}
	err := db.Delete(&sponsorBlock).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateComment(comment Comment, db *gorm.DB) error {
	// open database and check if comment exists, if not, create it
	if !ifCommentExists(comment.CommentID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&comment).Where("comment_id=?", comment.CommentID).Updates(comment).Error
	if err != nil {
		return err
	}
	return nil
}

func CreatePlaylist(playlist Playlist, db *gorm.DB) (string, error) {
	// If a playlist id does not exist, generate one
	if playlist.PlaylistID == "" {
		for {
			tempID := RandomString(32)
			playlist.PlaylistID = tempID
			if !ifPlaylistExists(playlist.PlaylistID, db) {
				break
			}
		}
	}
	// open database and check if playlist exists, if not, create it
	if !ifPlaylistExists(playlist.PlaylistID, db) {
		db.Create(&playlist)
		return playlist.PlaylistID, nil
	}
	return "", errors.New("record already exists")
}

func UpdatePlaylist(playlist Playlist, db *gorm.DB) error {
	// open database and check if playlist exists, if not, create it
	if !ifPlaylistExists(playlist.PlaylistID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&playlist).Where("playlist_id=?", playlist.PlaylistID).Updates(playlist).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateCreator(creator Creator, db *gorm.DB) error {
	// open database and check if creator exists, if not, create it
	if !ifChannelExists(creator.ChannelID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&creator).Where("channel_id=?", creator.ChannelID).Updates(creator).Error
	if err != nil {
		return err
	}
	return nil
}

func InsertTweet(tweet Tweet, db *gorm.DB) error {
	// open database and check if tweet exists, if not, create it
	if !ifTweetExists(tweet.TweetID, db) {
		db.Create(&tweet)
		return nil
	}
	return errors.New("record already exists")
}

func UpdateTweet(tweet Tweet, db *gorm.DB) error {
	// open database and check if tweet exists, if not, create it
	if !ifTweetExists(tweet.TweetID, db) {
		return errors.New("record does not exist")
	}
	err := db.Model(&tweet).Where("tweet_id=?", tweet.TweetID).Updates(tweet).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateDownloadQueueItem(item DownloadQueue, altVidID string, db *gorm.DB) error {
	// open database and check if item exists, if not, create it
	vidID := item.VideoID
	if !ifDownloadQueueItemExists(vidID, item.VideoType, db) {
		vidID = altVidID
		if !ifDownloadQueueItemExists(vidID, item.VideoType, db) {
			return errors.New("record does not exist")
		}
	}
	err := db.Model(&item).Where("video_id=? AND video_type=?", vidID, item.VideoType).Updates(item).Error
	if err != nil {
		return err
	}
	return nil
}
