package visits

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewModule(t *testing.T) {
	m := NewModule(&pgxpool.Pool{})
	if m == nil {
		t.Errorf("expected not nil")
	}
	if m.Handler == nil {
		t.Errorf("expected not nil")
	}

	r := gin.New()
	group := r.Group("/test")
	m.RegisterRoutes(group)
}
