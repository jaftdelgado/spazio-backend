package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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

func TestUUIDConversionHelpers(t *testing.T) {
	t.Run("valid uuid roundtrip", func(t *testing.T) {
		want := "8a6fbb17-b64b-4f40-a09d-b6639b357ef5"

		pgUUID, err := toPgUUID(want)
		if err != nil {
			t.Fatalf("toPgUUID() error = %v", err)
		}

		got, err := fromPgUUID(pgUUID)
		if err != nil {
			t.Fatalf("fromPgUUID() error = %v", err)
		}
		if got != want {
			t.Fatalf("uuid roundtrip = %q, want %q", got, want)
		}
	})

	t.Run("invalid uuid", func(t *testing.T) {
		if _, err := toPgUUID("not-a-uuid"); err == nil {
			t.Fatal("expected invalid uuid error")
		}
	})
}
