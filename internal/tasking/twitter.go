package tasking

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	log "github.com/sirupsen/logrus"
)

func UpdateTwitterCreators(tweetLimit int) error {
	// Get all download tasks where video_type = twitter
	queue, err := database.GetDownloadQueue(db.Where("video_type = ?", "twitter"))
	if err != nil {
		log.Error(err)
		return err
	}

	// Setting tweet limit to 3200 if over as this is the max for the api
	if tweetLimit > 3200 {
		tweetLimit = 3200
	}

	// Get twitter handle from video_id
	for _, item := range queue {
		// Get twitter handle from video_id
		updateCreatorTweets(item.VideoID, tweetLimit)
	}

	return nil

}

func updateCreatorTweets(handle string, tweetlimit int) error {
	// Get twitter handle from creator
	scraper := twitterscraper.New()
	err := scraper.LoginOpenAccount()
	if err != nil {
		log.Error(err)
		return err
	}

	// Get the creator, or create if it doesn't exist
	creator, err := database.GetAllCreators(db.Where("name = ? AND platform = ?", handle, "twitter"))
	if err != nil && err.Error() != "record not found" {
		log.Error(err)
		return err
	} else if (err != nil && err.Error() == "record not found") || len(creator) == 0 {
		// Create creator
		newcreator := &database.Creator{}

		// Get creator info from scraper
		profile, err := scraper.GetProfile(handle)
		if err != nil {
			log.Error(err)
			return err
		}

		basePath := filepath.Join(settings.BaseTwitterPath, handle)

		newcreator.Name = profile.Username
		newcreator.AltName = profile.Name
		newcreator.Platform = "twitter"
		newcreator.FilePath = filepath.Join(basePath, handle+".json")
		newcreator.BannerPath = filepath.Join(basePath, "banner.jpg")
		newcreator.ThumbnailPath = filepath.Join(basePath, "avatar.jpg")
		newcreator.Description = profile.Biography
		newcreator.Subscribers = fmt.Sprintf("%d", profile.FollowersCount)
		newcreator.ChannelID = profile.UserID

		// Make directories
		err = os.MkdirAll(basePath, os.ModePerm)
		if err != nil {
			log.Error(err)
			return err
		}

		if profile.Banner != "" {
			// Download banner
			bannerPath, err := downloadThumbnail(profile.Banner, newcreator.BannerPath, "banner.jpg")
			if err != nil {
				log.Error(err)
				return err
			}

			newcreator.BannerPath = bannerPath
		} else {
			newcreator.BannerPath = ""
		}

		if profile.Avatar != "" {
			// Download thumbnail
			thumbnailPath, err := downloadThumbnail(strings.ReplaceAll(profile.Avatar, "_normal", ""), newcreator.ThumbnailPath, "avatar.jpg")
			if err != nil {
				log.Error(err)
				return err
			}

			newcreator.ThumbnailPath = thumbnailPath
		} else {
			newcreator.ThumbnailPath = ""
		}

		// Save json to FilePath
		creatorjson, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			log.Error(err)
			return err
		}

		err = os.WriteFile(newcreator.FilePath, creatorjson, 0644)
		if err != nil {
			log.Error(err)
			return err
		}

		// Remove twitter path from paths
		newcreator.FilePath = strings.ReplaceAll(newcreator.FilePath, settings.BaseTwitterPath, "")
		newcreator.BannerPath = strings.ReplaceAll(newcreator.BannerPath, settings.BaseTwitterPath, "")
		newcreator.ThumbnailPath = strings.ReplaceAll(newcreator.ThumbnailPath, settings.BaseTwitterPath, "")

		// Add creator to database
		err = database.InsertCreator(*newcreator, db)
		if err != nil {
			log.Error(err)
			return err
		}

		creator = []database.Creator{}
		creator = append(creator, *newcreator)

		log.Info("Created new twitter creator: " + newcreator.Name)

	}

	log.Infof("Updating twitter creator: %v", creator[0].Name)

	basePath := filepath.Join(settings.BaseTwitterPath, creator[0].Name)
	tweetsPath := filepath.Join(basePath, "tweets")

	for tweet := range scraper.WithReplies(true).GetTweets(context.Background(), creator[0].Name, tweetlimit) {
		if tweet.Error != nil {
			log.Error(tweet.Error)
			continue
		}

		// Check if tweet currently exists
		_, err := database.GetTweetByTweetID(tweet.ID, db)
		if err == nil {
			continue
		}

		tweetPath := filepath.Join(tweetsPath, tweet.ID)
		if tweet.ConversationID != "" {
			tweetPath = filepath.Join(tweetsPath, tweet.ConversationID)
		}

		// Download tweet
		_, err = downloadTweets(&tweet.Tweet, tweetPath)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debugf("Downloaded tweet: %v", tweet.ID)
	}

	return nil
}

