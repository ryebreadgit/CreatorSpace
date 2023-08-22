package api

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	jwttoken "github.com/ryebreadgit/CreatorSpace/internal/jwt"
	log "github.com/sirupsen/logrus"
)

func apiUpdateWatchTime(c *gin.Context) {
	// get the current time from the request
	// get post as json progress
	bodyAsByteArray, _ := io.ReadAll(c.Request.Body)
	// convert json to struct
	var progresstk database.ProgressToken
	err := json.Unmarshal(bodyAsByteArray, &progresstk)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Invalid request body",
		})
		c.Abort()
		return
	}

	currentTime := progresstk.Progress

	// get jwt token
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Unauthorized",
		})
		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})
		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	// check if user is the same as our requested user
	reqID := c.Param("user_id")
	if reqID != userId {
		// check token is admin from account_type in token
		if parsedToken.Claims.(jwt.MapClaims)["account_type"] != "admin" {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})
			c.Abort()
			return
		}
	}

	// get the video id from the request
	videoId := c.Param("video_id")

	// Get the user's completed videos playlist.
	playlists, err := database.GetPlaylistsByUserID(userId, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting playlists",
		})

		c.Abort()
		return
	}

	// check if the user has a completed videos playlist
	progVidPl := database.Playlist{}
	for _, v := range playlists {
		if v.Name == "Progress" {
			progVidPl = v
			break
		}
	}

	if progVidPl.Name == "" {
		// create a progress playlist
		data := database.Playlist{Name: "Progress", UserID: userId, VideoIDs: "[]"}
		pid, err := database.CreatePlaylist(data, db)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Error creating playlist",
			})

			c.Abort()
			return
		}
		data.PlaylistID = pid
		progVidPl = data
	}

	// get userData.Progress and parse it as json with the format: [{"videoId": "watchTime"}, {"videoId": "watchTime"}]
	// ensure it's an array
	var progress []database.ProgressToken
	err = json.Unmarshal([]byte(progVidPl.VideoIDs), &progress)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user data",
		})
		c.Abort()
		return
	}

	// check if the video id is in the progress array
	// if it is, update the watch time
	// if it isn't, add it to the array
	found := false
	for i, v := range progress {
		if v.VideoID == videoId {
			progress[i].Progress = currentTime
			found = true
			break
		}
	}

	if !found {
		progress = append(progress, database.ProgressToken{VideoID: videoId, Progress: currentTime})
	}

	// convert progress back to json
	progressJson, err := json.Marshal(progress)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user data",
		})
		c.Abort()
		return
	}

	// update the user's progress which is in database which is func UpdateUserProgress(user User, db *gorm.DB) error {}
	err = database.UpdateUserProgress(userId, string(progressJson), db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating user progress",
		})
		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"ret":  200,
		"data": "Progress updated",
	})

}

func apiGetWatchTime(c *gin.Context) {
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})
		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})
		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	reqID := c.Param("user_id")
	if reqID != userId {
		// check token is admin from account_type in token
		if parsedToken.Claims.(jwt.MapClaims)["account_type"] != "admin" {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})
			c.Abort()
			return
		}
	}

	// get the video id from the request
	videoId := c.Param("videoId")

	// Get the user's completed videos playlist.
	playlists, err := database.GetPlaylistsByUserID(userId, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting playlists",
		})

		c.Abort()
		return
	}

	// check if the user has a completed videos playlist
	progVidPl := database.Playlist{}
	for _, v := range playlists {
		if v.Name == "Progress" {
			progVidPl = v
			break
		}
	}

	if progVidPl.Name == "" {
		// create a progress playlist
		data := database.Playlist{Name: "Progress", UserID: userId, VideoIDs: "[]"}
		pid, err := database.CreatePlaylist(data, db)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Error creating playlist",
			})

			c.Abort()
			return
		}
		data.PlaylistID = pid
		progVidPl = data
	}

	// get userData.Progress and parse it as json with the format: [{"videoId": "watchTime"}]
	var progress map[string]interface{}
	err = json.Unmarshal([]byte(progVidPl.VideoIDs), &progress)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user data",
		})
		c.Abort()
		return
	}

	// get the watch time from the progress
	watchTime := progress[videoId]

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"progress": watchTime,
		},
	})

}

