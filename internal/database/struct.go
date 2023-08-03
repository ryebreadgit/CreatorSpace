package database

import (
	"time"

	"gorm.io/gorm"
)

// GORM Database Models

type Creator struct {
	gorm.Model
	ID             int `gorm:"primaryKey"`
	ChannelID      string
	Name           string
	AltName        string
	Description    string
	VideoIDs       string
	FilePath       string
	ThumbnailPath  string
	BannerPath     string
	LinkedAccounts string
	Platform       string
	Subscribers    string
	VideoCount     int
}

type Video struct {
	gorm.Model
	ID            int `gorm:"primaryKey"`
	VideoID       string
	ChannelID     string
	ChannelTitle  string
	PublishedAt   string
	Epoch         int64
	Title         string
	Description   string
	Views         string
	Likes         string
	Length        string
	FilePath      string
	MetadataPath  string
	ThumbnailPath string
	CommentsPath  string
	SubtitlePath  string
	Availability  string
	Progress      string
	VideoType     string
	Categories    string
	Tags          string
	Watched       bool
	AgeRestricted bool
	Updated       bool
	MimeType      string
	SponsorTag    string
}

type Tweet struct {
	gorm.Model
	ID              int `gorm:"primaryKey"`
	TweetID         string
	ConversationID  string
	UserID          string
	Username        string
	UserDisplayName string
	Epoch           int64
	IsQuote         bool
	QuoteID         string
	IsReply         bool
	InReplyToID     string
	IsRetweet       bool
	RetweetID       string
	IsPin           bool
	Likes           int
	ReplyCount      int
	RetweetCount    int
	Photos          string
	Videos          string
	Text            string
	URLs            string
	FilePath        string
}

type SponsorBlock struct {
	gorm.Model
	Deleted      gorm.DeletedAt
	ID           int `gorm:"primaryKey"`
	VideoID      string
	SegmentID    string
	SegmentStart float64
	SegmentEnd   float64
	Category     string
	ActionType   string
	Hidden       int
	ShadowHidden int
	Votes        int
	FilePath     string
}

type Comment struct {
	gorm.Model
	ID              int    `gorm:"primaryKey"`
	CommentID       string `json:"cid"`
	VideoID         string
	Text            string
	Author          string
	Heart           bool
	TimeParsed      float64 `json:"time_parsed"`
	TimeString      string
	ParentCommentID string
	Votes           string
	MetadataJson    string
	FilePath        string
}

type Playlist struct {
	gorm.Model
	ID           int `gorm:"primaryKey"`
	PlaylistID   string
	VideoIDs     string
	ChannelID    string
	ChannelTitle string
	Name         string
	Description  string
	Thumbnail    string
	UserID       string
}

type Tasking struct {
	gorm.Model
	ID           int `gorm:"primaryKey"`
	TaskName     string
	Epoch        int64
	EpochLastRan int64
	Interval     time.Duration
}

type Settings struct {
	gorm.Model
	ID               int `gorm:"primaryKey"`
	BaseYouTubePath  string
	BaseTwitchPath   string
	BaseTwitterPath  string
	DatabasePath     string
	DatabaseType     string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePass     string
	DatabaseName     string
	DatabaseSSLMode  string
	DatabaseTimeZone string
	RedisAddress     string
	RedisPassword    string
	ServerPath       string
	RedisDB          int
	JwtSecret        string
	OpenRegister     bool
	PublicImages     bool
}

type User struct {
	gorm.Model
	ID int `gorm:"primaryKey"`
	// User info
	UserID      string
	Username    string
	Password    string
	Email       string
	AccountType string

	// User settings
	SponsorBlockEnabled    bool
	SponsorBlockCategories string
}

type DownloadQueue struct {
	gorm.Model
	ID           int `gorm:"primaryKey"`
	VideoID      string
	DownloadPath string
	VideoType    string
	Source       string
	Requester    string
	Running      bool
	Downloaded   bool
	Approved     bool
	FilePath     string
}

// Custom Structs

type FileJsonSponsorBlock struct {
	SegmentCount int `json:"segmentCount"`
	Page         int `json:"page"`
	Segments     []struct {
		UUID          string  `json:"UUID"`
		TimeSubmitted int64   `json:"timeSubmitted"`
		StartTime     float64 `json:"startTime"`
		EndTime       float64 `json:"endTime"`
		Category      string  `json:"category"`
		ActionType    string  `json:"actionType"`
		Votes         int     `json:"votes"`
		Views         int     `json:"views"`
		Locked        int     `json:"locked"`
		Hidden        int     `json:"hidden"`
		ShadowHidden  int     `json:"shadowHidden"`
		UserID        string  `json:"userID"`
		Description   string  `json:"description"`
	} `json:"segments"`
}

