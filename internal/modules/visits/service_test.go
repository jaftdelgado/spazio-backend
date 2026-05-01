package visits

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockRepo struct {
	Repository
	agentID        int32
	schedules      []sqlcgen.GetAgentScheduleRow
	exceptions     []sqlcgen.GetPropertyExceptionsRow
	occupied       []time.Time
	visit          sqlcgen.Visit
	role           int32
	propStatus     int32
	isDeleted      bool
	err            error
	updateCalled   bool
}

func (m *mockRepo) GetPrimaryAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	return m.agentID, m.err
}
func (m *mockRepo) GetAgentSchedule(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
	return m.schedules, m.err
}
func (m *mockRepo) GetPropertyExceptions(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
	return m.exceptions, m.err
}
func (m *mockRepo) GetOccupiedVisits(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
	return m.occupied, m.err
}
func (m *mockRepo) GetUserRole(ctx context.Context, userID int32) (int32, error) {
	return m.role, m.err
}
func (m *mockRepo) GetVisitByUUID(ctx context.Context, visitUUID uuid.UUID) (sqlcgen.Visit, error) {
	return m.visit, m.err
}
func (m *mockRepo) CreateVisit(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
	return m.visit, m.err
}
func (m *mockRepo) UpdateVisitStatus(ctx context.Context, visitID int32, statusID int32) error {
	m.updateCalled = true
	return m.err
}
func (m *mockRepo) GetPropertyStatusAndCheckDeleted(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
	var del pgtype.Timestamptz
	if m.isDeleted { del = pgtype.Timestamptz{Time: time.Now(), Valid: true} }
	return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: m.propStatus, DeletedAt: del}, m.err
}
func (m *mockRepo) CheckUserActive(ctx context.Context, userID int32) (int32, error) {
	return userID, m.err
}
func (m *mockRepo) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *mockRepo) WithTx(tx pgx.Tx) Repository { return m }
func (m *mockRepo) CreateVisitStatusHistory(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil }

func TestScheduleVisitServiceHarden(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	
	tests := []struct {
		name      string
		visitDate time.Time
		repo      *mockRepo
		wantErr   string
	}{
		{
			name:      "property not available",
			visitDate: now.Add(72 * time.Hour),
			repo:      &mockRepo{propStatus: 3}, // Sold
			wantErr:   "la propiedad no está disponible para recibir visitas en este momento",
		},
		{
			name:      "property deleted",
			visitDate: now.Add(72 * time.Hour),
			repo:      &mockRepo{propStatus: 2, isDeleted: true},
			wantErr:   "la propiedad ya no está disponible (eliminada)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(tt.repo)
			_, err := s.ScheduleVisit(context.Background(), 1, 1, tt.visitDate)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
