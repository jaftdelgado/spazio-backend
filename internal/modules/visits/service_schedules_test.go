package visits

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestService_GetAvailableSlots(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Monday

	tests := []struct {
		name        string
		propertyID  int32
		setupRepo   func() *mockVisitsRepository
		wantSlots   int
		wantErr     bool
		errContains string
	}{
		{
			name:       "error when agent not found",
			propertyID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) {
						return 0, errors.New("agent not found")
					},
				}
			},
			wantErr:     true,
			errContains: "agent not found",
		},
		{
			name:       "error when agent schedule fails",
			propertyID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) {
						return 2, nil
					},
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return nil, errors.New("schedule error")
					},
				}
			},
			wantErr:     true,
			errContains: "schedule error",
		},
		{
			name:       "empty slots when no schedule for the day",
			propertyID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						return []sqlcgen.GetAgentScheduleRow{
							{DayOfWeek: 2}, // Tuesday, but test asks for Monday (1)
						}, nil
					},
				}
			},
			wantSlots: 0,
			wantErr:   false,
		},
		{
			name:       "success available slots with exception and occupied",
			propertyID: 1,
			setupRepo: func() *mockVisitsRepository {
				return &mockVisitsRepository{
					getPrimaryAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 2, nil },
					getAgentScheduleFunc: func(ctx context.Context, agentID int32) ([]sqlcgen.GetAgentScheduleRow, error) {
						// 09:00 to 12:00 (3 slots)
						return []sqlcgen.GetAgentScheduleRow{
							{
								DayOfWeek: 1, // Monday
								StartTime: pgtype.Time{Microseconds: 9 * 3600 * 1e6, Valid: true},
								EndTime:   pgtype.Time{Microseconds: 12 * 3600 * 1e6, Valid: true},
							},
						}, nil
					},
					getPropertyExceptionsFunc: func(ctx context.Context, propertyID int32, start, end time.Time) ([]sqlcgen.GetPropertyExceptionsRow, error) {
						// Exception from 10:00 to 11:00
						return []sqlcgen.GetPropertyExceptionsRow{
							{
								StartTime: pgtype.Time{Microseconds: 10 * 3600 * 1e6, Valid: true},
								EndTime:   pgtype.Time{Microseconds: 11 * 3600 * 1e6, Valid: true},
							},
						}, nil
					},
					getOccupiedVisitsFunc: func(ctx context.Context, agentID int32, start, end time.Time) ([]time.Time, error) {
						// Occupied at 11:00
						return []time.Time{
							time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
						}, nil
					},
				}
			},
			wantSlots: 2, // 09-10 (Avail), 10-11 (Exception -> Skipped), 11-12 (Not Avail)
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo)

			slots, err := svc.GetAvailableSlots(ctx, tt.propertyID, date)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, slots, tt.wantSlots)

				// For the success case, let's also verify availability manually
				if tt.wantSlots == 2 {
					assert.True(t, slots[0].Available)  // 09:00
					assert.False(t, slots[1].Available) // 11:00 (Occupied)
				}
			}
		})
	}
}
