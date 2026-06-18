package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

var errInvalidTokenFormat = errors.New("invalid token format")

func Auth(jwtService auth.JWTService, db *pgxpool.Pool) gin.HandlerFunc {
	queries := sqlcgen.New(db)

	return func(c *gin.Context) {
		tokenString, err := resolveAccessToken(c)
		if err != nil {
			if errors.Is(err, errInvalidTokenFormat) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
				c.Abort()
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token was not provided"})
			c.Abort()
			return
		}

		claims, err := jwtService.Validate(tokenString)
		if err != nil || claims.UserUUID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		userUUID, err := toPgUUID(claims.UserUUID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		user, err := queries.GetAuthenticatedUserByUUID(c.Request.Context(), userUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not resolve authenticated user"})
			c.Abort()
			return
		}

		resolvedUUID, err := fromPgUUID(user.UserUuid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not resolve authenticated user"})
			c.Abort()
			return
		}

		c.Set(contextUserIDKey, user.UserID)
		c.Set(contextRoleIDKey, user.RoleID)
		c.Set(contextRoleNameKey, user.RoleName)
		c.Set(contextUserUUIDKey, resolvedUUID)
		c.Set(contextUserEmailKey, user.Email)

		c.Next()
	}
}

func OptionalAuth(jwtService auth.JWTService, db *pgxpool.Pool) gin.HandlerFunc {
	queries := sqlcgen.New(db)

	return func(c *gin.Context) {
		tokenString, err := resolveAccessToken(c)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtService.Validate(tokenString)
		if err != nil || claims.UserUUID == "" {
			c.Next()
			return
		}

		userUUID, err := toPgUUID(claims.UserUUID)
		if err != nil {
			c.Next()
			return
		}

		user, err := queries.GetAuthenticatedUserByUUID(c.Request.Context(), userUUID)
		if err != nil {
			c.Next()
			return
		}

		resolvedUUID, err := fromPgUUID(user.UserUuid)
		if err != nil {
			c.Next()
			return
		}

		c.Set(contextUserIDKey, user.UserID)
		c.Set(contextRoleIDKey, user.RoleID)
		c.Set(contextRoleNameKey, user.RoleName)
		c.Set(contextUserUUIDKey, resolvedUUID)
		c.Set(contextUserEmailKey, user.Email)

		c.Next()
	}
}

func resolveAccessToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return "", errInvalidTokenFormat
		}

		return parts[1], nil
	}

	token, err := c.Cookie("spazio_access_token")
	if err != nil || token == "" {
		return "", errors.New("access token missing")
	}

	return token, nil
}

func toPgUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse uuid: %w", err)
	}

	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func fromPgUUID(value pgtype.UUID) (string, error) {
	if !value.Valid {
		return "", errors.New("uuid is null")
	}

	parsed, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return "", fmt.Errorf("format uuid: %w", err)
	}

	return parsed.String(), nil
}
