package api

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	"gorm.io/gorm"
)

var (
	db           *gorm.DB
	settings     *database.Settings
	ctx          context.Context = context.Background()
	rdb          *redis.Client
	GitCommit    string = "unknown"
	BuildDate    string = "unknown"
	AppVersion   string = "unknown"
	YTDLPVersion string = GetYTDLPVersion()
	GoVersion    string = runtime.Version()
	ApiStartTime time.Time
)

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

			// Check if public images is enabled and only enable JWT after if so
			if !settings.PublicImages {
				media.Use(jwttoken.JwtMiddleware())
			}
			media.GET("/:video_id/thumbnail", apiThumbnail)
			if settings.PublicImages {
				media.Use(jwttoken.JwtMiddleware())
			}

			media.GET("/:video_id", apiMedia)
			media.GET("/:video_id/recommendations", func(c *gin.Context) {
				vidID := c.Param("video_id")
				userID, err := jwttoken.GetUserFromToken(c)
				if err != nil {
					c.AbortWithStatusJSON(500, gin.H{"ret": 500, "err": err.Error()})
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

			/* TODO - finish trancoding

			media.GET("/:video_id/manifest.m3u8", streamTranscodedVideo)
			media.GET("/transcoding/video/:uuid/manifest.m3u8", ServeHLSManifest)
			media.GET("/transcoding/video/:uuid/:chunk_name", ServeVideoChunk)
			*/
		}

		downloads := api.Group("/downloads")
		{
			downloads.Use(jwttoken.JwtMiddleware())
			downloads.POST("/:video_id/:video_type", apiDownloadVideo)
			downloads.GET("/downloads", apiGetDownloadInfo)
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

			user.PATCH("/:user_id/password", apiUpdatePassword)
		}

		creators := api.Group("/creators")
		{
			creators.Use(jwttoken.JwtMiddleware())
			creators.GET("/:creator_id", apiGetCreator)
			creators.GET("/:creator_id/videos", apiGetCreatorVideos)
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

		admin := api.Group("/admin")
		{
			admin.Use(jwttoken.JwtMiddleware())
			admin.Use(AdminMiddleware())
			admin.GET("/users", apiGetAllUsers)
			admin.POST("/users", apiCreateUser)
			admin.GET("/users/:user_id", apiGetUser)
			admin.DELETE("/users/:user_id", apiDeleteUser)
			admin.PATCH("/users/:user_id/password", apiUpdateUserPassword)
			admin.PATCH("/users/:user_id/role", apiUpdateUserRole)
		}
	}
	version := route.Group("/version")
	{
		version.Use(jwttoken.JwtMiddleware())
		version.GET("", apiVersion)
	}
}

func GetYTDLPVersion() string {
	// Run yt-dlp --version
	args := []string{"--version"}
	cmd := exec.Command("yt-dlp", args...)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running yt-dlp: %s\n", err)
		return "unknown"
	}
	return strings.TrimSpace(string(stdout))
}

func GetVersion() apiVersionStruct {
	uptime := time.Since(ApiStartTime).Round(time.Millisecond).String()
	var ver apiVersionStruct = apiVersionStruct{
		CommitHash:   GitCommit,
		BuildDate:    BuildDate,
		AppVersion:   AppVersion,
		GoVersion:    GoVersion,
		YTDLPVersion: YTDLPVersion,
		Uptime:       uptime,
	}
	return ver
}

func apiVersion(c *gin.Context) {
	c.JSON(200, gin.H{"ret": 200, "data": GetVersion()})
}

func init() {
	var err error
	ApiStartTime = time.Now()
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

}
