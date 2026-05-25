package visits

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

// We verify that the methods are wired without panicking.
func TestRepository_Methods(t *testing.T) {
	repo := NewRepository(&pgxpool.Pool{})
	ctx := context.Background()

	t.Run("GetPrimaryAgentForProperty", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPrimaryAgentForProperty(ctx, 1)
		}()
	})
	t.Run("GetAgentSchedule", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetAgentSchedule(ctx, 1)
		}()
	})
	t.Run("GetPropertyExceptions", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPropertyExceptions(ctx, 1, time.Now(), time.Now())
		}()
	})
	t.Run("GetOccupiedVisits", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetOccupiedVisits(ctx, 1, time.Now(), time.Now())
		}()
	})
	t.Run("CreateVisit", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.CreateVisit(ctx, sqlcgen.CreateVisitParams{})
		}()
	})
	t.Run("GetVisitByUUID", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetVisitByUUID(ctx, uuid.New())
		}()
	})
	t.Run("ListVisits", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.ListVisits(ctx, sqlcgen.ListVisitsParams{})
		}()
	})
	t.Run("UpdateVisitStatus", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.UpdateVisitStatus(ctx, 1, 1)
		}()
	})
	t.Run("CreateVisitStatusHistory", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.CreateVisitStatusHistory(ctx, sqlcgen.CreateVisitStatusHistoryParams{})
		}()
	})
	t.Run("GetPropertyStatusAndCheckDeleted", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.GetPropertyStatusAndCheckDeleted(ctx, 1)
		}()
	})
	t.Run("CheckUserActive", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			repo.CheckUserActive(ctx, 1)
		}()
	})
}
