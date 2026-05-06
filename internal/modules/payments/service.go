package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

const (
	PaymentStatusPending   = 1
	PaymentStatusCompleted = 2
	PaymentStatusFailed    = 3

	ContractStatusCancelled = 3
	ContractStatusFinished  = 4

	roleAdminID  int32 = 1
	roleAgentID  int32 = 2
	roleClientID int32 = 3
)

type Service interface {
	// UC-16 & UC-17: Process and Confirm
	ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error

	// UC-17: List and Get
	ListPayments(ctx context.Context, userID int32, input ListPaymentsInput) (ListPaymentsResult, error)
	GetPaymentByID(ctx context.Context, userID int32, paymentID int32) (PaymentDetail, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
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

	if contract.StatusID == ContractStatusCancelled || contract.StatusID == ContractStatusFinished {
		return PaymentResponse{}, errors.New("operación denegada: el contrato ya está cancelado o finalizado")
	}

	if contract.ClientID != userID {
		return PaymentResponse{}, errors.New("operación no autorizada: este contrato no pertenece al usuario")
	}

	existingCompleted, err := txRepo.GetPaymentByContract(ctx, req.ContractID, PaymentStatusCompleted)
	if err == nil && len(existingCompleted) > 0 {
		p := existingCompleted[0]
		return PaymentResponse{
			PaymentUUID: p.PaymentUuid.Bytes,
			Status:      "Completed (Ya existía)",
			StatusID:    p.StatusID,
			Amount:      req.Amount,
			GatewayID:   p.GatewayPaymentID.String,
		}, nil
	}

	existingPending, _ := txRepo.GetPaymentByContract(ctx, req.ContractID, PaymentStatusPending)
	if len(existingPending) > 0 {
		p := existingPending[0]
		if time.Now().Before(p.DueDate.Time) {
			return PaymentResponse{}, fmt.Errorf("ya tienes un pago pendiente (OXXO) activo. Por favor págalo o espera a que expire (Expiración: %s)", p.DueDate.Time.Format("15:04"))
		}
	}

	contractAmount, _ := contract.AgreedAmount.Float64Value()
	if req.Amount != contractAmount.Float64 {
		return PaymentResponse{}, errors.New("el monto del pago no coincide con el monto pactado en el contrato")
	}

	statusID := int32(PaymentStatusCompleted)
	gatewayStatus := "succeeded"

	if strings.HasSuffix(req.CardNumber, "0000") {
		statusID = PaymentStatusFailed
		gatewayStatus = "declined"
	}

	if req.PaymentMethodID == 3 {
		statusID = PaymentStatusPending
		gatewayStatus = "pending_payment"
	}

	now := time.Now()
	billingPeriod := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	metadata := map[string]interface{}{
		"simulation_engine": "v1.2",
		"concurrency_safe":  true,
		"processed_at":      now.Format(time.RFC3339),
		"sandbox_mode":      true,
	}
	if req.CardNumber != "" {
		last4 := "xxxx"
		if len(req.CardNumber) >= 4 {
			last4 = req.CardNumber[len(req.CardNumber)-4:]
		}
		metadata["card_last_4"] = last4
	}
	metadataJSON, _ := json.Marshal(metadata)

	payment, err := txRepo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{
		ContractID:       req.ContractID,
		BillingPeriod:    pgtype.Date{Time: billingPeriod, Valid: true},
		DueDate:          pgtype.Date{Time: now.Add(24 * time.Hour), Valid: true},
		Amount:           pgtype.Numeric{Int: big.NewInt(int64(req.Amount * 100)), Exp: -2, Valid: true},
		PaymentMethodID:  req.PaymentMethodID,
		GatewayID:        pgtype.Int4{Int32: req.GatewayID, Valid: true},
		StatusID:         statusID,
		GatewayPaymentID: pgtype.Text{String: "MOCK-" + uuid.New().String()[:8], Valid: true},
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
		PaymentUUID: payment.PaymentUuid.Bytes,
		StatusID:    payment.StatusID,
		Amount:      req.Amount,
		GatewayID:   payment.GatewayPaymentID.String,
	}

	switch statusID {
	case PaymentStatusCompleted:
		res.Status = "Success"
		pDate := payment.PaymentDate.Time
		res.PaymentDate = &pDate
	case PaymentStatusFailed:
		res.Status = "Failed"
	case PaymentStatusPending:
		res.Status = "Pending"
		ref := "REF-" + strings.ToUpper(uuid.UUID(payment.PaymentUuid.Bytes).String()[:8])
		res.ReferenceNumber = &ref
	}

	return res, nil
}

func (s *service) ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error {
	payment, err := s.repo.GetPaymentByUUID(ctx, paymentUUID)
	if err != nil {
		return errors.New("pago no encontrado")
	}

	if payment.ClientID != userID {
		return errors.New("operación no autorizada: este pago no te pertenece")
	}

	if payment.StatusID != PaymentStatusPending {
		return errors.New("solo se pueden confirmar pagos que estén en estado pendiente")
	}

	if time.Now().After(payment.DueDate.Time) {
		return fmt.Errorf("la referencia de pago ha expirado (venció el %s)", payment.DueDate.Time.Format("2006-01-02 15:04"))
	}

	return s.repo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
		PaymentID:        payment.PaymentID,
		StatusID:         PaymentStatusCompleted,
		GatewayPaymentID: pgtype.Text{String: "CONFIRMED-" + uuid.New().String()[:8], Valid: true},
		GatewayStatus:    pgtype.Text{String: "confirmed", Valid: true},
		PaymentDate:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
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
	payment, err := s.repo.GetPaymentByID(ctx, paymentID)
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
		return payment, nil
	case roleAgentID:
		if payment.AgentID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	case roleClientID:
		if payment.ClientID != userID {
			return PaymentDetail{}, ErrPaymentForbidden
		}
	default:
		return PaymentDetail{}, ErrUnsupportedRole
	}

	return payment, nil
}

func isSupportedRole(roleID int32) bool {
	return roleID == roleAdminID || roleID == roleAgentID || roleID == roleClientID
}
