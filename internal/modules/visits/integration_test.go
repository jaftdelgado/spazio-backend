//go:build integration

package visits

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_ScheduleVisit(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	clientID := int32(203)
	propertyID := int32(501) // Use property 501 (Sale, Available) to avoid conflicts with Payments test changing 500 to Rented.

	now := time.Now()
	daysUntilMonday := int((time.Monday - now.Weekday() + 7) % 7)
	if daysUntilMonday <= 2 {
		daysUntilMonday += 7
	}
	loc, _ := time.LoadLocation("America/Mexico_City")
	validDate := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 10, 0, 0, 0, loc)
	invalidDatePast := time.Date(now.Year(), now.Month(), now.Day()-1, 10, 0, 0, 0, loc)

	tests := []struct {
		name       string
		propertyID int32
		visitDate  time.Time
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "Failure_Invalid_Date_Past",
			propertyID: propertyID,
			visitDate:  invalidDatePast,
			wantStatus: "",
			wantErr:    true,
		},
		{
			name:       "Failure_Invalid_Property",
			propertyID: 9999,
			visitDate:  validDate,
			wantStatus: "",
			wantErr:    true,
		},
		{
			name:       "Success_Schedule_Visit",
			propertyID: propertyID,
			visitDate:  validDate,
			wantStatus: "Pending",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shared.WithTransaction(t, pool, func(tx pgx.Tx) {
				txRepo := repo.WithTx(tx)
				txSvc := NewService(txRepo)

				res, err := txSvc.ScheduleVisit(ctx, clientID, tt.propertyID, tt.visitDate)

				if tt.wantErr {
					if err == nil {
						t.Errorf("Expected an error but got nil")
					}
					var count int
					_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM visits WHERE property_id = $1 AND client_id = $2", tt.propertyID, clientID).Scan(&count)
					if count != 0 {
						t.Errorf("Expected 0 visits inserted on failure, found %d", count)
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if res.Status != tt.wantStatus {
					t.Errorf("Expected status %s, got %s", tt.wantStatus, res.Status)
				}

				var count int
				err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM visits WHERE property_id = $1 AND client_id = $2", tt.propertyID, clientID).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query visits: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 visit in DB, got %d", count)
				}
			})
		})
	}
}
