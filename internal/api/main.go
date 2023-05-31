package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

var db *gorm.DB
var settings *database.Settings
var ctx context.Context = context.Background()
var rdb *redis.Client

func Routes(route *gin.Engine) {
	api := route.Group("/api")
	{
		search := api.Group("/search")
		{
			search.Use(jwttoken.JwtMiddleware())
			search.GET("/creators", apiSearchCreators)
			search.GET("/videos", apiSearchVideos)
		}
		media := api.Group("/media")
		{
			media.Use(gin.Logger())
			media.Use(jwttoken.JwtMiddleware())
			media.GET("/:video_id", apiMedia)
			media.GET("/:video_id/thumbnail", apiThumbnail)
			media.GET("/:video_id/recommendations", func(c *gin.Context) {
				vidID := c.Param("video_id")
				userID, err := jwttoken.GetUserFromToken(c)
				if err != nil {
					c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": err.Error()})
					return
				}
				if userID == "" {
					c.AbortWithStatusJSON(401, gin.H{"ret": 401, "err": "Not logged in"})
					return
				}

				watchedVideos, err := database.GetPlaylistByUserID(userID, "Completed Videos", db)
				if err != nil {
					c.HTML(http.StatusNotFound, "error.tmpl", gin.H{
						"ret": 404,
						"err": "Unable to get watched videos",
					})
					return
				}

				retData, err := GetRecommendations(vidID, watchedVideos)
				if err != nil {
					c.AbortWithStatusJSON(503, gin.H{"ret": 503, "err": err.Error()})
					return
				}
				c.JSON(200, gin.H{"ret": 200, "data": retData})
			})

			// TODO - finish trancoding

			media.GET("/:video_id/manifest.m3u8", streamTranscodedVideo)
			media.GET("/transcoding/video/:uuid/manifest.m3u8", ServeHLSManifest)
			media.GET("/transcoding/video/:uuid/:chunk_name", ServeVideoChunk)

		}

		downloads := api.Group("/downloads")
		{
			downloads.Use(jwttoken.JwtMiddleware())
			downloads.POST("/:video_id/:video_type", apiDownloadVideo)
		}

		user := api.Group("/user")
		{
			user.Use(jwttoken.JwtMiddleware())
			user.POST("/:user_id/sponsorblock", apiUpdateSponsorblock)

			user.GET("/:user_id/progress/:video_id", apiGetWatchTime)
			user.POST("/:user_id/progress/:video_id", apiUpdateWatchTime)
			user.POST("/:user_id/progress/:video_id/complete", apiMarkVideoComplete)

			user.POST("/:user_id/subscriptions/:creator_id", apiAddSubscription)
			user.DELETE("/:user_id/subscriptions/:creator_id", apiRemoveSubscription)
		}

		yt := api.Group("/youtube")
		{
			yt.Use(jwttoken.JwtMiddleware())
			yt.GET("/creators/:creator", wrapper(apiCreators))
			yt.GET("/creators/:creator/thumbnail", apiThumbnail)
			yt.GET("/creators/:creator/banner", apiCreatorBanner)

			yt.GET("/metadata/:video_id", wrapper(apiVideoMetadata))

			yt.GET("/sponsorblock/:video_id", wrapper(apiVideoSponsorblock))

			yt.GET("/comments/:video_id", wrapper(apiVideoComments))
			yt.GET("/comments/:video_id/all", apiAllVideoComments)

			yt.GET("/subtitles/:video_id/:lang", getVideoSubtitles)
		}

		twitch := api.Group("/twitch")
		{
			twitch.Use(jwttoken.JwtMiddleware())
			twitch.GET("/creators/:creator", wrapper(apiCreators))
			twitch.GET("/creators/:creator/thumbnail", apiThumbnail)

			twitch.GET("/comments/:video_id/all", apiAllVideoComments)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/login", wrapper(apiUserLogin))
			auth.POST("/logout", wrapper(apiUserLogout))
			auth.POST("/register", wrapper(apiUserSignup))
		}

	}
}

func init() {
	var err error
	// get database
	db, err = database.GetDatabase()
	if err != nil {
		fmt.Printf("Error connecting to database: %s\n", err)
		return
	}

	// get settings
	settings, err = database.GetSettings(db)
	if err != nil {
		fmt.Printf("Error getting settings: %s\n", err)
		return
	}
	rdb, err = initRedis(ctx)
	if err != nil {
		fmt.Printf("Error connecting to redis: %v\n", err)
	}

	// if ./newuser.txt exists, create a new user
	if ifFileExists("./newuser.txt") {
		// Open the file and read the contents to database.User
		f, err := os.Open("./newuser.txt")
		if err != nil {
			fmt.Printf("Error opening file: %s\n", err)
			return
		}
		defer f.Close()

		var userdata database.User
		err = json.NewDecoder(f).Decode(&userdata)
		if err != nil {
			fmt.Printf("Error decoding file: %s\n", err)
			return
		}

		// create user
		err = database.SignupUser(userdata, db)
		if err != nil {
			fmt.Printf("Error creating user: %s\n", err)
			return
		}

		// delete file
		err = os.Remove("./newuser.txt")
		if err != nil {
			fmt.Printf("Error deleting file: %s\n", err)
			return
		}

		fmt.Printf("New user created: %v\n", userdata.Username)
	}

}

func ifFileExists(file string) bool {
	// Check if file exists
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
