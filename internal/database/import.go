package database

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ryebreadgit/CreatorSpace/internal/general"
	"gorm.io/gorm"
)

func importVideoMetadata(filePath string, thumbnailPath string, subtitlePath string, commentPath string, settings *Settings, db *gorm.DB) error {
	// open file at filePath
	f, err := os.Open(filePath)
	if err != nil {
		// return error
		return err
	}
	defer f.Close()

	// read file
	r := bufio.NewReader(f)
	b, err := io.ReadAll(r)
	if err != nil {
		// return error
		return err
	}

	// unmarshal JSON
	var v YouTubeVideoInfoStruct
	var vid Video
	err = json.Unmarshal(b, &v)
	if err != nil || v.Title == "" {
		// Try to unmarshal as a YouTubeApiVideoInfoStruct
		var y YouTubeApiVideoInfoStruct
		err = json.Unmarshal(b, &y)
		if err != nil {
			return err
		} else {

			vid = Video{
				VideoID:      y.ID,
				Title:        y.Snippet.Title,
				Description:  y.Snippet.Description,
				PublishedAt:  y.Snippet.PublishedAt,
				ChannelID:    y.Snippet.ChannelID,
				ChannelTitle: y.Snippet.ChannelTitle,
				Length:       y.ContentDetails.Duration,
				Views:        y.Statistics.ViewCount,
				Likes:        y.Statistics.LikeCount,
			}
			epoch, err := general.DateToEpoch(v.UploadDate)
			if err == nil {
				vid.Epoch = epoch
			} else {
				vid.Epoch = 0
			}
		}
	} else {
		vid = Video{
			VideoID:      v.ID,
			Title:        v.Title,
			Description:  v.Description,
			ChannelID:    v.ChannelID,
			ChannelTitle: v.Uploader,
			Length:       fmt.Sprintf("%v", v.Duration),
			Views:        strconv.Itoa(v.ViewCount),
		}

		// convert 20200130 upload date to published at 00:00:00T00:00:00Z if format is correct
		if len(v.UploadDate) == 8 {
			tempDate := fmt.Sprintf("%v-%v-%vT00:00:00Z", v.UploadDate[0:4], v.UploadDate[4:6], v.UploadDate[6:8])
			vid.PublishedAt = tempDate
		} else {
			vid.PublishedAt = v.UploadDate
			vid.Epoch = int64(v.ReleaseTimestamp)
		}

		epoch, err := general.DateToEpoch(v.UploadDate)
		if err == nil {
			vid.Epoch = epoch
		} else {
			vid.Epoch = 0
		}
	}

	// if subtitlePath exists on disk, set vid.SubtitlePath to subtitlePath
	if _, err := os.Stat(subtitlePath); err == nil {
		vid.SubtitlePath = subtitlePath
	}

	fp := filePath
	// remove all extensions from file path
	for {
		ext := filepath.Ext(fp)
		// I know this looks dumb, I tried to just kill all extensions but it, on rare occasions, also include the . in the filename. And sometimes not. So here we are.
		if ext == ".json" || ext == ".mp4" || ext == ".ts" || ext == ".jpg" || ext == ".png" || ext == ".webp" || strings.Contains(ext, ".00") || strings.Contains(ext, ".01") || strings.Contains(ext, ".02") {
			// remove extension
			fp = strings.TrimSuffix(fp, ext)
		} else {
			break
		}
	}
	// add .mp4 extension to file path
	fp += ".mp4"
	fp = strings.ReplaceAll(fp, "/metadata/metadata/", "/videos/")
	fp = strings.ReplaceAll(fp, "//", "/")
	fp = strings.ReplaceAll(fp, "\\", "/")

	// check if file exists
	if _, err := os.Stat(fp); err != nil {
		// Check fp parent folder for file with video id in name
		parentFolder := filepath.Dir(fp)
		fp = ""
		// get all files in parent folder
		files, err := os.ReadDir(parentFolder)
		if err != nil {
			return err
		}
		// loop through files
		for _, file := range files {
			// check if file has video id in name
			if strings.Contains(file.Name(), vid.VideoID) {
				// set file path to file path
				fp = filepath.Join(parentFolder, file.Name())
				break
			}
		}
	}

	if fp == "" {
		return errors.New("file path not found")
	}

	metaPath := filepath.Clean(filePath)
	// Remove all extensions from the file path
	for {
		ext := filepath.Ext(metaPath)
		// I know this looks dumb, I tried to just kill all extensions but it, on rare occasions, also include the . in the filename. And sometimes not. So here we are.
		if ext == ".json" || ext == ".mp4" || ext == ".ts" || ext == ".jpg" || ext == ".png" || ext == ".webp" || strings.Contains(ext, ".00") || strings.Contains(ext, ".01") || strings.Contains(ext, ".02") {
			// remove extension
			metaPath = strings.TrimSuffix(metaPath, ext)
		} else {
			break
		}
	}

	metaPath += ".json"

	thumbnailPath = filepath.Clean(thumbnailPath)

	// if thumbnail path doesn't exist, see if there's any thumnails with the video id in the thumbnail path parent folder
	if _, err := os.Stat(thumbnailPath); err != nil {
		// get parent folder of thumbnail path
		parentFolder := filepath.Dir(thumbnailPath)
		thumbnailPath = ""
		// get all files in parent folder
		files, err := os.ReadDir(parentFolder)
		if err != nil {
			return err
		}
		// loop through files
		for _, file := range files {
			// check if file has video id in name
			if strings.Contains(file.Name(), vid.VideoID) {
				// set thumbnail path to file path
				thumbnailPath = filepath.Join(parentFolder, file.Name())
				break
			}
		}
	}

	vid.FilePath = strings.ReplaceAll(fp, settings.BaseYouTubePath, "")
	vid.MetadataPath = strings.ReplaceAll(metaPath, settings.BaseYouTubePath, "")
	vid.ThumbnailPath = strings.ReplaceAll(thumbnailPath, settings.BaseYouTubePath, "")
	vid.CommentsPath = strings.ReplaceAll(commentPath, settings.BaseYouTubePath, "")

	// check if vid title is blank
	if vid.Title == "" {
		// return error
		return errors.New("video title is blank, unable to marshall json")
	}

	// check if video exists
	if ifVideoExists(vid.VideoID, db) {
		return nil
	}

	// import Video struct into database
	return InsertVideo(vid, db)
}

