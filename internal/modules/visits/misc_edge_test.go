package visits

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestService_ValidateEntityIntegrity_Deleted(t *testing.T) {
	ctx := context.Background()
	repo := &mockVisitsRepository{
		checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
		getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
			return sqlcgen.GetPropertyStatusAndCheckDeletedRow{
				StatusID:  PropertyStatusAvailable,
				DeletedAt: pgtype.Timestamptz{Valid: true}, // Marked as deleted
			}, nil
		},
	}
	svc := NewService(repo).(*service)

	err := svc.validateEntityIntegrity(ctx, repo, 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "eliminada")
}

func TestService_ValidateEntityIntegrity_NotAvailable(t *testing.T) {
	ctx := context.Background()
	repo := &mockVisitsRepository{
		checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
		getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
			return sqlcgen.GetPropertyStatusAndCheckDeletedRow{
				StatusID: 1, // Not available
			}, nil
		},
	}
	svc := NewService(repo).(*service)

	err := svc.validateEntityIntegrity(ctx, repo, 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no está disponible")
}

func TestHandler_ValidateCreateVisitRequest(t *testing.T) {
	req := CreateVisitRequest{
		PropertyID: 1,
		VisitDate:  time.Now().Add(48 * time.Hour).Truncate(time.Hour).Add(15 * time.Minute), // Invalid minutes
	}

	err := validateCreateVisitRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minuto :00")
}

func TestHandler_ListVisits_InvalidParsing(t *testing.T) {
	h := NewHandler(&mockVisitsService{})
	rec, ctx := newHandlerTestContext(http.MethodGet, "/visits?status_id=abc&property_id=def&date=bad-date")
	setAuthenticatedContext(ctx, 10, 3)

	// Since strconv.Atoi ignores errors and puts 0, it shouldn't fail, but let's test the path
	h.listVisits(ctx)

	assert.Equal(t, http.StatusOK, rec.Code)
}