func downloadTweets(tweet *twitterscraper.Tweet, fp string) ([]database.Tweet, error) {
	if tweet == nil || tweet.ID == "" {
		return nil, nil
	}

	// Check if tweet exists in database
	_, err := database.GetTweetByTweetID(tweet.ID, db)
	if err == nil {
		// Tweet exists, skip
		return nil, nil
	}

	// Tweet does not exist, add to database
	var dbtweet database.Tweet
	var retTweets []database.Tweet

	dbtweet.TweetID = tweet.ID
	dbtweet.ConversationID = tweet.ConversationID
	dbtweet.Username = tweet.Username
	dbtweet.Text = tweet.Text
	dbtweet.Epoch = tweet.Timestamp
	dbtweet.UserID = tweet.UserID
	dbtweet.IsReply = tweet.IsReply
	dbtweet.InReplyToID = tweet.InReplyToStatusID
	dbtweet.IsRetweet = tweet.IsRetweet
	dbtweet.RetweetID = tweet.RetweetedStatusID
	dbtweet.IsQuote = tweet.IsQuoted
	dbtweet.QuoteID = tweet.QuotedStatusID
	dbtweet.IsPin = tweet.IsPin
	dbtweet.Likes = tweet.Likes
	dbtweet.URLs = strings.Join(tweet.URLs, ",")
	dbtweet.ReplyCount = tweet.Replies
	dbtweet.RetweetCount = tweet.Retweets

	// Make directories
	err = os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Save tweet to file
	savePath := filepath.Join(fp, tweet.ID+".json")
	tweetJson, err := json.MarshalIndent(tweet, "", "  ")
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = os.WriteFile(savePath, []byte(tweetJson), 0644)
	if err != nil {
		return nil, err
	}

	photos := []string{}
	videos := []string{}

	// Save images to file
	for _, img := range tweet.Photos {
		// Download image
		baseImgPath := filepath.Join(fp, img.ID+".jpg")
		imgPath, err := downloadThumbnail(img.URL, baseImgPath, tweet.ID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		photos = append(photos, imgPath)
	}

	// Save videos to file
	for _, vid := range tweet.Videos {
		// Download video
		baseVidPath := filepath.Join(fp, vid.ID+".mp4")
		vidPath, err := downloadThumbnail(vid.URL, baseVidPath, tweet.ID)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		videos = append(videos, vidPath)
	}

	dbtweet.FilePath = strings.ReplaceAll(savePath, settings.BaseTwitterPath, "")

	photosJson, err := json.Marshal(photos)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	videosJson, err := json.Marshal(videos)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	dbtweet.Photos = string(photosJson)
	dbtweet.Videos = string(videosJson)

	// Add tweet to database
	err = database.InsertTweet(dbtweet, db)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	retTweets = append(retTweets, dbtweet)

	// Check if tweet is a reply
	if tweet.IsReply {
		scraper := twitterscraper.New()
		err := scraper.LoginOpenAccount()
		if err != nil {
			log.Error(err)
			return nil, err
		}
		// Download parent tweet
		replytweet, err := scraper.GetTweet(tweet.InReplyToStatusID)
		if err != nil {
			log.Warnf("Unable to get tweet %v: %v", tweet.InReplyToStatusID, err)
			// Add to download queue
			err = database.InsertDownloadQueueItem(database.DownloadQueue{
				VideoID:      tweet.InReplyToStatusID,
				VideoType:    "twitter",
				DownloadPath: fp,
				Source:       "twitter",
				Requester:    "system",
				Approved:     true,
			}, db)
			if err != nil {
				log.Error(err)
				return nil, err
			}
		} else {
			replies, err := downloadTweets(replytweet, fp)
			if err != nil {
				return nil, err
			}
			retTweets = append(retTweets, replies...)
		}
	}

	// Check if tweet is a quote
	if tweet.IsQuoted {
		scraper := twitterscraper.New()
		err := scraper.LoginOpenAccount()
		if err != nil {
			log.Error(err)
			return nil, err
		}
		// Download parent tweet
		replytweet, err := scraper.GetTweet(tweet.QuotedStatusID)
		if err != nil {
			log.Warnf("Unable to get tweet %v: %v", tweet.QuotedStatusID, err)
			// Add to download queue
			err = database.InsertDownloadQueueItem(database.DownloadQueue{
				VideoID:      tweet.QuotedStatusID,
				VideoType:    "twitter",
				DownloadPath: fp,
				Source:       "twitter",
				Requester:    "system",
				Approved:     true,
			}, db)
			if err != nil {
				log.Error(err)
				return nil, err
			}
		} else {
			replies, err := downloadTweets(replytweet, fp)
			if err != nil {
				return nil, err
			}
			retTweets = append(retTweets, replies...)
		}
	}

	// Check if tweet is a retweet
	if tweet.IsRetweet {
		// Download parent tweet
		scraper := twitterscraper.New()
		err := scraper.LoginOpenAccount()
		if err != nil {
			log.Error(err)
			return nil, err
		}
		replytweet, err := scraper.GetTweet(tweet.RetweetedStatusID)
		if err != nil {
			log.Warnf("Unable to get tweet %v: %v", tweet.RetweetedStatusID, err)
			// Add to download queue
			err = database.InsertDownloadQueueItem(database.DownloadQueue{
				VideoID:      tweet.RetweetedStatusID,
				VideoType:    "twitter",
				DownloadPath: fp,
				Source:       "twitter",
				Requester:    "system",
				Approved:     true,
			}, db)
			if err != nil {
				log.Error(err)
				return nil, err
			}
		} else {
			replies, err := downloadTweets(replytweet, fp)
			if err != nil {
				return nil, err
			}
			retTweets = append(retTweets, replies...)
		}
	}

	return retTweets, nil

}
