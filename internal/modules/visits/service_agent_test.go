package visits

import (
	"context"
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
			name:   "success confirm as client (pending -> waiting agent)",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)
			err := svc.ConfirmVisit(ctx, userID, tt.roleID, visitUUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
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
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestService_CancelVisit(t *testing.T) {
	ctx := context.Background()
	visitUUID := uuid.New()
	userID := int32(10)

	tests := []struct {
		name         string
		userID       int32
		roleID       int32
		setupRepo    func() *mockVisitsRepository
		wantErr      bool
		errSubstring string
	}{
		{
			name:   "success as client",
			userID: userID,
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 100, ClientID: userID, StatusID: StatusPending}, nil
					},
					updateVisitStatusFunc: func(ctx context.Context, vid, sid int32) error {
						return nil
					},
					createVisitStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateVisitStatusHistoryParams) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "fail as admin",
			userID: userID,
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 100, StatusID: StatusPending}, nil
					},
				}
			},
			wantErr:      true,
			errSubstring: "rol no autorizado",
		},
		{
			name:   "fail wrong client",
			userID: userID,
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 100, ClientID: 999, StatusID: StatusPending}, nil
					},
				}
			},
			wantErr:      true,
			errSubstring: "no tienes permiso",
		},
		{
			name:   "fail confirmed visit",
			userID: userID,
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getVisitByUUIDFunc: func(ctx context.Context, vid uuid.UUID) (sqlcgen.Visit, error) {
						return sqlcgen.Visit{VisitID: 100, ClientID: userID, StatusID: StatusConfirmed}, nil
					},
				}
			},
			wantErr:      true,
			errSubstring: "solo se pueden cancelar visitas que no han sido confirmadas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)
			err := svc.CancelVisit(ctx, tt.userID, tt.roleID, visitUUID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstring)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
