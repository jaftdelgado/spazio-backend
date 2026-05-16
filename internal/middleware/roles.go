package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, _ := c.Get("user_role")
		roleStr, ok := userRole.(string)

		if ok {
			for _, role := range allowedRoles {
				if strings.EqualFold(roleStr, role) {
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to perform this action"})
		c.Abort()
	}
}
