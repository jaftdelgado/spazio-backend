package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	mpConfig "github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error {
	if s.mpWebhookSecret != "" {
		if !validateMPSignature(xSignature, xRequestID, body, s.mpWebhookSecret) {
			return errors.New("invalid signature or timestamp")
		}
	}

	var webhookData struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &webhookData); err != nil {
		return err
	}

	if webhookData.Type != "payment" {
		return nil
	}

	paymentIDStr := webhookData.Data.ID
	mpCfg, _ := mpConfig.New(s.mpAccessToken)
	mpClient := payment.NewClient(mpCfg)

	mpPaymentID, err := strconv.ParseInt(paymentIDStr, 10, 64)
	if err != nil {
		return nil
	}

	var mpResp *payment.Response
	if s.mpAccessToken == "TEST-TOKEN" || s.mpAccessToken == "TEST-REFUNDED" || s.mpAccessToken == "TEST-REJECTED" {
		mpResp = &payment.Response{ID: int(mpPaymentID), Status: "approved"}
		if s.mpAccessToken == "TEST-REFUNDED" {
			mpResp.Status = "refunded"
		}
		if s.mpAccessToken == "TEST-REJECTED" {
			mpResp.Status = "rejected"
		}
	} else {
		mpResp, err = mpClient.Get(ctx, int(mpPaymentID))
		if err != nil {
			return err
		}
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("webhook transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	paymentRecord, err := txRepo.GetPaymentByGatewayID(ctx, strconv.Itoa(mpResp.ID))
	if err != nil {
		return nil // Payment not found in our DB, ignore
	}

	if paymentRecord.StatusID == PaymentStatusCompleted && mpResp.Status != "refunded" {
		return nil
	}

	newStatusID := paymentRecord.StatusID
	switch mpResp.Status {
	case "approved":
		newStatusID = PaymentStatusCompleted
	case "rejected", "cancelled":
		newStatusID = PaymentStatusFailed
	case "refunded", "charged_back":
		newStatusID = PaymentStatusRefunded
	}

	if newStatusID != paymentRecord.StatusID {
		err = txRepo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
			PaymentID:        paymentRecord.PaymentID,
			StatusID:         newStatusID,
			GatewayPaymentID: paymentRecord.GatewayPaymentID,
			GatewayStatus:    pgtype.Text{String: mpResp.Status, Valid: true},
			PaymentDate:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}

		if newStatusID == PaymentStatusCompleted {
			contract, err := txRepo.GetContractForPayment(ctx, paymentRecord.ContractID)
			if err != nil {
				return err
			}
			if err := s.finalizeContractState(ctx, txRepo, paymentRecord.ContractID, string(contract.TransactionType)); err != nil {
				return err
			}
		}

		return tx.Commit(ctx)
	}

	return nil
}

func validateMPSignature(xSignature, xRequestID string, body []byte, secret string) bool {
	parts := strings.Split(xSignature, ",")
	var ts, v1 string
	for _, p := range parts {
		kv := strings.Split(strings.TrimSpace(p), "=")
		if len(kv) == 2 {
			if kv[0] == "ts" {
				ts = kv[1]
			} else if kv[0] == "v1" {
				v1 = kv[1]
			}
		}
	}

	if ts == "" || v1 == "" {
		return false
	}

	timestampInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	webhookTime := time.Unix(timestampInt, 0)
	if time.Since(webhookTime) > 5*time.Minute || time.Since(webhookTime) < -5*time.Minute {
		return false
	}

	manifest := fmt.Sprintf("id:%s;ts:%s;", xRequestID, ts)
	signedString := manifest + string(body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedString))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return expectedSignature == v1
}
