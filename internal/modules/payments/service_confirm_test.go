package payments

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestService_ConfirmPendingPayment(t *testing.T) {
	ctx := context.Background()
	clientID := int32(10)
	pUUID := uuid.New()

	tests := []struct {
		name        string
		setupRepo   func() *mockPaymentRepository
		wantErr     bool
		errContains string
	}{
		{
			name: "success manual confirmation with atomic state update",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getPaymentByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error) {
						return sqlcgen.GetPaymentByUUIDRow{
							ClientID: clientID, StatusID: PaymentStatusPending, ContractID: 1,
							DueDate: pgtype.Date{Time: time.Now().Add(24 * time.Hour), Valid: true},
						}, nil
					},
					updatePaymentStatusFunc: func(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error { return nil },
					getContractForPaymentFunc: func(ctx context.Context, cid int32) (sqlcgen.GetContractForPaymentRow, error) {
						return sqlcgen.GetContractForPaymentRow{TransactionType: "rent"}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, cid int32) (int64, error) { return 1, nil },
					updateTransactionStatusByContractFunc: func(ctx context.Context, cid, sid int32) error { return nil },
					updatePropertyStatusByContractFunc:    func(ctx context.Context, cid, sid int32) error { return nil },
					updateContractStatusFunc:              func(ctx context.Context, cid, sid int32) error { return nil },
				}
			},
			wantErr: false,
		},
		{
			name: "error when payment has expired",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getPaymentByUUIDFunc: func(ctx context.Context, id uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error) {
						return sqlcgen.GetPaymentByUUIDRow{
							ClientID: clientID, StatusID: PaymentStatusPending,
							DueDate: pgtype.Date{Time: time.Now().Add(-24 * time.Hour), Valid: true},
						}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "la referencia de pago ha expirado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			svc := NewService(repo, "TOKEN", "SECRET")

			err := svc.ConfirmPendingPayment(ctx, clientID, pUUID)

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