func apiMarkVideoComplete(c *gin.Context) {
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})

		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})

		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	reqID := c.Param("user_id")
	if reqID != userId {
		// check token is admin from account_type in token
		if parsedToken.Claims.(jwt.MapClaims)["account_type"] != "admin" {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})

			c.Abort()
			return
		}
	}

	// get the video id from the request
	videoId := c.Param("video_id")

	// check if video id is empty
	if videoId == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Video id is empty",
		})

		c.Abort()
		return
	}

	// Get the user's completed videos playlist.
	playlists, err := database.GetPlaylistsByUserID(userId, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting playlists",
		})

		c.Abort()
		return
	}

	// check if the user has a completed videos playlist
	compVidPl := database.Playlist{}
	progVidPl := database.Playlist{}
	for _, v := range playlists {
		if v.Name == "Completed Videos" {
			compVidPl = v
		} else if v.Name == "Progress" {
			progVidPl = v
		}
		if compVidPl.Name != "" && progVidPl.Name != "" {
			break
		}
	}

	// Check completed video playlist
	if compVidPl.Name == "" {
		// create a completed videos playlist
		data := database.Playlist{Name: "Completed Videos", UserID: userId, VideoIDs: "[]"}
		pid, err := database.CreatePlaylist(data, db)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Error creating playlist",
			})

			c.Abort()
			return
		}
		data.PlaylistID = pid
		compVidPl = data
	}

	// Check progress now
	if progVidPl.Name == "" {
		// create a progress playlist
		data := database.Playlist{Name: "Progress", UserID: userId, VideoIDs: "[]"}
		pid, err := database.CreatePlaylist(data, db)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{
				"ret": 500,
				"err": "Error creating playlist",
			})

			c.Abort()
			return
		}
		data.PlaylistID = pid
		progVidPl = data
	}

	// get compVidPl.VideoIDs and parse it as json with the format: ["${video_id}"]
	var completed_videos []string
	err = json.Unmarshal([]byte(compVidPl.VideoIDs), &completed_videos)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error parsing completed videos",
		})

		c.Abort()
		return
	}

	// check if the video is already in the array
	found := false
	for _, v := range completed_videos {
		if v == videoId {
			found = true
			break
		}
	}

	// if the video is not in the array, add it
	if !found {
		completed_videos = append(completed_videos, videoId)
	}

	// convert completed_videos back to json
	completed_videosJson, err := json.Marshal(completed_videos)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error parsing completed videos",
		})

		c.Abort()
		return
	}

	// update the user's progress which is in database which is func UpdateUserProgress(user User, db *gorm.DB) error {}
	err = database.UpdatePlaylistByUserId(userId, "Completed Videos", string(completed_videosJson), db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating completed videos",
		})

		c.Abort()
		return
	}

	// remove video from progress. Loop through progress, check if videoId is equal to videoId in request, if it is, remove it
	var progress []database.ProgressToken
	err = json.Unmarshal([]byte(progVidPl.VideoIDs), &progress)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user data",
		})
		c.Abort()
		return
	}

	// loop through progress
	for i, v := range progress {
		// check if videoId is equal to videoId in request
		if v.VideoID == videoId {
			// remove it
			progress = append(progress[:i], progress[i+1:]...)
			break
		}
	}

	// convert progress back to json
	progressJson, err := json.Marshal(progress)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error parsing progress",
		})

		c.Abort()
		return
	}

	// update the user's progress which is in database which is func UpdateUserProgress(user User, db *gorm.DB) error {}
	err = database.UpdateUserProgress(userId, string(progressJson), db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating progress",
		})

		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Video marked as complete",
		},
	})
}

func apiUpdateSponsorblock(c *gin.Context) {
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})

		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})

		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	// get the sponsorblockEnabled from the request
	sponsorblockEnabled := c.PostForm("sponsorblockEnabled")

	// check if sponsorblockEnabled is empty
	if sponsorblockEnabled == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Sponsorblock enabled is empty",
		})

		c.Abort()
		return
	}

	// convert to bool
	bool_sponsorblockEnabled, err := strconv.ParseBool(sponsorblockEnabled)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Sponsorblock enabled is not a boolean",
		})

		c.Abort()
		return
	}

	// update the user's sponsorblockEnabled which is in database which is func UpdateUserSponsorblockEnabled(user User, db *gorm.DB) error {}
	err = database.UpdateUserSponsorblockEnabled(userId, bool_sponsorblockEnabled, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating sponsorblock enabled",
		})

		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Sponsorblock enabled updated",
		},
	})
}

