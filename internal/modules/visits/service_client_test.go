package visits

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestService_ScheduleVisit(t *testing.T) {
	ctx := context.Background()
	clientID := int32(10)
	propertyID := int32(1)
	loc, _ := time.LoadLocation("America/Mexico_City")
	// Normalize base date for tests
	visitDate := time.Now().In(loc).Add(96 * time.Hour).Truncate(time.Hour)

	tests := []struct {
		name        string
		visitDate   time.Time
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:        "error when date is too close (< 48h)",
			visitDate:   time.Now().In(loc).Add(24 * time.Hour),
			setupRepo:   func() *mockVisitsRepository { return &mockVisitsRepository{} },
			wantErr:     true,
			errContains: "debe agendar con al menos 48 horas",
		},
		{
			name:        "error when date is too far (> 30 days)",
			visitDate:   time.Now().In(loc).Add(32 * 24 * time.Hour),
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
						return 0, errors.New("el cliente no está activo o no existe")
					},
				}
			},
			wantErr:     true,
			errContains: "el cliente no está activo o no existe",
		},
		{
			name:      "error when primary agent fails",
			visitDate: visitDate,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					checkUserActiveFunc: func(ctx context.Context, userID int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) {
						return 0, errors.New("agent db error")
					},
				}
			},
			wantErr:     true,
			errContains: "la propiedad no tiene un agente asignado disponible",
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
						return []sqlcgen.GetAgentScheduleRow{
							{
								DayOfWeek: int16(visitDate.Weekday()),
								StartTime: pgtype.Time{Valid: true},
								EndTime:   pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true},
							},
						}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, pid int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
						return nil, nil
					},
					getOccupiedVisitsFunc: func(ctx context.Context, aid int32, start, end time.Time) ([]time.Time, error) {
						// Slot is occupied
						return []time.Time{visitDate}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "el horario seleccionado ya no está disponible",
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
								StartTime: pgtype.Time{Valid: true},
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
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("expected %v to contain %v", err.Error(), tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestService_RescheduleVisit(t *testing.T) {
	ctx := context.Background()
	userID := int32(10)
	visitUUID := uuid.New()
	loc, _ := time.LoadLocation("America/Mexico_City")
	newDate := time.Now().In(loc).Add(120 * time.Hour).Truncate(time.Hour)

	tests := []struct {
		name        string
		role        int32
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name: "error when transaction begin fails",
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) {
						return nil, errors.New("tx error")
					},
				}
			},
			wantErr:     true,
			errContains: "tx error",
		},
		{
			name: "error when visit not found",
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{}, pgx.ErrNoRows
					},
				}
			},
			wantErr:     true,
			errContains: "visita no encontrada",
		},
		{
			name: "error when already cancelled",
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{StatusID: StatusCancelled}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no se puede reagendar una cita ya cancelada",
		},
		{
			name: "error when unauthorized client",
			role: 3, // Client
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{ClientID: 999, StatusID: StatusPending}, nil
					},
					checkUserActiveFunc: func(ctx context.Context, uid int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, pid int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no tienes permiso",
		},
		{
			name: "error when update visit status fails",
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, ClientID: userID, PropertyID: 1, StatusID: StatusPending}, nil
					},
					// scheduleVisitInternal mock
					checkUserActiveFunc: func(ctx context.Context, uid int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, pid int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, pid int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, aid int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return []sqlcgen.GetAgentScheduleRow{{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}}}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, pid int32, s, e time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
						return nil, nil
					},
					getOccupiedVisitsFunc: func(ctx context.Context, aid int32, s, e time.Time) ([]time.Time, error) {
						return nil, nil
					},
					createVisitFunc: func(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{}, nil
					},
					// Failing update
					updateVisitStatusFunc: func(ctx context.Context, vid, sid int32) error {
						return errors.New("update fail")
					},
				}
			},
			wantErr:     true,
			errContains: "fallo al cancelar cita anterior",
		},
		{
			name: "success reschedule (admin)",
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, ClientID: userID, PropertyID: 1, StatusID: StatusPending}, nil
					},
					// scheduleVisitInternal mock
					checkUserActiveFunc: func(ctx context.Context, uid int32) (int32, error) { return 1, nil },
					getPropertyStatusAndCheckDeletedFunc: func(ctx context.Context, pid int32) (sqlcgen.GetPropertyStatusAndCheckDeletedRow, error) {
						return sqlcgen.GetPropertyStatusAndCheckDeletedRow{StatusID: PropertyStatusAvailable}, nil
					},
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, pid int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, aid int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return []sqlcgen.GetAgentScheduleRow{{DayOfWeek: int16(newDate.Weekday()), StartTime: pgtype.Time{Valid: true}, EndTime: pgtype.Time{Microseconds: 24 * 3600 * 1e6, Valid: true}}}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, pid int32, s, e time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
						return nil, nil
					},
					getOccupiedVisitsFunc: func(ctx context.Context, aid int32, s, e time.Time) ([]time.Time, error) {
						return nil, nil
					},
					createVisitFunc: func(ctx context.Context, arg sqlcgen.CreateVisitParams) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{
							VisitUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true},
						}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)

			testRole := int32(1) // Default to Admin
			if tt.role != 0 {
				testRole = tt.role
			}

			_, err := svc.RescheduleVisit(ctx, userID, testRole, visitUUID, newDate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("expected %v to contain %v", err.Error(), tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
