package jwttoken

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"gorm.io/gorm"
)

var db *gorm.DB
var settings *database.Settings

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the client secret from the settings
		secret := settings.JwtSecret

		// Get the token from the header
		token, err := GetToken(c)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		// Parse the token
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			// Make sure the token method conform to "SigningMethodHMAC"
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		// Check if the token is valid
		if err != nil || !parsedToken.Valid {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Invalid token signature",
			})
			c.Abort()
			return
		}

		// Check if the token is expired
		if time.Now().Unix() > int64(parsedToken.Claims.(jwt.MapClaims)["exp"].(float64)) {
			c.AbortWithStatusJSON(401, gin.H{
				"ret": 401,
				"err": "Token expired",
			})
			c.Abort()
			return
		}

		// check if passed refresh time, if so refresh token
		if time.Now().Unix() > int64(parsedToken.Claims.(jwt.MapClaims)["ref"].(float64)) {
			// get user id from token
			userId := parsedToken.Claims.(jwt.MapClaims)["user_id"].(string)
			// create new token
			token, err := CreateToken(userId)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{
					"ret": 401,
					"err": "Error creating new token",
				})
				c.Abort()
				return
			}
			// set new token
			SetToken(c, token)
			fmt.Println("refreshed token")
		}

		// Continue
		c.Next()
	}
}

// Get the token from the header
func GetToken(c *gin.Context) (string, error) {
	token, err := c.Cookie("jwt-token")
	if err != nil {
		return "", err
	}
	return token, nil
}

// Parse the token
func ParseToken(token string, secret string) (*jwt.Token, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return parsedToken, nil
}

// Set the token in the header
func SetToken(c *gin.Context, token string) {
	// 7 day expiration
	c.SetCookie("jwt-token", token, 86400*7, "/", "", false, true)
}

// Delete the token from the header
func DeleteToken(c *gin.Context) {
	c.SetCookie("jwt-token", "", -1, "/", "", false, true)
}

func IsLoggedIn(c *gin.Context) bool {
	// Get the client secret from the settings
	secret := settings.JwtSecret

	// Get the token from the header
	token, err := GetToken(c)
	if err != nil {
		return false
	}

	// Parse the token
	parsedToken, err := ParseToken(token, secret)
	if err != nil {
		return false
	}

	// check if token is valid
	if !parsedToken.Valid {
		return false
	}

	return true

}

func GetUserFromToken(c *gin.Context) (string, error) {
	// Get the client secret from the settings
	secret := settings.JwtSecret

	// Get the token from the header
	token, err := GetToken(c)
	if err != nil {
		return "", err
	}

	// Parse the token
	parsedToken, err := ParseToken(token, secret)
	if err != nil {
		return "", err
	}

	// check if token is valid
	if !parsedToken.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Get the user id from the token
	claims := parsedToken.Claims.(jwt.MapClaims)
	user := claims["user_id"].(string)

	return user, nil
}

// refresh token
func RefreshToken(c *gin.Context) {
	// get the client secret from the settings
	secret := settings.JwtSecret

	// get the token from the header
	token, err := GetToken(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret":   401,
			"error": "No token provided",
		})
		return
	}

	// parse the token
	parsedToken, err := ParseToken(token, secret)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{
			"ret":   401,
			"error": "Invalid token signature",
		})
		return
	}

	// check if token is valid
	if !parsedToken.Valid {
		c.AbortWithStatusJSON(401, gin.H{
			"ret":   401,
			"error": "Invalid token signature",
		})
		return
	}

	// get the user id from the token
	claims := parsedToken.Claims.(jwt.MapClaims)
	user := claims["user_id"].(string)

	// create a new token
	newToken, err := CreateToken(user)
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"ret":   500,
			"error": "Could not create token",
		})
		return
	}

	// set the new token in the header
	SetToken(c, newToken)

	// return the new token
	c.JSON(200, gin.H{
		"ret": 200,
		"data": gin.H{
			"token": newToken,
		},
	})
}

func CreateToken(user string) (string, error) {
	// get the client secret from the settings
	secret := settings.JwtSecret

	// Get user from database
	userdata, err := database.GetUserByID(user, db)
	if err != nil {
		return "", err
	}

	// create a new token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userdata.UserID,
		"username": userdata.Username,
		"role":     userdata.AccountType,
		"iss":      "creatorspace",                            // issuer
		"iat":      time.Now().Unix(),                         // issued at
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 24 * 7 hour expiration time = 7 day expiration
		"ref":      time.Now().Add(time.Hour * 1).Unix(),      // 1 hour refresh time
	})

	// sign the token
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
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
}