func apiAddSubscription(c *gin.Context) {
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})

		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})

		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	reqID := c.Param("user_id")
	if reqID != userId {
		// check token is admin from account_type in token
		if parsedToken.Claims.(jwt.MapClaims)["account_type"] != "admin" {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})

			c.Abort()
			return
		}
	}

	// get the video id from the request
	creatorID := c.Param("creator_id")

	// check if video id is empty
	if creatorID == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Video id is empty",
		})

		c.Abort()
		return
	}

	// Get the user's subscriptions playlist.
	subs, err := database.GetPlaylistByUserID(userId, "Subscriptions", db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting subscriptions",
		})

		c.Abort()
		return
	}

	// check if the user has a subscriptions playlist

	// check if the video is already in the array
	found := false
	for _, v := range subs {
		if v == creatorID {
			found = true
			break
		}
	}

	// if the video is not in the array, add it
	if !found {
		subs = append(subs, creatorID)
	}

	// convert completed_videos back to json
	vidIDJson, err := json.Marshal(subs)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error parsing subscrtiptions",
		})

		c.Abort()
		return
	}

	// update the user's progress which is in database which is func UpdateUserProgress(user User, db *gorm.DB) error {}
	err = database.UpdatePlaylistByUserId(userId, "Subscriptions", string(vidIDJson), db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating subscriptions",
		})

		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Added subscription",
		},
	})
}

func apiRemoveSubscription(c *gin.Context) {
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})

		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})

		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	reqID := c.Param("user_id")
	if reqID != userId {
		// check token is admin from account_type in token
		if parsedToken.Claims.(jwt.MapClaims)["account_type"] != "admin" {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})

			c.Abort()
			return
		}
	}

	// get the video id from the request
	creatorID := c.Param("creator_id")

	// check if video id is empty
	if creatorID == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Video id is empty",
		})

		c.Abort()
		return
	}

	// Get the user's subscriptions playlist.
	subs, err := database.GetPlaylistByUserID(userId, "Subscriptions", db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting subscriptions",
		})

		c.Abort()
		return
	}

	// check if the user has a subscriptions playlist

	// check if the video is already in the array
	found := false
	for _, v := range subs {
		if v == creatorID {
			found = true
			break
		}
	}

	// if the video is not in the array, remove it
	if found {
		for i, v := range subs {
			if v == creatorID {
				subs = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	// convert completed_videos back to json
	vidIDJson, err := json.Marshal(subs)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error parsing subscrtiptions",
		})

		c.Abort()
		return
	}

	// update the user's progress which is in database which is func UpdateUserProgress(user User, db *gorm.DB) error {}
	err = database.UpdatePlaylistByUserId(userId, "Subscriptions", string(vidIDJson), db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating subscriptions",
		})

		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Added subscription",
		},
	})
}

func apiUpdatePassword(c *gin.Context) {
	var isAdmin bool
	reqUser := c.Param("user_id")
	// get the token from the header
	token, err := jwttoken.GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "No token provided",
		})

		c.Abort()
		return
	}

	// parse the token
	parsedToken, err := jwttoken.ParseToken(token, settings.JwtSecret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret": 401,
			"err": "Invalid token signature",
		})

		c.Abort()
		return
	}

	// get the user id from the token
	userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)

	// Get user from database
	curUser, err := database.GetUserByID(userId, db.Select("user_id", "account_type"))
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user",
		})
		c.Abort()
		return
	}

	if curUser.AccountType == "admin" {
		isAdmin = true
	}

	// Check if request user is the same as the user in the token
	if reqUser != userId {
		// check token is admin from account_type in token
		if !isAdmin {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Unauthorized",
			})

			c.Abort()
			return
		}
	}

	// Get user from database
	user, err := database.GetUserByID(userId, db.Select("user_id", "password"))
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error getting user",
		})
		c.Abort()
		return
	}

	// parse json
	bodyAsByteArray, _ := io.ReadAll(c.Request.Body)
	// convert json to array { oldPassword, newPassword }
	var passwords struct{ OldPassword, NewPassword string }
	err = json.Unmarshal(bodyAsByteArray, &passwords)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Invalid request body",
		})
		c.Abort()
		return
	}

	// check if passwords is empty
	if passwords.NewPassword == "" {
		c.AbortWithStatusJSON(400, gin.H{
			"ret": 400,
			"err": "Password is empty",
		})

		c.Abort()
		return
	}

	// check if old password is correct
	if !database.CheckPasswordHash(passwords.OldPassword, user.Password) {
		if !isAdmin || (isAdmin && reqUser == userId) {
			c.AbortWithStatusJSON(400, gin.H{
				"ret": 400,
				"err": "Old password is incorrect",
			})

			c.Abort()
			return
		}
	}

	// hash the new password
	hashedPassword, err := database.HashPassword(passwords.NewPassword)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error hashing password",
		})
		c.Abort()
		return
	}

	// update the user's password
	user.Password = hashedPassword
	err = database.UpdateUser(user, db)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret": 500,
			"err": "Error updating password",
		})
		c.Abort()
		return
	}

	// Revoke jwt token
	jwttoken.DeleteToken(c)

	if reqUser != userId {
		log.Warn("Admin " + userId + " updated the password for user " + reqUser)
	} else {
		log.Info("User " + userId + " updated their password")
	}

	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"message": "Password updated",
		},
	})
}
