package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/stretchr/testify/assert"
)

func TestService_HandleWebhook(t *testing.T) {
	ctx := context.Background()
	secret := "TEST-SECRET"

	tests := []struct {
		name          string
		mpAccessToken string
		requestID     string
		body          string
		setupRepo     func() *mockPaymentRepository
		setupSig      func(body string) string
		wantErr       bool
		errContains   string
	}{
		{
			name: "error when timestamp is too old (F3 Replay Protection)",
			setupSig: func(body string) string {
				oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
				return calculateFreshTestSig("req1", oldTs, body, secret)
			},
			body:        `{"type":"payment","data":{"id":"123"}}`,
			setupRepo: func() *mockPaymentRepository { return &mockPaymentRepository{} },
			wantErr:     true,
			errContains: "invalid signature or timestamp",
		},
		{
			name: "success when fresh timestamp and valid signature approved",
			mpAccessToken: "TEST-TOKEN",
			setupSig: func(body string) string {
				freshTs := strconv.FormatInt(time.Now().Unix(), 10)
				return calculateFreshTestSig("req1", freshTs, body, secret)
			},
			body: `{"type":"payment","data":{"id":"123"}}`,
			setupRepo: func() *mockPaymentRepository {
				return &mockPaymentRepository{
					getPaymentByGatewayIDFunc: func(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error) {
						return sqlcgen.GetPaymentByGatewayIDRow{PaymentID: 1, StatusID: PaymentStatusPending, ContractID: 1}, nil
					},
					beginFunc: func(ctx context.Context) (pgx.Tx, error) { return &mockTx{}, nil },
					updatePaymentStatusFunc: func(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error { return nil },
					getContractForPaymentFunc: func(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			token := tt.mpAccessToken
			if token == "" {
				token = "TOKEN"
			}
			svc := NewService(repo, token, secret)

			sig := tt.setupSig(tt.body)
			reqID := tt.requestID
			if reqID == "" {
				reqID = "req1"
			}

			err := svc.HandleWebhook(ctx, sig, reqID, []byte(tt.body))

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func calculateFreshTestSig(requestID, ts, body, secret string) string {
	manifest := fmt.Sprintf("id:%s;ts:%s;", requestID, ts)
	signedString := manifest + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedString))
	v1 := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("ts=%s,v1=%s", ts, v1)
}
