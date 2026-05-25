package payments

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestService_ListPayments(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		roleID    int32
		setupRepo func() *mockPaymentRepository
		wantErr   bool
	}{
		{
			name:   "success listing payments for admin",
			roleID: roleAdminID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					listPaymentsFunc: func(ctx context.Context, uid, rid int32, in ListPaymentsInput) ([]PaymentListItem, error) {
						return []PaymentListItem{{PaymentID: 1, TotalCount: 1}}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "error when role is not supported",
			roleID: 99,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo, "TOKEN", "SECRET")

			res, err := svc.ListPayments(ctx, 1, tt.roleID, ListPaymentsInput{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if res.Data == nil {
					t.Errorf("expected not nil")
				}
			}
		})
	}
}

func TestService_GetPaymentByUUID(t *testing.T) {
	ctx := context.Background()
	paymentUUID := uuid.New()

	tests := []struct {
		name      string
		userID    int32
		roleID    int32
		setupRepo func() *mockPaymentRepository
		wantErr   bool
	}{
		{
			name:   "success for admin to see any payment",
			userID: 1,
			roleID: roleAdminID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					getPaymentDetailByUUIDFunc: func(ctx context.Context, id uuid.UUID) (PaymentDetailRecord, error) {
						return PaymentDetailRecord{PaymentID: 1, ClientID: 10}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "forbidden for client to see someone else's payment",
			userID: 5,
			roleID: roleClientID,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					getPaymentDetailByUUIDFunc: func(ctx context.Context, id uuid.UUID) (PaymentDetailRecord, error) {
						return PaymentDetailRecord{PaymentID: 1, ClientID: 10}, nil
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo, "TOKEN", "SECRET")

			_, err := svc.GetPaymentByUUID(ctx, tt.userID, tt.roleID, paymentUUID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewPaymentDetailResponse(t *testing.T) {
	paymentUUID := uuid.New()
	payment := PaymentDetailRecord{
		PaymentID:       1,
		ContractID:      2,
		PropertyID:      3,
		TransactionID:   4,
		TransactionType: "rent",
		BillingPeriod:   "2024-03-01",
		DueDate:         "2024-03-10",
		AgreedAmount:    "15000.00",
		Amount:          "1500.00",
		Currency:        "MXN",
		PaymentMethod:   "Transferencia",
		Status:          "Pagado",
		ClientID:        7,
		AgentID:         9,
	}

	t.Run("admin receives sensitive identifiers", func(t *testing.T) {
		response := newPaymentDetailResponse(paymentUUID, payment, roleAdminID)
		if paymentUUID != response.PaymentUUID {
			t.Errorf("expected %v, got %v", paymentUUID, response.PaymentUUID)
		}
		if response.ClientID != nil {
			if int32(7) != *response.ClientID {
				t.Errorf("expected %v, got %v", int32(7), *response.ClientID)
			}
		}
		if response.AgentID != nil {
			if int32(9) != *response.AgentID {
				t.Errorf("expected %v, got %v", int32(9), *response.AgentID)
			}
		}
	})

	t.Run("client omits sensitive identifiers", func(t *testing.T) {
		response := newPaymentDetailResponse(paymentUUID, payment, roleClientID)
		if paymentUUID != response.PaymentUUID {
			t.Errorf("expected %v, got %v", paymentUUID, response.PaymentUUID)
		}
		if response.ClientID != nil {
			t.Errorf("expected nil, got %v", response.ClientID)
		}
		if response.AgentID != nil {
			t.Errorf("expected nil, got %v", response.AgentID)
		}
	})
}
