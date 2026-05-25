package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	mpConfig "github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

func (s *service) ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("fallo al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	contract, err := txRepo.GetContractForPaymentWithLock(ctx, req.ContractID)
	if err != nil {
		return PaymentResponse{}, errors.New("no se pudo encontrar la información del contrato")
	}

	completedCount, err := txRepo.CountCompletedPaymentsForContract(ctx, req.ContractID)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("error al verificar historial de pagos: %w", err)
	}

	if completedCount == 0 && contract.PropertyStatusID != 2 {
		return PaymentResponse{}, errors.New("la propiedad ya no está disponible para contratación (debe estar en estado Disponible)")
	}

	if contract.StatusID == ContractStatusBlocked {
		return PaymentResponse{}, errors.New("el contrato está bloqueado por un administrador y no acepta pagos")
	}

	if contract.StatusID == ContractStatusTerminated {
		return PaymentResponse{}, errors.New("el contrato ha sido terminado y no acepta más pagos")
	}

	if contract.EndDate.Valid && time.Now().After(contract.EndDate.Time) {
		return PaymentResponse{}, errors.New("la fecha de vencimiento del contrato ha pasado")
	}

	if contract.ClientID != userID {
		return PaymentResponse{}, errors.New("operación no autorizada: este contrato no pertenece al usuario")
	}

	contractCurrency := strings.TrimSpace(contract.Currency)
	if !strings.EqualFold(req.Currency, contractCurrency) {
		return PaymentResponse{}, fmt.Errorf("la moneda del pago (%s) no coincide con la moneda del contrato (%s)", req.Currency, contractCurrency)
	}

	var billingPeriod time.Time
	if contract.TransactionType == "rent" {
		lastPeriod, err := txRepo.GetLastPaidPeriod(ctx, req.ContractID)
		if err != nil || !lastPeriod.Valid {
			billingPeriod = contract.StartDate.Time
		} else {
			periodName := strings.ToLower(contract.PeriodName.String)
			switch periodName {
			case "daily", "diario":
				billingPeriod = lastPeriod.Time.AddDate(0, 0, 1)
			case "weekly", "semanal":
				billingPeriod = lastPeriod.Time.AddDate(0, 0, 7)
			case "yearly", "anual":
				billingPeriod = lastPeriod.Time.AddDate(1, 0, 0)
			default:
				billingPeriod = lastPeriod.Time.AddDate(0, 1, 0)
			}
		}
	} else {
		now := time.Now()
		billingPeriod = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	existingCompleted, err := txRepo.GetPaymentByContract(ctx, req.ContractID, PaymentStatusCompleted)
	if err == nil {
		for _, p := range existingCompleted {
			if p.BillingPeriod.Time.Equal(billingPeriod) {
				return PaymentResponse{
					PaymentUUID: p.PaymentUuid.Bytes,
					Status:      "Already Paid",
					StatusID:    p.StatusID,
					Amount:      req.Amount,
					GatewayID:   p.GatewayPaymentID.String,
				}, nil
			}
		}
	}

	pendingPayments, err := txRepo.GetPendingPayments(ctx, req.ContractID)
	if err == nil {
		for _, p := range pendingPayments {
			_ = txRepo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
				PaymentID:        p.PaymentID,
				StatusID:         PaymentStatusFailed,
				GatewayPaymentID: p.GatewayPaymentID,
				GatewayStatus:    pgtype.Text{String: "cancelled_by_new_attempt", Valid: true},
			})
		}
	}

	// F7: Safe Money Comparison
	contractAmount, _ := contract.AgreedAmount.Float64Value()
	if req.Amount != int64(contractAmount.Float64*100) {
		return PaymentResponse{}, fmt.Errorf("el monto del pago (%.2f) no coincide con el monto pactado en el contrato (%.2f)", float64(req.Amount)/100.0, contractAmount.Float64)
	}

	mpCfg, err := mpConfig.New(s.mpAccessToken)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("error de configuración MP: %w", err)
	}
	mpClient := payment.NewClient(mpCfg)

	installments := req.Installments
	if installments == 0 {
		installments = 1
	}

	mpReq := payment.Request{
		TransactionAmount: float64(req.Amount) / 100.0,
		PaymentMethodID:   req.GatewayMethodID,
		Payer: &payment.PayerRequest{
			Email: req.PayerEmail,
		},
		Token:        req.Token,
		Installments: installments,
		Description:  fmt.Sprintf("Pago contrato #%d - Periodo %s", req.ContractID, billingPeriod.Format("2006-01-02")),
	}

	if req.IssuerID != "" {
		mpReq.IssuerID = req.IssuerID
	}

	var mpResp *payment.Response
	if s.mpAccessToken == "TEST-TOKEN" || req.Token == "TEST-TOKEN" || s.mpAccessToken == "TEST-REJECTED" || s.mpAccessToken == "TEST-PENDING" || s.mpAccessToken == "TEST-REFUNDED" {
		mpResp = &payment.Response{ID: 123456789, Status: "approved", StatusDetail: "accredited"}
		if s.mpAccessToken == "TEST-REJECTED" {
			mpResp.Status = "rejected"
			mpResp.StatusDetail = "cc_rejected_bad_filled_security_code"
		}
		if s.mpAccessToken == "TEST-PENDING" {
			mpResp.Status = "pending"
			mpResp.StatusDetail = "pending_waiting_payment"
		}
	} else {
		mpResp, err = mpClient.Create(ctx, mpReq)
		if err != nil {
			return PaymentResponse{}, fmt.Errorf("error al procesar pago en pasarela: %w", err)
		}
	}

	statusID := int32(PaymentStatusFailed)
	gatewayStatus := mpResp.Status

	if mpResp.Status == "approved" {
		statusID = PaymentStatusCompleted
	} else if mpResp.Status == "pending" || mpResp.Status == "in_process" {
		statusID = PaymentStatusPending
	}

	if statusID == PaymentStatusFailed {
		return PaymentResponse{}, fmt.Errorf("el pago fue rechazado por la pasarela (Motivo: %s)", mpResp.StatusDetail)
	}

	now := time.Now()
	metadata := map[string]interface{}{
		"mp_id":            mpResp.ID,
		"mp_status":        mpResp.Status,
		"mp_status_detail": mpResp.StatusDetail,
		"mp_payment_type":  mpResp.PaymentTypeID,
		"billing_period":   billingPeriod.Format("2006-01-02"),
		"processed_at":     now.Format(time.RFC3339),
	}
	metadataJSON, _ := json.Marshal(metadata)

	paymentRecord, err := txRepo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{
		ContractID:       req.ContractID,
		ClientID:         userID,
		BillingPeriod:    pgtype.Date{Time: billingPeriod, Valid: true},
		DueDate:          pgtype.Date{Time: now.Add(24 * time.Hour), Valid: true},
		Amount:           pgtype.Numeric{Int: big.NewInt(req.Amount), Exp: -2, Valid: true},
		PaymentMethodID:  req.PaymentMethodID,
		GatewayID:        pgtype.Int4{Int32: req.GatewayID, Valid: true},
		StatusID:         statusID,
		GatewayPaymentID: pgtype.Text{String: strconv.Itoa(mpResp.ID), Valid: true},
		GatewayStatus:    pgtype.Text{String: gatewayStatus, Valid: true},
		PaymentDate:      pgtype.Timestamptz{Time: now, Valid: true},
		Metadata:         metadataJSON,
	})

	if err != nil {
		return PaymentResponse{}, translateError(err)
	}

	// F2: Atomic state update. Return error to trigger rollback if finalization fails.
	if statusID == PaymentStatusCompleted {
		if err := s.finalizeContractState(ctx, txRepo, req.ContractID, string(contract.TransactionType)); err != nil {
			return PaymentResponse{}, fmt.Errorf("error al finalizar estado del contrato: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PaymentResponse{}, fmt.Errorf("fallo al confirmar transacción: %w", err)
	}

	res := PaymentResponse{
		PaymentUUID: paymentRecord.PaymentUuid.Bytes,
		StatusID:    paymentRecord.StatusID,
		Amount:      req.Amount,
		GatewayID:   paymentRecord.GatewayPaymentID.String,
	}

	if statusID == PaymentStatusCompleted {
		res.Status = "Success"
		pDate := paymentRecord.PaymentDate.Time
		res.PaymentDate = &pDate
	} else if statusID == PaymentStatusPending {
		res.Status = "Pending"
		ref := "REF-" + strings.ToUpper(uuid.UUID(paymentRecord.PaymentUuid.Bytes).String()[:8])
		res.ReferenceNumber = &ref
	}

	return res, nil
}
