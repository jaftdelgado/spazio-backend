package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateSupabaseToken(t *testing.T) {
	t.Run("invalid response status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		_, err := validateSupabaseToken(context.Background(), server.Client(), server.URL, "anon", "token")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("valid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("authorization header = %q", got)
			}
			if got := r.Header.Get("apikey"); got != "anon" {
				t.Fatalf("apikey header = %q", got)
			}

			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":    "uuid-123",
				"email": "user@example.com",
			})
		}))
		defer server.Close()

		identity, err := validateSupabaseToken(context.Background(), server.Client(), server.URL, "anon", "token")
		if err != nil {
			t.Fatalf("validateSupabaseToken() error = %v", err)
		}
		if identity.UserUUID != "uuid-123" || identity.Email != "user@example.com" {
			t.Fatalf("unexpected identity: %+v", identity)
		}
	})
}

func TestAuthenticatedContextHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing keys", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		if _, err := AuthenticatedUserID(c); err == nil {
			t.Fatal("expected user_id error")
		}
		if _, err := AuthenticatedRoleID(c); err == nil {
			t.Fatal("expected role_id error")
		}
		if _, err := AuthenticatedRoleName(c); err == nil {
			t.Fatal("expected user_role error")
		}
		if _, err := AuthenticatedUserUUID(c); err == nil {
			t.Fatal("expected user_uuid error")
		}
		if _, err := AuthenticatedUserEmail(c); err == nil {
			t.Fatal("expected user_email error")
		}
	})

	t.Run("invalid types", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(contextUserIDKey, "1")
		c.Set(contextRoleIDKey, "2")
		c.Set(contextRoleNameKey, 1)
		c.Set(contextUserUUIDKey, 99)
		c.Set(contextUserEmailKey, 88)

		if _, err := AuthenticatedUserID(c); err == nil {
			t.Fatal("expected user_id type error")
		}
		if _, err := AuthenticatedRoleID(c); err == nil {
			t.Fatal("expected role_id type error")
		}
		if _, err := AuthenticatedRoleName(c); err == nil {
			t.Fatal("expected user_role type error")
		}
		if _, err := AuthenticatedUserUUID(c); err == nil {
			t.Fatal("expected user_uuid type error")
		}
		if _, err := AuthenticatedUserEmail(c); err == nil {
			t.Fatal("expected user_email type error")
		}
	})

	t.Run("valid values", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(contextUserIDKey, int32(10))
		c.Set(contextRoleIDKey, int32(2))
		c.Set(contextRoleNameKey, "agent")
		c.Set(contextUserUUIDKey, "uuid-123")
		c.Set(contextUserEmailKey, "user@example.com")

		userID, err := AuthenticatedUserID(c)
		if err != nil || userID != 10 {
			t.Fatalf("AuthenticatedUserID() = %d, %v", userID, err)
		}

		roleID, err := AuthenticatedRoleID(c)
		if err != nil || roleID != 2 {
			t.Fatalf("AuthenticatedRoleID() = %d, %v", roleID, err)
		}

		roleName, err := AuthenticatedRoleName(c)
		if err != nil || roleName != "agent" {
			t.Fatalf("AuthenticatedRoleName() = %q, %v", roleName, err)
		}

		userUUID, err := AuthenticatedUserUUID(c)
		if err != nil || userUUID != "uuid-123" {
			t.Fatalf("AuthenticatedUserUUID() = %q, %v", userUUID, err)
		}

		userEmail, err := AuthenticatedUserEmail(c)
		if err != nil || userEmail != "user@example.com" {
			t.Fatalf("AuthenticatedUserEmail() = %q, %v", userEmail, err)
		}
	})
}
