package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows configured origin from quoted env value", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, router := gin.CreateTestContext(recorder)
		router.Use(CORS(`"http://localhost:3000,https://myspazio.app"`))
		router.OPTIONS("/api/v1/users/login", func(c *gin.Context) {})

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/users/login", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		ctx.Request = req

		router.HandleContext(ctx)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}

		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
		}
	})

	t.Run("does not allow unknown origin", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, router := gin.CreateTestContext(recorder)
		router.Use(CORS("http://localhost:3000"))
		router.OPTIONS("/api/v1/users/login", func(c *gin.Context) {})

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/users/login", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		ctx.Request = req

		router.HandleContext(ctx)

		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
		}
	})
}
