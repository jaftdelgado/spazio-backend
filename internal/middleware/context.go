package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
)

const (
	contextUserIDKey    = "user_id"
	contextRoleIDKey    = "role_id"
	contextRoleNameKey  = "user_role"
	contextUserUUIDKey  = "user_uuid"
	contextUserEmailKey = "user_email"
)

var errMissingAuthenticatedContext = errors.New("authenticated context is missing")

func AuthenticatedUserID(c *gin.Context) (int32, error) {
	return getInt32ContextValue(c, contextUserIDKey)
}

func AuthenticatedRoleID(c *gin.Context) (int32, error) {
	return getInt32ContextValue(c, contextRoleIDKey)
}

func AuthenticatedRoleName(c *gin.Context) (string, error) {
	return getStringContextValue(c, contextRoleNameKey)
}

func AuthenticatedUserUUID(c *gin.Context) (string, error) {
	return getStringContextValue(c, contextUserUUIDKey)
}

func AuthenticatedUserEmail(c *gin.Context) (string, error) {
	return getStringContextValue(c, contextUserEmailKey)
}

func getInt32ContextValue(c *gin.Context, key string) (int32, error) {
	value, exists := c.Get(key)
	if !exists || value == nil {
		return 0, errMissingAuthenticatedContext
	}

	intValue, ok := value.(int32)
	if !ok || intValue <= 0 {
		return 0, errMissingAuthenticatedContext
	}

	return intValue, nil
}

func getStringContextValue(c *gin.Context, key string) (string, error) {
	value, exists := c.Get(key)
	if !exists || value == nil {
		return "", errMissingAuthenticatedContext
	}

	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return "", errMissingAuthenticatedContext
	}

	return stringValue, nil
}