func importYouTubeCreatorMetadata(filePath string, thumbnailPath string, bannerPath string, settings *Settings, db *gorm.DB) error {
	// open file at filePath
	f, err := os.Open(filePath)
	if err != nil {
		// return error
		return err
	}
	defer f.Close()

	// read file
	r := bufio.NewReader(f)
	b, err := io.ReadAll(r)
	if err != nil {
		// return error
		return err
	}

	// unmarshal JSON
	var v map[string]interface{}
	err = json.Unmarshal(b, &v)
	if err != nil {
		// return error
		return err
	}

	// check if exists
	if ifChannelExists(v["id"].(string), db) {
		return nil
	}

	cre := Creator{
		ChannelID:   v["id"].(string),
		Name:        v["snippet"].(map[string]interface{})["title"].(string),
		Description: v["snippet"].(map[string]interface{})["description"].(string),
		VideoIDs:    fmt.Sprintf("/api/youtube/creator/%s", v["id"].(string)),
		Subscribers: v["statistics"].(map[string]interface{})["subscriberCount"].(string),
	}

	cre.FilePath = strings.ReplaceAll(filePath, "//", "/")
	cre.ThumbnailPath = strings.ReplaceAll(thumbnailPath, "//", "/")
	cre.BannerPath = strings.ReplaceAll(bannerPath, "//", "/")

	cre.FilePath = strings.ReplaceAll(filePath, "\\", "/")
	cre.ThumbnailPath = strings.ReplaceAll(thumbnailPath, "\\", "/")
	cre.BannerPath = strings.ReplaceAll(bannerPath, "\\", "/")

	cre.FilePath = strings.ReplaceAll(filePath, settings.BaseYouTubePath, "")
	cre.ThumbnailPath = strings.ReplaceAll(thumbnailPath, settings.BaseYouTubePath, "")
	cre.BannerPath = strings.ReplaceAll(bannerPath, settings.BaseYouTubePath, "")

	// import Video struct into database
	return InsertCreator(cre, db)
}

