package visits

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestService_ConfirmVisit(t *testing.T) {
	ctx := context.Background()
	visitUUID := uuid.New()
	userID := int32(10)

	tests := []struct {
		name        string
		roleID      int32
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:   "error when visit not found",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{}, errors.New("not found")
					},
				}
			},
			wantErr:     true,
			errContains: "visita no encontrada",
		},
		{
			name:   "error when client unauthorized",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{ClientID: 99}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no tienes permiso",
		},
		{
			name:   "error when agent unauthorized",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{AgentID: pgtype.Int4{Int32: 99, Valid: true}}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no eres el agente asignado",
		},
		{
			name:   "error when already confirmed",
			roleID: 1, // Admin
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{StatusID: StatusConfirmed}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "operación no válida o ya confirmada",
		},
		{
			name:   "success confirm from client (Pending to WaitingAgent)",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, ClientID: userID, StatusID: StatusPending}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
				}
			},
			wantErr: false,
		},
		{
			name:   "success confirm from client (WaitingClient to Confirmed)",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, ClientID: userID, StatusID: StatusWaitingClient}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
				}
			},
			wantErr: false,
		},
		{
			name:   "success confirm from agent (Pending to WaitingClient)",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: userID, Valid: true}, StatusID: StatusPending}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
				}
			},
			wantErr: false,
		},
		{
			name:   "success confirm from agent (WaitingAgent to Confirmed)",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: userID, Valid: true}, StatusID: StatusWaitingAgent}, nil
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

			err := svc.ConfirmVisit(ctx, userID, tt.roleID, visitUUID)

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

func TestService_CompleteVisit(t *testing.T) {
	ctx := context.Background()
	visitUUID := uuid.New()
	userID := int32(10)

	tests := []struct {
		name        string
		roleID      int32
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:   "error when client tries to complete",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "solo el agente o administrador",
		},
		{
			name:   "error when wrong agent tries to complete",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{AgentID: pgtype.Int4{Int32: 99, Valid: true}}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no eres el agente",
		},
		{
			name:   "error when visit not confirmed",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{StatusID: StatusPending}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "solo se pueden completar visitas que estén confirmadas",
		},
		{
			name:   "error when visit not found",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{}, errors.New("not found db error")
					},
				}
			},
			wantErr:     true,
			errContains: "visita no encontrada",
		},
		{
			name:   "error when update status fails",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return errors.New("update db err") },
				}
			},
			wantErr:     true,
			errContains: "update db err",
		},
		{
			name:   "error when create history fails",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return errors.New("history db err") },
				}
			},
			wantErr:     true,
			errContains: "history db err",
		},
		{
			name:   "success complete",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, StatusID: StatusConfirmed}, nil
					},
					updateVisitStatusFunc:        func(ctx context.Context, vid, sid int32) error { return nil },
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error { return nil },
				}
			},
			wantErr: false,
		},
		{
			name:   "success complete (agent)",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 1, AgentID: pgtype.Int4{Int32: userID, Valid: true}, StatusID: StatusConfirmed}, nil
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

			err := svc.CompleteVisit(ctx, userID, tt.roleID, visitUUID)

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
