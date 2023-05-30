package database

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
)

func GetCreator(channelid string, db *gorm.DB) (Creator, error) {
	// check if creator exists
	var c Creator
	err := db.Where("channel_id = ?", channelid).First(&c).Error
	if err != nil {
		return Creator{}, err
	}
	return c, nil
}

func GetVideo(video string, db *gorm.DB) (Video, error) {
	// check if video exists
	var v Video
	err := db.Where("video_id = ?", video).First(&v).Error
	if err != nil {
		return Video{}, err
	}
	return v, nil
}

func GetCreatorVideos(creator string, db *gorm.DB) ([]Video, error) {
	// get videos from creator

	dbargs := db.Order("published_at desc, title")
	if creator != "000" {
		dbargs = dbargs.Where("channel_id = ?", creator)
	}

	var videos []Video
	err := dbargs.Find(&videos).Error
	if err != nil {
		return nil, err
	}

	return videos, nil
}

func GetVideoComments(video string, db *gorm.DB) ([]Comment, error) {
	// get comments from video
	var comments []Comment
	err := db.Where("video_id = ?", video).Find(&comments).Error
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func GetVideoSubtitles(video string, db *gorm.DB) ([]VidSubtitle, error) {
	// get subtitles from video
	var vidTmp Video
	var vidSub []VidSubtitle
	err := db.Where("video_id = ?", video).Find(&vidTmp).Error
	if err != nil {
		return vidSub, err
	}
	if vidTmp.SubtitlePath != "" {
		// parse json into vidSub
		err = json.Unmarshal([]byte(vidTmp.SubtitlePath), &vidSub)
		if err != nil {
			return vidSub, errors.New("error parsing subtitle json")
		}

		return vidSub, nil

	} else {
		return vidSub, nil
	}
}

func GetAllVideos(db *gorm.DB) ([]Video, error) {
	// get all videos
	var videos []Video
	err := db.Order("published_at desc, title").Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func GetVideoSponsorBlock(video string, db *gorm.DB) ([]SponsorBlock, error) {
	// get sponsorblock from video
	var sponsorblock []SponsorBlock
	err := db.Where("video_id = ?", video).Find(&sponsorblock).Error
	if err != nil {
		return nil, err
	}
	return sponsorblock, nil
}

// GetVideoCommentsPaginated function to get comments from video paginated
func GetVideoCommentsPaginated(video string, db *gorm.DB, page int) []Comment {
	// get comments from video
	var comments []Comment
	db.Where("video_id = ?", video).Limit(20).Offset(page * 20).Find(&comments)
	return comments
}

func GetAllCreators(db *gorm.DB) ([]Creator, error) {
	// get all creators
	var creators []Creator
	err := db.Order("name asc").Find(&creators).Error
	if err != nil {
		return nil, err
	}
	return creators, nil
}

func GetAllVideosFromCreator(creator string, db *gorm.DB) []Video {
	// get all videos from creator
	var videos []Video
	db.Where("channel_id = ?", creator).Find(&videos)
	return videos
}

func GetSettings(db *gorm.DB) (*Settings, error) {
	// get settings
	var settings Settings
	err := db.First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func GetUserByName(username string, db *gorm.DB) (User, error) {
	// get user, only username and id
	var user User
	err := db.Select("user_id, username").Where("username = ?", username).First(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func GetUserByID(userID string, db *gorm.DB) (User, error) {
	// get user, only username and id
	var user User
	err := db.Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func LoginUser(username string, password string, db *gorm.DB) (User, error) {
	// Get invalid names out of the way
	if username == "" || password == "" {
		return User{}, nil
	}
	username = strings.ToLower(username)
	// get user and check password with bcrypt
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return User{}, err
	}
	if user.Password == "" {
		return User{}, nil
	}

	if CheckPasswordHash(password, user.Password) {
		user.Password = ""
		return user, nil
	} else {
		return User{}, nil
	}
}

func GetUsers(db *gorm.DB) ([]User, error) {
	// get all users from database, only get username and id
	var users []User
	err := db.Select("user_id, username").Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func GetSponsorBlockSettings(user_id string, db *gorm.DB) (bool, string, error) {
	// get sponsorblock SponsorBlockEnabled    bool SponsorBlockCategories string from user id
	var user User
	err := db.Where("user_id = ?", user_id).First(&user).Error
	if err != nil {
		return false, "", err
	}
	return user.SponsorBlockEnabled, user.SponsorBlockCategories, nil
}

// create GetVideoProgress(videoid, c.GetString("user_id"), db). Stored in User.Progress as a json string formatted as [{"videoID": progress}]
func GetVideoProgress(videoID string, userID string, db *gorm.DB) (string, error) {
	// open database and check if user exists, if not, create it
	prog, err := GetAllVideoProgress(userID, db)
	if err != nil {
		return "0", err
	}

	for _, v := range prog {
		if v.VideoID == videoID {
			return v.Progress, nil
		}
	}

	return "0", nil
}

func GetAllVideoProgress(userID string, db *gorm.DB) ([]ProgressToken, error) {

	// Get the user's completed videos playlist.
	playlists, err := GetPlaylistsByUserID(userID, db)
	if err != nil {
		return nil, err
	}

	// check if the user has a completed videos playlist
	progVidPl := Playlist{}
	for _, v := range playlists {
		if v.Name == "Progress" {
			progVidPl = v
			break
		}
	}

	// Check in-progress video playlist
	if progVidPl.Name == "" {
		// Create one
		progVidPl.Name = "Progress"
		progVidPl.Description = "Videos you are currently watching"
		progVidPl.VideoIDs = "[]"
		progVidPl.UserID = userID
		_, err := CreatePlaylist(progVidPl, db)
		if err != nil {
			return nil, err
		}
	}

	var progress []ProgressToken
	err = json.Unmarshal([]byte(progVidPl.VideoIDs), &progress)
	if err != nil {
		return nil, err
	}
	return progress, nil
}

func GetPlaylistByUserID(userID string, playlistName string, db *gorm.DB) ([]string, error) {

	// Get the user's completed videos playlist.
	playlists, err := GetPlaylistsByUserID(userID, db)
	if err != nil {
		return nil, err
	}

	// check if the user has a completed videos playlist
	playlist := Playlist{}
	for _, v := range playlists {
		if v.Name == playlistName {
			playlist = v
			break
		}
	}

	// Check completed video playlist
	if playlist.Name == "" {
		return nil, errors.New("playlist not found")
	}

	var videoIDs []string
	err = json.Unmarshal([]byte(playlist.VideoIDs), &videoIDs)
	if err != nil {
		return nil, err
	}
	return videoIDs, nil
}

func GetCommentPath(videoID string, db *gorm.DB) (string, error) {
	// get comment path from video
	var comment Comment
	err := db.Select("file_path").Where("video_id = ?", videoID).First(&comment).Error
	if err != nil {
		return "", err
	}
	if comment.FilePath == "" {
		return "", nil
	}
	return comment.FilePath, nil
}

func GetCreatorVideoCount(creatorID string, db *gorm.DB) (int, error) {
	// get video count from creator
	var count int64
	err := db.Model(&Video{}).Where("channel_id = ?", creatorID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func GetTask(db *gorm.DB) (*Tasking, error) {
	// get task from database
	var task Tasking
	err := db.First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func GetAllTasks(db *gorm.DB) ([]*Tasking, error) {
	var tasks []*Tasking
	err := db.Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func GetComment(comment_id string, db *gorm.DB) (Comment, error) {
	// get comment from video
	var comment Comment
	err := db.Where("comment_id = ?", comment_id).First(&comment).Error
	if err != nil {
		return comment, err
	}
	// check if comment is empty
	if comment.VideoID == "" {
		return comment, errors.New("video id is empty")
	}
	return comment, nil
}

// get DownloadQueue from database
func GetDownloadQueue(db *gorm.DB) ([]DownloadQueue, error) {
	var queue []DownloadQueue
	err := db.Find(&queue).Error
	if err != nil {
		return nil, err
	}
	return queue, nil
}

func GetDownloadQueueItem(videoID string, vidtype string, db *gorm.DB) (DownloadQueue, error) {
	var item DownloadQueue
	err := db.Where("video_id = ?", videoID).Where("video_type = ?", vidtype).First(&item).Error
	if err != nil {
		return item, err
	}
	return item, nil
}

func GetPlaylist(playlistID string, db *gorm.DB) (Playlist, error) {
	var playlist Playlist
	err := db.Where("playlist_id = ?", playlistID).First(&playlist).Error
	if err != nil {
		return playlist, err
	}
	return playlist, nil
}

func GetPlaylistsByUserID(userID string, db *gorm.DB) ([]Playlist, error) {
	var playlists []Playlist
	err := db.Where("user_id = ?", userID).Find(&playlists).Error
	if err != nil {
		return nil, err
	}
	return playlists, nil
}