func importSponsorBlockMetadata(filePath string, settings *Settings, db *gorm.DB) error {
	// open file at filePath
	f, err := os.Open(filePath)
	if err != nil {
		// return error
		return err
	}
	defer f.Close()

	// read file
	r := bufio.NewReader(f)
	b, err := io.ReadAll(r)
	if err != nil {
		// return error
		return err
	}

	// unmarshal JSON
	var v FileJsonSponsorBlock
	err = json.Unmarshal(b, &v)
	if err != nil {
		// return error
		return err
	}

	for _, s := range v.Segments {
		spon := SponsorBlock{
			SegmentStart: s.StartTime,
			SegmentEnd:   s.EndTime,
			SegmentID:    s.UUID,
			ActionType:   s.ActionType,
			Category:     s.Category,
			Hidden:       s.Hidden,
			ShadowHidden: s.ShadowHidden,
		}
		spon.FilePath = strings.ReplaceAll(filePath, "//", "/")
		spon.FilePath = strings.ReplaceAll(filePath, "\\", "/")
		spon.FilePath = strings.ReplaceAll(filePath, settings.BaseYouTubePath, "")
		spon.VideoID = filePath[strings.LastIndex(filePath, "(")+1 : strings.LastIndex(filePath, ")")]
		// import Video struct into database
		err = InsertSponsorBlock(spon, db)
		if err != nil && err.Error() != "record already exists" {
			return err
		}
	}
	// import Video struct into database
	return nil
}

func importCommentMetadata(filePath string, settings *Settings, db *gorm.DB) error {
	// check if file exists, if not skip. Use os module
	isValid, err := os.Stat(filePath)
	if err != nil {
		return nil
	}
	if isValid.IsDir() {
		return nil
	}

	// check if any comments exist for this video, if so skip
	dbFilePath := strings.ReplaceAll(filePath, settings.BaseYouTubePath, "")
	if ifCommentsExist(dbFilePath, db) {
		return nil
	}
	// open file at filePath
	f, err := os.Open(filePath)
	if err != nil {
		// return error
		return err
	}
	defer f.Close()

	// read file
	r := bufio.NewReader(f)
	b, err := io.ReadAll(r)
	if err != nil {
		// return error
		return err
	}

	cnt := 0

	// read line by line and unmarshal JSON into map for each line
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	for scanner.Scan() {
		f := scanner.Text()
		var v Comment
		err = json.Unmarshal([]byte(f), &v)
		if err != nil {
			// return error
			return err
		}
		commid := v.CommentID
		parentid := ""
		if strings.Contains(commid, ".") {
			parentid = strings.Split(commid, ".")[0]
			commid = strings.Split(commid, ".")[1]
		}
		v.CommentID = commid
		v.ParentCommentID = parentid
		v.FilePath = strings.ReplaceAll(filePath, "//", "/")
		v.FilePath = strings.ReplaceAll(filePath, "\\", "/")
		v.FilePath = strings.ReplaceAll(filePath, settings.BaseYouTubePath, "")

		v.VideoID = filePath[strings.LastIndex(filePath, "(")+1 : strings.LastIndex(filePath, ")")]
		err = InsertComment(v, db)
		if err != nil && err.Error() != "record already exists" {
			return err
		} else if err != nil && err.Error() == "record already exists" {
			// skip remaining comments
			break
		}

		cnt++

		if cnt > 25 { // only download first 25 comments
			break
		}

	}
	return nil
}

