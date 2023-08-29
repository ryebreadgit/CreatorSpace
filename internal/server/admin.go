package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"gorm.io/gorm"
)

func page_user_management(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			// Redirect to login
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		user := userData.(database.User)

		// Get all users
		users, err := database.GetUsers(db.Order("id asc"))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"ret": 500,
				"err": err,
			})
			return
		}

		// For each user get their role
		for i, user := range users {
			role, err := database.GetUserByID(user.UserID, db)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
					"ret": 500,
					"err": err,
				})
				return
			}
			users[i].AccountType = role.AccountType
		}

		c.HTML(http.StatusOK, "user-management.tmpl", gin.H{
			"User":  user,
			"Users": users,
		})
	}
}
