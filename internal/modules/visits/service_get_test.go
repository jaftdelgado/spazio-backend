package visits

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestService_ListUserVisits(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		roleID      int32
		setupRepo   func() *mockVisitsRepository
		wantErr     bool
		errContains string
	}{
		{
			name:   "error when role unrecognized",
			roleID: 99,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{}
			},
			wantErr:     true,
			errContains: "rol de usuario no reconocido",
		},
		{
			name:   "error from repo",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					listVisitsFunc: func(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantErr:     true,
			errContains: "db error",
		},
		{
			name:   "success listing admin",
			roleID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					listVisitsFunc: func(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
						return []sqlcgen.ListVisitsRow{
							{
								VisitID:    1,
								PropertyID: 1,
								AgentID:    pgtype.Int4{Int32: 2, Valid: true},
								StatusName: "Pending",
								ClientName: "John",
								AgentName:  "Doe",
								Address:    "123 St",
							},
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "success listing agent",
			roleID: 2,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					listVisitsFunc: func(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
						return []sqlcgen.ListVisitsRow{{VisitID: 1, AgentID: pgtype.Int4{Int32: 10, Valid: true}}}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "success listing client",
			roleID: 3,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					listVisitsFunc: func(ctx context.Context, arg sqlcgen.ListVisitsParams) ([]sqlcgen.ListVisitsRow, error) {
						return []sqlcgen.ListVisitsRow{{VisitID: 1, AgentID: pgtype.Int4{Int32: 10, Valid: true}}}, nil
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

			now := time.Now()
			pID := int32(1)
			sID := int32(1)

			filter := ListVisitsFilter{
				Date:       &now,
				PropertyID: &pID,
				StatusID:   &sID,
			}

			res, err := svc.ListUserVisits(ctx, 10, tt.roleID, filter)

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
				if len(res) != 1 {
					t.Errorf("expected len %v, got %v", 1, len(res))
				}
			}
		})
	}
}