type ProgressToken struct {
	VideoID  string `json:"video_id"`
	Progress string `json:"progress" binding:"required"`
}

// Raw Metadata Structs

type _twitch_raw_metadata_struct struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	CreatedAt       string `json:"created_at"`
}

type YouTubeApiVideoInfoStruct struct {
	Kind    string `json:"kind"`
	Etag    string `json:"etag"`
	ID      string `json:"id"`
	Snippet struct {
		PublishedAt string `json:"publishedAt"`
		ChannelID   string `json:"channelId"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Thumbnails  struct {
			Default struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"default"`
			Medium struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"medium"`
			High struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"high"`
			Standard struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"standard"`
			Maxres struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"maxres"`
		} `json:"thumbnails"`
		ChannelTitle         string   `json:"channelTitle"`
		Tags                 []string `json:"tags"`
		CategoryID           string   `json:"categoryId"`
		LiveBroadcastContent string   `json:"liveBroadcastContent"`
		DefaultLanguage      string   `json:"defaultLanguage"`
		Localized            struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"localized"`
		DefaultAudioLanguage string `json:"defaultAudioLanguage"`
	} `json:"snippet"`
	ContentDetails struct {
		Duration          string      `json:"duration"`
		Dimension         string      `json:"dimension"`
		Definition        string      `json:"definition"`
		Caption           string      `json:"caption"`
		LicensedContent   bool        `json:"licensedContent"`
		RegionRestriction interface{} `json:"regionRestriction"`
		ContentRating     struct {
		} `json:"contentRating"`
		Projection         string      `json:"projection"`
		HasCustomThumbnail interface{} `json:"hasCustomThumbnail"`
	} `json:"contentDetails"`
	Status struct {
		UploadStatus            string      `json:"uploadStatus"`
		FailureReason           interface{} `json:"failureReason"`
		RejectionReason         interface{} `json:"rejectionReason"`
		PrivacyStatus           string      `json:"privacyStatus"`
		PublishAt               interface{} `json:"publishAt"`
		License                 string      `json:"license"`
		Embeddable              bool        `json:"embeddable"`
		PublicStatsViewable     bool        `json:"publicStatsViewable"`
		MadeForKids             bool        `json:"madeForKids"`
		SelfDeclaredMadeForKids interface{} `json:"selfDeclaredMadeForKids"`
	} `json:"status"`
	Statistics struct {
		ViewCount    string      `json:"viewCount"`
		LikeCount    string      `json:"likeCount"`
		DislikeCount interface{} `json:"dislikeCount"`
		CommentCount string      `json:"commentCount"`
	} `json:"statistics"`
	TopicDetails struct {
		TopicIds         interface{} `json:"topicIds"`
		RelevantTopicIds interface{} `json:"relevantTopicIds"`
		TopicCategories  []string    `json:"topicCategories"`
	} `json:"topicDetails"`
	LiveStreamingDetails interface{} `json:"liveStreamingDetails"`
}

type YouTubeVideoInfoStruct struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Formats []struct {
		FormatID   string  `json:"format_id"`
		FormatNote string  `json:"format_note"`
		Ext        string  `json:"ext"`
		Protocol   string  `json:"protocol"`
		Acodec     string  `json:"acodec"`
		Vcodec     string  `json:"vcodec"`
		URL        string  `json:"url"`
		Width      int     `json:"width"`
		Height     int     `json:"height"`
		Fps        float64 `json:"fps"`
		Rows       int     `json:"rows,omitempty"`
		Columns    int     `json:"columns,omitempty"`
		Fragments  []struct {
			URL      string  `json:"url"`
			Duration float64 `json:"duration"`
		} `json:"fragments,omitempty"`
		Resolution  string  `json:"resolution"`
		AspectRatio float64 `json:"aspect_ratio"`
		HTTPHeaders struct {
			UserAgent      string `json:"User-Agent"`
			Accept         string `json:"Accept"`
			AcceptLanguage string `json:"Accept-Language"`
			SecFetchMode   string `json:"Sec-Fetch-Mode"`
		} `json:"http_headers"`
		AudioExt           string      `json:"audio_ext"`
		VideoExt           string      `json:"video_ext"`
		Format             string      `json:"format"`
		Asr                int         `json:"asr,omitempty"`
		Filesize           int         `json:"filesize,omitempty"`
		SourcePreference   int         `json:"source_preference,omitempty"`
		AudioChannels      int         `json:"audio_channels,omitempty"`
		Quality            float64     `json:"quality,omitempty"`
		HasDrm             bool        `json:"has_drm,omitempty"`
		Tbr                float64     `json:"tbr,omitempty"`
		Language           interface{} `json:"language,omitempty"`
		LanguagePreference int         `json:"language_preference,omitempty"`
		Preference         interface{} `json:"preference,omitempty"`
		DynamicRange       interface{} `json:"dynamic_range,omitempty"`
		Abr                float64     `json:"abr,omitempty"`
		Container          string      `json:"container,omitempty"`
		Vbr                float64     `json:"vbr,omitempty"`
		FilesizeApprox     int         `json:"filesize_approx,omitempty"`
	} `json:"formats"`
	Thumbnails []struct {
		URL        string `json:"url"`
		Preference int    `json:"preference"`
		ID         string `json:"id"`
		Height     int    `json:"height,omitempty"`
		Width      int    `json:"width,omitempty"`
		Resolution string `json:"resolution,omitempty"`
	} `json:"thumbnails"`
	Thumbnail            string                                    `json:"thumbnail"`
	Description          string                                    `json:"description"`
	Uploader             string                                    `json:"uploader"`
	UploaderID           string                                    `json:"uploader_id"`
	UploaderURL          string                                    `json:"uploader_url"`
	ChannelID            string                                    `json:"channel_id"`
	ChannelURL           string                                    `json:"channel_url"`
	Duration             int                                       `json:"duration"`
	ViewCount            int                                       `json:"view_count"`
	AverageRating        interface{}                               `json:"average_rating"`
	AgeLimit             int                                       `json:"age_limit"`
	WebpageURL           string                                    `json:"webpage_url"`
	Categories           []string                                  `json:"categories"`
	Tags                 []string                                  `json:"tags"`
	PlayableInEmbed      bool                                      `json:"playable_in_embed"`
	LiveStatus           string                                    `json:"live_status"`
	ReleaseTimestamp     int                                       `json:"release_timestamp"`
	FormatSortFields     []string                                  `json:"_format_sort_fields"`
	Subtitles            map[string][]YouTubeApiVideoInfoStructSub `json:"subtitles"`
	CommentCount         int                                       `json:"comment_count"`
	Chapters             interface{}                               `json:"chapters"`
	LikeCount            int                                       `json:"like_count"`
	Channel              string                                    `json:"channel"`
	ChannelFollowerCount int                                       `json:"channel_follower_count"`
	UploadDate           string                                    `json:"upload_date"`
	Availability         string                                    `json:"availability"`
	OriginalURL          string                                    `json:"original_url"`
	WebpageURLBasename   string                                    `json:"webpage_url_basename"`
	WebpageURLDomain     string                                    `json:"webpage_url_domain"`
	Extractor            string                                    `json:"extractor"`
	ExtractorKey         string                                    `json:"extractor_key"`
	Playlist             interface{}                               `json:"playlist"`
	PlaylistIndex        interface{}                               `json:"playlist_index"`
	DisplayID            string                                    `json:"display_id"`
	Fulltitle            string                                    `json:"fulltitle"`
	DurationString       string                                    `json:"duration_string"`
	ReleaseDate          string                                    `json:"release_date"`
	IsLive               bool                                      `json:"is_live"`
	WasLive              bool                                      `json:"was_live"`
	RequestedSubtitles   interface{}                               `json:"requested_subtitles"`
	HasDrm               interface{}                               `json:"_has_drm"`
	Comments             []struct {
		ID               string `json:"id"`
		Text             string `json:"text"`
		Timestamp        int    `json:"timestamp"`
		TimeText         string `json:"time_text"`
		LikeCount        int    `json:"like_count"`
		IsFavorited      bool   `json:"is_favorited"`
		Author           string `json:"author"`
		AuthorID         string `json:"author_id"`
		AuthorThumbnail  string `json:"author_thumbnail"`
		AuthorIsUploader bool   `json:"author_is_uploader"`
		Parent           string `json:"parent"`
	} `json:"comments"`
	RequestedFormats []struct {
		Asr                interface{} `json:"asr"`
		Filesize           int         `json:"filesize"`
		FormatID           string      `json:"format_id"`
		FormatNote         string      `json:"format_note"`
		SourcePreference   int         `json:"source_preference"`
		Fps                float64     `json:"fps"`
		AudioChannels      interface{} `json:"audio_channels"`
		Height             int         `json:"height"`
		Quality            float64     `json:"quality"`
		HasDrm             bool        `json:"has_drm"`
		Tbr                float64     `json:"tbr"`
		URL                string      `json:"url"`
		Width              int         `json:"width"`
		Language           interface{} `json:"language"`
		LanguagePreference int         `json:"language_preference"`
		Preference         interface{} `json:"preference"`
		Ext                string      `json:"ext"`
		Vcodec             string      `json:"vcodec"`
		Acodec             string      `json:"acodec"`
		DynamicRange       string      `json:"dynamic_range"`
		Vbr                float64     `json:"vbr,omitempty"`
		Protocol           string      `json:"protocol"`
		Fragments          []struct {
			URL string `json:"url"`
		} `json:"fragments"`
		Container   string  `json:"container"`
		Resolution  string  `json:"resolution"`
		AspectRatio float64 `json:"aspect_ratio"`
		HTTPHeaders struct {
			UserAgent      string `json:"User-Agent"`
			Accept         string `json:"Accept"`
			AcceptLanguage string `json:"Accept-Language"`
			SecFetchMode   string `json:"Sec-Fetch-Mode"`
		} `json:"http_headers"`
		VideoExt string  `json:"video_ext"`
		AudioExt string  `json:"audio_ext"`
		Format   string  `json:"format"`
		Abr      float64 `json:"abr,omitempty"`
	} `json:"requested_formats"`
	Format         string      `json:"format"`
	FormatID       string      `json:"format_id"`
	Ext            string      `json:"ext"`
	Protocol       string      `json:"protocol"`
	Language       interface{} `json:"language"`
	FormatNote     string      `json:"format_note"`
	FilesizeApprox int         `json:"filesize_approx"`
	Tbr            float64     `json:"tbr"`
	Width          int         `json:"width"`
	Height         int         `json:"height"`
	Resolution     string      `json:"resolution"`
	Fps            float64     `json:"fps"`
	DynamicRange   string      `json:"dynamic_range"`
	Vcodec         string      `json:"vcodec"`
	Vbr            float64     `json:"vbr"`
	StretchedRatio interface{} `json:"stretched_ratio"`
	AspectRatio    float64     `json:"aspect_ratio"`
	Acodec         string      `json:"acodec"`
	Abr            float64     `json:"abr"`
	Asr            int         `json:"asr"`
	AudioChannels  int         `json:"audio_channels"`
	Epoch          int         `json:"epoch"`
	Filename       string      `json:"_filename"`
	Filename0      string      `json:"filename"`
	Urls           string      `json:"urls"`
	Type           string      `json:"_type"`
	Version        struct {
		Version        string      `json:"version"`
		CurrentGitHead interface{} `json:"current_git_head"`
		ReleaseGitHead string      `json:"release_git_head"`
		Repository     string      `json:"repository"`
	} `json:"_version"`
}

type YouTubeApiVideoInfoStructSub struct {
	Ext      string `json:"ext"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	VideoID  string `json:"video_id"`
	Protocol string `json:"protocol"`
}

