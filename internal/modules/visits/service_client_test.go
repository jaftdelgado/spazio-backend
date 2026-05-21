package visits

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestService_ScheduleVisit(t *testing.T) {
	ctx := context.Background()
	clientID := int32(10)
	propertyID := int32(1)
	visitDate := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Hour) // Valid date (3 days later)

	tests := []struct {
		name        string
		visitDate   time.Time
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:        "error when date is too close (< 48h)",
			visitDate:   time.Now().UTC().Add(24 * time.Hour),
			setupRepo:   func() *mockVisitsRepository { return &mockVisitsRepository{} },
			wantErr:     true,
			errContains: "debe agendar con al menos 48 horas",
		},
		{
			name:        "error when date is too far (> 30 days)",
			visitDate:   time.Now().UTC().Add(32 * 24 * time.Hour),
			setupRepo:   func() *mockVisitsRepository { return &mockVisitsRepository{} },
			wantErr:     true,
			errContains: "no puede agendar con más de 30 días",
		},
		{
			name:      "error when user is inactive",
			visitDate: visitDate,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) {
						return 0, errors.New("inactive")
					},
				}
			},
			wantErr:     true,
			errContains: "el cliente no está activo o no existe",
		},
		{
			name:      "error when slot is unavailable",
			visitDate: visitDate,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return nil, nil // No schedules = no slots
					},
				}
			},
			wantErr:     true,
			errContains: "horario seleccionado ya no está disponible",
		},
		{
			name:      "success scheduling",
			visitDate: visitDate,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return []sqlcgen.GetAgentScheduleRow{
							{
								DayOfWeek: int16(visitDate.Weekday()),
								StartTime: pgtype.Time{Microseconds: 0, Valid: true},
								EndTime:   pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true},
							},
						}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, pid int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
						return nil, nil
					},
					getOccupiedVisitsFunc: func(ctx context.Context, aid int32, start, end time.Time) ([]time.Time, error) {
						return nil, nil
					},
					createVisitFunc: func(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{
							VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true},
							StatusID:  StatusPending,
						}, nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)

			_, err := svc.ScheduleVisit(ctx, clientID, propertyID, tt.visitDate)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_RescheduleVisit(t *testing.T) {
	ctx := context.Background()
	userID := int32(10)
	visitUUID := uuid.New()
	newDate := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Hour)

	tests := []struct {
		name        string
		roleID      int32
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:   "error when transaction begin fails",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return nil, errors.New("begin fail") },
				}
			},
			wantErr:     true,
			errContains: "fallo al iniciar transacción",
		},
		{
			name:   "error when visit not found",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					withTxFunc: func(tx pgx.Tx) Repository { return &mockVisitsRepository{
						getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
							return sqlcgen.Visit{}, errors.New("not found")
						},
					}},
				}
			},
			wantErr:     true,
			errContains: "visita no encontrada",
		},
		{
			name:   "error when already cancelled",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					withTxFunc: func(tx pgx.Tx) Repository { return &mockVisitsRepository{
						getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
							return sqlcgen.Visit{StatusID: StatusCancelled}, nil
						},
					}},
				}
			},
			wantErr:     true,
			errContains: "no se puede reagendar una cita ya cancelada",
		},
		{
			name:   "error when unauthorized client",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					withTxFunc: func(tx pgx.Tx) Repository { return &mockVisitsRepository{
						getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
							return sqlcgen.Visit{ClientID: 99}, nil // different client
						},
					}},
				}
			},
			wantErr:     true,
			errContains: "no tienes permiso para reagendar",
		},
		{
			name:   "success reschedule (admin)",
			roleID: 1, // Admin can always reschedule
			setupRepo: func() *mockVisitsRepository {
				txMock := &mockTx{}
				repoMock := &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return txMock, nil },
					// Need these on base repo because GetAvailableSlots uses s.repo
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return []sqlcgen.GetAgentScheduleRow{
							{DayOfWeek: 0, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 1, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 2, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 3, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 4, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 5, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
							{DayOfWeek: 6, StartTime: pgtype.Time{Microseconds: 0, Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}},
						}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, pid int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) { return nil, nil },
					getOccupiedVisitsFunc:     func(ctx context.Context, aid int32, start, end time.Time) ([]time.Time, error) { return nil, nil },
				}
				repoMock.withTxFunc = func(tx pgx.Tx) Repository {
					return &mockVisitsRepository{
						getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
							return sqlcgen.Visit{VisitID: 1, ClientID: 99, PropertyID: 1, StatusID: StatusPending}, nil
						},
						checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
						getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
							return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
						},
						getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
						createVisitFunc: func(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
							return sqlcgen.Visit{VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true}}, nil
						},
						updateVisitStatusFunc: func(ctx context.Context, visitID int32, statusID int32) error { return nil },
						createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
					}
				}
				return repoMock
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)

			_, err := svc.RescheduleVisit(ctx, userID, tt.roleID, visitUUID, newDate)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
