package payments

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestService_ProcessPayment(t *testing.T) {
	ctx := context.Background()
	clientID := int32(10)

	baseReq := RegisterPaymentRequest{
		ContractID:      1,
		PaymentMethodID: 1,
		Amount:          1000.0,
		Currency:        "MXN",
		PayerEmail:      "test@test.com",
		GatewayMethodID: "visa",
		GatewayID:       1,
	}

	tests := []struct {
		name          string
		mpAccessToken string
		req           RegisterPaymentRequest
		setupRepo     func() *mockPaymentRepository
		wantErr       bool
		errContains   string
		wantStatus    string
	}{
		{
			name: "error when transaction begin fails",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) {
						return nil, errors.New("begin fail")
					},
				}
			},
			wantErr:     true,
			errContains: "fallo al iniciar transacción",
		},
		{
			name: "error when contract not found",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { 
						return &mockTx{}, nil 
					},
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{}, errors.New("not found")
					},
				}
			},
			wantErr:     true,
			errContains: "no se pudo encontrar la información del contrato",
		},
		{
			name: "error when property is not available for first payment",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { 
						return &mockTx{}, nil 
					},
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{PropertyStatusID: 1}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) {
						return 0, nil
					},
				}
			},
			wantErr:     true,
			errContains: "la propiedad ya no está disponible",
		},
		{
			name: "error when currency mismatch",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { 
						return &mockTx{}, nil 
					},
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{
							ClientID: clientID, 
							Currency: "USD",
						}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) {
						return 1, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no coincide con la moneda del contrato",
		},
		{
			name: "error when amount mismatch (F7 Protection)",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { 
						return &mockTx{}, nil 
					},
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{
							ClientID: clientID, 
							Currency: "MXN",
							AgreedAmount: pgtype.Numeric{Int: big.NewInt(500000), Exp: -2, Valid: true},
						}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) {
						return 1, nil
					},
				}
			},
			wantErr:     true,
			errContains: "no coincide con el monto pactado",
		},
		{
			name:          "success and finalize first payment (Transactional F2)",
			mpAccessToken: "TEST-TOKEN",
			setupRepo: func() *mockPaymentRepository {
				finalized := false
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { 
						return &mockTx{}, nil 
					},
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{
							ClientID:         clientID,
							Currency:         "MXN",
							PropertyStatusID: 2,
							TransactionType:  "rent",
							AgreedAmount:     pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
						}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) {
						if finalized {
							return 1, nil
						}
						return 0, nil
					},
					getPaymentByContractFunc: func(ctx context.Context, cid, sid int32) ([]sqlcgen.Payment, error) {
						return nil, nil
					},
					getPendingPaymentsFunc: func(ctx context.Context, cid int32) ([]sqlcgen.GetPendingPaymentsRow, error) {
						return []sqlcgen.GetPendingPaymentsRow{}, nil
					},
					createPaymentFunc: func(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error) {
						finalized = true
						return sqlcgen.Payment{StatusID: PaymentStatusCompleted}, nil
					},
					updateTransactionStatusByContractFunc: func(ctx context.Context, cid, sid int32) error { return nil },
					updatePropertyStatusByContractFunc:    func(ctx context.Context, cid, sid int32) error { return nil },
					updateContractStatusFunc:              func(ctx context.Context, cid, sid int32) error { return nil },
				}
			},
			wantErr:    false,
			wantStatus: "Success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			token := tt.mpAccessToken
			if token == "" {
				token = "TOKEN"
			}
			svc := NewService(repo, token, "SECRET")

			req := tt.req
			if req.ContractID == 0 {
				req = baseReq
			}

			res, err := svc.ProcessPayment(ctx, clientID, req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errContains))
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, res.Status)
			}
		})
	}
}
