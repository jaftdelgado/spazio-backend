//go:build integration

package payments

import (
	"context"
	"testing"

	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_ProcessPayment(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	clientID := int32(203)
	contractID := int32(1000)

	tests := []struct {
		name       string
		req        RegisterPaymentRequest
		wantStatus string
		wantErr    bool
	}{
		{
			name: "Failure_Incorrect_Amount",
			req: RegisterPaymentRequest{
				ContractID:      contractID,
				PaymentMethodID: 1,
				GatewayID:       1,
				Amount:          400000, // Wrong amount
				Currency:        "MXN",
				PayerEmail:      "client@spazio.com",
				GatewayMethodID: "visa",
				Token:           "TEST-TOKEN",
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "Failure_Invalid_Currency",
			req: RegisterPaymentRequest{
				ContractID:      contractID,
				PaymentMethodID: 1,
				GatewayID:       1,
				Amount:          500000,
				Currency:        "USD", // Wrong currency
				PayerEmail:      "client@spazio.com",
				GatewayMethodID: "visa",
				Token:           "TEST-TOKEN",
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "Failure_Invalid_Contract",
			req: RegisterPaymentRequest{
				ContractID:      9999, // Non-existent contract
				PaymentMethodID: 1,
				GatewayID:       1,
				Amount:          500000,
				Currency:        "MXN",
				PayerEmail:      "client@spazio.com",
				GatewayMethodID: "visa",
				Token:           "TEST-TOKEN",
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "Success_Process_First_Payment",
			req: RegisterPaymentRequest{
				ContractID:      contractID,
				PaymentMethodID: 1,
				GatewayID:       1,
				Amount:          500000,
				Currency:        "MXN",
				PayerEmail:      "client@spazio.com",
				GatewayMethodID: "visa",
				Token:           "TEST-TOKEN",
			},
			wantStatus: "Success",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Clean up any payments left over from previous tests to ensure isolation
			_, _ = pool.Exec(ctx, "DELETE FROM payments WHERE contract_id = $1", contractID)

			txSvc := &service{
				repo:          repo,
				mpAccessToken: "TEST-TOKEN",
				mpClient: &mockMPClient{
					createPaymentFunc: func(ctx context.Context, req payment.Request) (*payment.Response, error) {
						return &payment.Response{
							ID:           123456789,
							Status:       "approved",
							StatusDetail: "accredited",
						}, nil
					},
				},
			}

			res, err := txSvc.ProcessPayment(ctx, clientID, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				var count int
				_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE contract_id = $1", tt.req.ContractID).Scan(&count)
				if count != 0 {
					t.Errorf("Expected no payment inserted on failure, found %d", count)
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
			err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM payments WHERE contract_id = $1", tt.req.ContractID).Scan(&count)
			if err != nil {
				t.Fatalf("Failed to query payments: %v", err)
			}
			if count != 1 {
				t.Errorf("Expected 1 payment in DB, got %d", count)
			}

			var propStatus int
			err = pool.QueryRow(ctx, "SELECT status_id FROM properties WHERE property_id = 500").Scan(&propStatus)
			if err != nil {
				t.Fatalf("Failed to query property status: %v", err)
			}
			if propStatus != 4 {
				t.Errorf("Expected property status 4 (Rented), got %d", propStatus)
			}
		})
	}
}