type SponsorBlockRawApi struct {
	Category      string    `json:"category"`
	ActionType    string    `json:"actionType"`
	Segment       []float64 `json:"segment"`
	UUID          string    `json:"UUID"`
	VideoDuration float64   `json:"videoDuration"`
	Locked        int       `json:"locked"`
	Votes         int       `json:"votes"`
	Description   string    `json:"description"`
}

type YoutubePlaylistStruct struct {
	ID                   string      `json:"id"`
	Uploader             string      `json:"uploader"`
	UploaderID           string      `json:"uploader_id"`
	UploaderURL          string      `json:"uploader_url"`
	Title                string      `json:"title"`
	Availability         interface{} `json:"availability"`
	ChannelFollowerCount int         `json:"channel_follower_count"`
	Description          string      `json:"description"`
	Tags                 []string    `json:"tags"`
	Thumbnails           []struct {
		URL        string `json:"url"`
		Height     int    `json:"height,omitempty"`
		Width      int    `json:"width,omitempty"`
		Preference int    `json:"preference,omitempty"`
		ID         string `json:"id"`
		Resolution string `json:"resolution,omitempty"`
	} `json:"thumbnails"`
	ModifiedDate       interface{}   `json:"modified_date"`
	ViewCount          int64         `json:"view_count"`
	PlaylistCount      int           `json:"playlist_count"`
	Channel            string        `json:"channel"`
	ChannelID          string        `json:"channel_id"`
	ChannelURL         string        `json:"channel_url"`
	Type               string        `json:"_type"`
	Entries            []interface{} `json:"entries"`
	ExtractorKey       string        `json:"extractor_key"`
	Extractor          string        `json:"extractor"`
	WebpageURL         string        `json:"webpage_url"`
	OriginalURL        string        `json:"original_url"`
	WebpageURLBasename string        `json:"webpage_url_basename"`
	WebpageURLDomain   string        `json:"webpage_url_domain"`
	FilesToMove        struct {
	} `json:"__files_to_move"`
	Epoch   int `json:"epoch"`
	Version struct {
		Version        string      `json:"version"`
		CurrentGitHead interface{} `json:"current_git_head"`
		ReleaseGitHead string      `json:"release_git_head"`
		Repository     string      `json:"repository"`
	} `json:"_version"`
}

type VidSubtitle struct {
	Language     string `json:"language"`
	LanguageText string
	FilePath     string `json:"filepath"`
}