// import creator metadata from path ${creator}/${cretor}.json and video metadata are stroed in multiple files formatted ${creator}/metadata/metadata/${video}.json
func ImportMetadata(creator string, settings *Settings, db *gorm.DB) error {
	// import creator metadata
	err := importYouTubeCreatorMetadata(fmt.Sprintf("%s/%s/%s.json", settings.BaseYouTubePath, creator, creator), fmt.Sprintf("%s/%s/avatar.png", settings.BaseYouTubePath, creator), fmt.Sprintf("%s/%s/banner.png", settings.BaseYouTubePath, creator), settings, db)
	if err != nil {
		// return error
		return err
	}
	// import video metadata
	files, err := os.ReadDir(fmt.Sprintf("%s/%s/metadata/metadata/", settings.BaseYouTubePath, creator))
	if err != nil && err.Error() != "record already exists" {
		// return error
		return err
	}

	var reterr []error

	for _, file := range files {
		// get thumbnail path which is stored in ${creator}/metadata/thumbnails/${video}.jpg replace extension with .jpg
		tempName := file.Name()[:len(file.Name())-5]
		thumbnailPath := fmt.Sprintf("%s/%s/metadata/thumbnails/%s.jpg", settings.BaseYouTubePath, creator, tempName)
		// get subtitle path which is stored in ${creator}/metadata/subtitles/${video}/${video}.vtt
		subtitlePath := fmt.Sprintf("%s/%s/metadata/subtitles/%s/%s.vtt", settings.BaseYouTubePath, creator, tempName, tempName)
		commentPath := fmt.Sprintf("%s/%s/metadata/comments/%s.jsonl", settings.BaseYouTubePath, creator, tempName)
		metadataPath := fmt.Sprintf("%s/%s/metadata/metadata/%s", settings.BaseYouTubePath, creator, file.Name())
		// check if .00n.json is in file.Name() and if so skip
		if strings.Contains(file.Name(), ".00") {
			continue
		}
		err = importVideoMetadata(metadataPath, thumbnailPath, subtitlePath, commentPath, settings, db)
		if err != nil && err.Error() != "record already exists" {
			// set reterr
			reterr = append(reterr, fmt.Errorf("error importing video metadata for %s: %s", file.Name(), err.Error()))
		}
	}

	if reterr != nil {
		for _, err := range reterr {
			fmt.Println(err.Error()) // Change to debug
		}
		return reterr[0]
	}

	return nil // TODO REMOVE (only for testing)

	reterr = nil

	// import sponsorblock metadata
	files, err = os.ReadDir(fmt.Sprintf("%s/%s/metadata/sponsorblock/", settings.BaseYouTubePath, creator))
	if err != nil {
		// return error
		reterr = append(reterr, err)
	}
	for _, file := range files {
		err = importSponsorBlockMetadata(fmt.Sprintf("%s/%s/metadata/sponsorblock/%s", settings.BaseYouTubePath, creator, file.Name()), settings, db)
		if err != nil && err.Error() != "record already exists" {
			// return error
			reterr = append(reterr, err)
		}
	}
	/*
		// import comment metadata
		files, err = os.ReadDir(fmt.Sprintf("%s/%s/metadata/comments/", settings.BaseYouTubePath, creator))
		if err != nil {
			// return error
			reterr = append(reterr, err)
		}
		for _, file := range files {
			err = importCommentMetadata(fmt.Sprintf("%s/%s/metadata/comments/%s", settings.BaseYouTubePath, creator, file.Name()), settings, db)
			if err != nil && err.Error() != "record already exists" {
				// return error
				reterr = append(reterr, err)
			}
		}
	*/
	if reterr != nil {
		for _, err := range reterr {
			fmt.Println(err.Error()) // Change to debug
		}
		return reterr[0]
	}
	return nil
}

func ImportTwitchMetadata(creator string, settings *Settings, db *gorm.DB) error {
	creator = strings.Trim(creator, "/")
	// import creator metadata
	channel_id, err := importTwitchCreatorMetadata(fmt.Sprintf("%s/%s/%s.json", settings.BaseTwitchPath, creator, creator), fmt.Sprintf("%s/%s/avatar.png", settings.BaseTwitchPath, creator), fmt.Sprintf("%s/%s/banner.png", settings.BaseTwitchPath, creator), settings, db)
	if err != nil {
		// return error
		return err
	}

	// import videos
	files, err := os.ReadDir(fmt.Sprintf("%s/%s/videos/", settings.BaseTwitchPath, creator))
	if err != nil {
		// return error
		return err
	}

	for _, file := range files {
		// ignore - Extras -
		if file.Name() == "- Extras -" {
			continue
		}
		// get thumbnail path which is stored in ${creator}/metadata/thumbnails/${video} (${id}).jpg replace extension with .jpg
		// using filepath remove the extension from the file name
		tempName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		chatPath := fmt.Sprintf("%s/%s/metadata/twitch-chat/%s.cht", settings.BaseTwitchPath, creator, tempName)
		thumbPath := fmt.Sprintf("%s/%s/metadata/thumbnails/%s.jpg", settings.BaseTwitchPath, creator, tempName)

		err = importTwitchVideo(fmt.Sprintf("%s/%s/videos/%s", settings.BaseTwitchPath, creator, file.Name()), chatPath, thumbPath, channel_id, creator, settings, db)
		if err != nil && err.Error() != "record already exists" {
			return err
		}

	}

	return nil
}

