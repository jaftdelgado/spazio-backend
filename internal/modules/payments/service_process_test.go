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
	"github.com/mercadopago/sdk-go/pkg/payment"
)

func TestService_ProcessPayment(t *testing.T) {
	ctx := context.Background()
	clientID := int32(10)

	baseReq := RegisterPaymentRequest{
		ContractID:      1,
		PaymentMethodID: 1,
		Amount:          100000,
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
		setupMPClient func() *mockMPClient
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
						return sqlcgen.GetContractForPaymentWithLockRow{PropertyStatusID: 4}, nil
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
							ClientID:     clientID,
							Currency:     "MXN",
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
			name: "error when contract is blocked",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{StatusID: ContractStatusBlocked}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) { return 1, nil },
				}
			},
			wantErr:     true,
			errContains: "bloqueado por un administrador",
		},
		{
			name: "error when unauthorized client",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{ClientID: 999, StatusID: ContractStatusActive}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) { return 1, nil },
				}
			},
			wantErr:     true,
			errContains: "operación no autorizada",
		},
		{
			name: "error when gateway rejects payment",
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					getContractForPaymentWithLockFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error) {
						return sqlcgen.GetContractForPaymentWithLockRow{
							ClientID: clientID, Currency: "MXN", StatusID: ContractStatusActive,
							AgreedAmount: pgtype.Numeric{Int: big.NewInt(100000), Exp: -2, Valid: true},
						}, nil
					},
					countCompletedPaymentsForContractFunc: func(ctx context.Context, contractID int32) (int64, error) { return 1, nil },
					getPaymentByContractFunc: func(ctx context.Context, cid, sid int32) ([]sqlcgen.Payment, error) { return nil, nil },
					getPendingPaymentsFunc: func(ctx context.Context, cid int32) ([]sqlcgen.GetPendingPaymentsRow, error) { return nil, nil },
				}
			},
			setupMPClient: func() *mockMPClient {
				return &mockMPClient{
					createPaymentFunc: func(ctx context.Context, req payment.Request) (*payment.Response, error) {
						return &payment.Response{
							ID:           123456789,
							Status:       "rejected",
							StatusDetail: "cc_rejected_bad_filled_security_code",
						}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "fue rechazado por la pasarela",
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
			var mpClient *mockMPClient
			if tt.setupMPClient != nil {
				mpClient = tt.setupMPClient()
			} else {
				mpClient = &mockMPClient{
					createPaymentFunc: func(ctx context.Context, req payment.Request) (*payment.Response, error) {
						return &payment.Response{
							ID:           123456789,
							Status:       "approved",
							StatusDetail: "accredited",
						}, nil
					},
				}
			}
			svc := NewTestService(repo, mpClient, "TOKEN", "SECRET")

			req := tt.req
			if req.ContractID == 0 {
				req = baseReq
			}

			res, err := svc.ProcessPayment(ctx, clientID, req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errContains != "" {
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
						t.Errorf("expected %v to contain %v", strings.ToLower(err.Error()), strings.ToLower(tt.errContains))
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantStatus != res.Status {
					t.Errorf("expected %v, got %v", tt.wantStatus, res.Status)
				}
			}
		})
	}
}
