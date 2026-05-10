package visits

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestNewModule(t *testing.T) {
	m := NewModule(&pgxpool.Pool{})
	assert.NotNil(t, m)
	assert.NotNil(t, m.Handler)

	r := gin.New()
	group := r.Group("/test")
	m.RegisterRoutes(group)
}