func importTwitchCreatorMetadata(filePath string, thumbnailPath string, bannerPath string, settings *Settings, db *gorm.DB) (string, error) {
	// read file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// unmarshall json into _twitch_raw_metadata_struct struct
	var v _twitch_raw_metadata_struct
	err = json.NewDecoder(file).Decode(&v)
	if err != nil {
		return "", err
	}

	// create new creator struct
	creator := Creator{
		ChannelID:   v.ID,
		Name:        v.DisplayName,
		Description: v.Description,
		Platform:    "Twitch",
	}
	creator.FilePath = strings.ReplaceAll(filePath, "//", "/")
	creator.FilePath = strings.ReplaceAll(filePath, "\\", "/")
	creator.FilePath = strings.ReplaceAll(creator.FilePath, settings.BaseTwitchPath, "")

	// if thumbnailpath exists, set thumbnail path
	if _, err := os.Stat(thumbnailPath); err == nil {
		thumbnailPath = strings.ReplaceAll(thumbnailPath, "//", "/")
		thumbnailPath = strings.ReplaceAll(thumbnailPath, "\\", "/")
		creator.ThumbnailPath = strings.ReplaceAll(thumbnailPath, settings.BaseTwitchPath, "")
	}

	// if bannerpath exists, set banner path
	if _, err := os.Stat(bannerPath); err == nil {
		bannerPath = strings.ReplaceAll(bannerPath, "//", "/")
		bannerPath = strings.ReplaceAll(bannerPath, "\\", "/")
		creator.BannerPath = strings.ReplaceAll(bannerPath, settings.BaseTwitchPath, "")
	}

	// check if creator already exists
	var existingCreator Creator
	db.Where("channel_id = ?", creator.ChannelID).First(&existingCreator)
	if existingCreator.ID != 0 {
		return creator.ChannelID, nil
	}

	// save creator
	err = db.Create(&creator).Error
	if err != nil {
		return "", err
	}

	return creator.ChannelID, nil

}

func importTwitchVideo(filePath string, chatPath string, thumbnailPath string, channelid string, channelname string, settings *Settings, db *gorm.DB) error {

	// filePath is in the format ${twitch-base-path)/${creator}/videos/${video_name} ($video_id).${ext}. We need to extract the video name, id, ane mime type.

	// get video name
	videoName := filePath[strings.LastIndex(filePath, "/")+1 : strings.LastIndex(filePath, "(")-1]
	// get video id
	videoID := filePath[strings.LastIndex(filePath, "(")+1 : strings.LastIndex(filePath, ")")]
	// get extension
	ext := filePath[strings.LastIndex(filePath, ".")+1:]
	// get mime type from extension
	mimeType := mime.TypeByExtension("." + ext)

	// check if text/vnd.trolltech.linguist; charset=utf-8 as this is an old format, change to video/MP2T
	if mimeType == "text/vnd.trolltech.linguist; charset=utf-8" {
		mimeType = "video/MP2T"
	}
	// set published at to the last modified time of the file
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	publishedAt := fileInfo.ModTime()
	// Set to human readable string
	publishedAtStr := publishedAt.Format("2006-01-02T15:04:05")

	// Convert to video struct
	vi := Video{
		VideoID:      videoID,
		ChannelID:    channelid,
		ChannelTitle: channelname,
		Title:        videoName,
		MimeType:     mimeType,
		PublishedAt:  publishedAtStr,
		VideoType:    "Twitch",
	}

	filePath = strings.ReplaceAll(filePath, "//", "/")
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	vi.FilePath = strings.ReplaceAll(filePath, settings.BaseTwitchPath, "")

	// get video length
	videoLength, err := getVideoLength(filePath)
	if err != nil {
		return err
	}

	// convert to string
	vi.Length = videoLength

	// if comments path exists, set comments path
	comOs, err := os.Stat(chatPath)
	if err == nil && !comOs.IsDir() {
		chatPath = strings.ReplaceAll(chatPath, "//", "/")
		chatPath = strings.ReplaceAll(chatPath, "\\", "/")
		vi.CommentsPath = strings.ReplaceAll(chatPath, settings.BaseTwitchPath, "")
	}
	// if thumbnail path exists, set thumbnail path
	thumbOs, err := os.Stat(thumbnailPath)
	if err == nil && !thumbOs.IsDir() {
		thumbnailPath = strings.ReplaceAll(thumbnailPath, "//", "/")
		thumbnailPath = strings.ReplaceAll(thumbnailPath, "\\", "/")
		vi.ThumbnailPath = strings.ReplaceAll(thumbnailPath, settings.BaseTwitchPath, "")
	}

	// check if video already exists
	var v Video
	err = db.Where("video_id = ?", videoID).First(&v).Error
	if err != nil && err.Error() != "record not found" {
		// return error
		return err
	} else if err == nil {
		// video already exists
		return errors.New("record already exists")
	}

	// insert video into database
	err = db.Create(&vi).Error
	if err != nil {
		return err
	}

	return nil

}
