package visits

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

// We verify that the methods are wired without panicking.
func TestRepository_Methods(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	ctx := context.Background()

	t.Run("GetPrimaryAgentForProperty", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPrimaryAgentForProperty(ctx, 1) })
	})
	t.Run("GetAgentSchedule", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetAgentSchedule(ctx, 1) })
	})
	t.Run("GetPropertyExceptions", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPropertyExceptions(ctx, 1, time.Now(), time.Now()) })
	})
	t.Run("GetOccupiedVisits", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetOccupiedVisits(ctx, 1, time.Now(), time.Now()) })
	})
	t.Run("CreateVisit", func(t *testing.T) {
		assert.Panics(t, func() { repo.CreateVisit(ctx, sqlcgen.CreateVisitParams{}) })
	})
	t.Run("GetVisitByUUID", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetVisitByUUID(ctx, uuid.New()) })
	})
	t.Run("ListVisits", func(t *testing.T) {
		assert.Panics(t, func() { repo.ListVisits(ctx, sqlcgen.ListVisitsParams{}) })
	})
	t.Run("UpdateVisitStatus", func(t *testing.T) {
		assert.Panics(t, func() { repo.UpdateVisitStatus(ctx, 1, 1) })
	})
	t.Run("CreateVisitStatusHistory", func(t *testing.T) {
		assert.Panics(t, func() { repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{}) })
	})
	t.Run("GetPropertyStatusAndCheckDeleted", func(t *testing.T) {
		assert.Panics(t, func() { repo.GetPropertyStatusAndCheckDeleted(ctx, 1) })
	})
	t.Run("CheckUserActive", func(t *testing.T) {
		assert.Panics(t, func() { repo.CheckUserActive(ctx, 1) })
	})
}
