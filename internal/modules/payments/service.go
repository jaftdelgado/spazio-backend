package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

const (
	PaymentStatusPending   = 1
	PaymentStatusCompleted = 2
	PaymentStatusFailed    = 3
	PaymentStatusRefunded  = 4

	ContractStatusExpired    = 3
	ContractStatusTerminated = 4
	ContractStatusBlocked    = 5

	roleAdminID  int32 = 1
	roleAgentID  int32 = 2
	roleClientID int32 = 3
)

type Service interface {
	ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error
	HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error

	ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error)
	GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error)
}

type service struct {
	repo            Repository
	mpAccessToken   string
	mpWebhookSecret string
}

func NewService(repo Repository, mpAccessToken string, mpWebhookSecret string) Service {
	return &service{
		repo:            repo,
		mpAccessToken:   mpAccessToken,
		mpWebhookSecret: mpWebhookSecret,
	}
}

func translateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return errors.New("el contrato o método de pago seleccionado no existe")
		}
	}
	return err
}

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
			now := time.Now()
			billingPeriod = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		} else {
			billingPeriod = lastPeriod.Time.AddDate(0, 1, 0)
		}
	} else {
		now := time.Now()
		billingPeriod = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
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

	pendingPayments, _ := txRepo.GetPendingPayments(ctx, req.ContractID)
	for _, p := range pendingPayments {
		_ = txRepo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
			PaymentID:        p.PaymentID,
			StatusID:         PaymentStatusFailed,
			GatewayPaymentID: p.GatewayPaymentID,
			GatewayStatus:    pgtype.Text{String: "cancelled_by_new_attempt", Valid: true},
		})
	}

	contractAmount, _ := contract.AgreedAmount.Float64Value()
	if req.Amount != contractAmount.Float64 {
		return PaymentResponse{}, fmt.Errorf("el monto del pago (%f) no coincide con el monto pactado en el contrato (%f)", req.Amount, contractAmount.Float64)
	}

	mpCfg, err := config.New(s.mpAccessToken)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("error de configuración MP: %w", err)
	}
	mpClient := payment.NewClient(mpCfg)

	installments := req.Installments
	if installments == 0 {
		installments = 1
	}

	mpReq := payment.Request{
		TransactionAmount: req.Amount,
		PaymentMethodID:   req.GatewayMethodID,
		Payer: &payment.PayerRequest{
			Email: req.PayerEmail,
		},
		Token:        req.Token,
		Installments: installments,
		Description:  fmt.Sprintf("Pago contrato #%d - Periodo %s", req.ContractID, billingPeriod.Format("2006-01")),
	}

	if req.IssuerID != "" {
		mpReq.IssuerID = req.IssuerID
	}

	var mpResp *payment.Response
	if s.mpAccessToken == "TEST-TOKEN" || s.mpAccessToken == "TEST-REJECTED" || s.mpAccessToken == "TEST-PENDING" || s.mpAccessToken == "TEST-REFUNDED" {
		mpResp = &payment.Response{ID: 123, Status: "approved"}
		if s.mpAccessToken == "TEST-REJECTED" {
			mpResp.Status = "rejected"
		}
		if s.mpAccessToken == "TEST-PENDING" {
			mpResp.Status = "pending"
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
		Amount:           pgtype.Numeric{Int: big.NewInt(int64(req.Amount * 100)), Exp: -2, Valid: true},
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

func (s *service) ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error {
	paymentRecord, err := s.repo.GetPaymentByUUID(ctx, paymentUUID)
	if err != nil {
		return errors.New("pago no encontrado")
	}

	if paymentRecord.ClientID != userID {
		return errors.New("operación no autorizada: este pago no te pertenece")
	}

	if paymentRecord.StatusID != PaymentStatusPending {
		return errors.New("solo se pueden confirmar pagos que estén en estado pendiente")
	}

	nowDate := time.Now().Truncate(24 * time.Hour)
	if nowDate.After(paymentRecord.DueDate.Time) {
		return fmt.Errorf("la referencia de pago ha expirado (venció el %s)", paymentRecord.DueDate.Time.Format("2006-01-02"))
	}

	return s.repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
		PaymentID:        paymentRecord.PaymentID,
		StatusID:         PaymentStatusCompleted,
		GatewayPaymentID: paymentRecord.GatewayPaymentID,
		GatewayStatus:    pgtype.Text{String: "confirmed", Valid: true},
		PaymentDate:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
}

func (s *service) HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error {
	if s.mpWebhookSecret != "" {
		if !validateMPSignature(xSignature, xRequestID, body, s.mpWebhookSecret) {
			return errors.New("invalid signature")
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
	mpCfg, _ := config.New(s.mpAccessToken)
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

	paymentRecord, err := s.repo.GetPaymentByGatewayID(ctx, strconv.Itoa(mpResp.ID))
	if err != nil {
		return nil
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
		err = s.repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
			PaymentID:        paymentRecord.PaymentID,
			StatusID:         newStatusID,
			GatewayPaymentID: paymentRecord.GatewayPaymentID,
			GatewayStatus:    pgtype.Text{String: mpResp.Status, Valid: true},
			PaymentDate:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		return err
	}

	return nil
}

func validateMPSignature(xSignature, xRequestID string, body []byte, secret string) bool {
	parts := strings.Split(xSignature, ",")
	var ts, v1 string
	for _, p := range parts {
		kv := strings.Split(p, "=")
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

	manifest := fmt.Sprintf("id:%s;ts:%s;", xRequestID, ts)
	signedString := manifest + string(body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedString))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return expectedSignature == v1
}

func (s *service) ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error) {
	roleID, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return ListPaymentsResult{}, fmt.Errorf("list payments: %w", err)
	}

	if !isSupportedRole(roleID) {
		return ListPaymentsResult{}, ErrUnsupportedRole
	}

	items, err := s.repo.ListPayments(ctx, userID, roleID, input)
	if err != nil {
		return ListPaymentsResult{}, fmt.Errorf("list payments: %w", err)
	}

	var total int64
	if len(items) > 0 {
		total = items[0].TotalCount
	}

	return ListPaymentsResult{
		Data: items,
		Pagination: PaymentsPagination{
			Limit:  input.Limit,
			Offset: input.Offset,
			Total:  total,
		},
	}, nil
}

func (s *service) GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error) {
	paymentRecord, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return PaymentDetail{}, ErrPaymentNotFound
		}
		return PaymentDetail{}, fmt.Errorf("get payment by id: %w", err)
	}

	roleID, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("get payment by id: %w", err)
	}

	switch roleID {
	case roleAdminID:
		return paymentRecord, nil
	case roleAgentID:
		if paymentRecord.AgentID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	case roleClientID:
		if paymentRecord.ClientID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	default:
		return PaymentDetail{}, ErrUnsupportedRole
	}

	return paymentRecord, nil
}

func isSupportedRole(roleID int32) bool {
	return roleID == roleAdminID || roleID == roleAgentID || roleID == roleClientID
}
