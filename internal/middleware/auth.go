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

func Auth(jwtService auth.JWTService, db *pgxpool.Pool) gin.HandlerFunc {
	queries := sqlcgen.New(db)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token was not provided"})
			c.Abort()
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		claims, err := jwtService.Validate(parts[1])
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
