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

// Payment status constants matching the database catalog
const (
	PaymentStatusPending   = 1
	PaymentStatusCompleted = 2
	PaymentStatusFailed    = 3

	// Contract status constants (Assuming 1=Draft/Pending, 2=Active, 3=Cancelled, 4=Finished)
	ContractStatusCancelled = 3
	ContractStatusFinished  = 4
)

type Service interface {
	// ProcessPayment handles the payment simulation and persistence.
	// It includes idempotency checks to prevent double charging for the same contract.
	ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	// ConfirmPendingPayment transitions a pending payment to completed.
	ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// translateError converts database errors to user-friendly messages.
func translateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return errors.New("el contrato o método de pago seleccionado no existe")
		}
	}
	return err
}

// ProcessPayment executes the payment logic with the following sandbox rules:
// 1. If card_number ends in '0000', the payment fails.
// 2. If payment_method_id is 3 (OXXO), the payment is marked as pending.
// 3. Otherwise, the payment is successful.
func (s *service) ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error) {
	// Start Atomic Transaction with Locking
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("fallo al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	// 1. Concurrency Control: Lock the contract row to prevent parallel payments
	contract, err := txRepo.GetContractForPaymentWithLock(ctx, req.ContractID)
	if err != nil {
		return PaymentResponse{}, errors.New("no se pudo encontrar la información del contrato")
	}

	// 2. Contract Status Validation: Only allow payments for valid contracts
	if contract.StatusID == ContractStatusCancelled || contract.StatusID == ContractStatusFinished {
		return PaymentResponse{}, errors.New("operación denegada: el contrato ya está cancelado o finalizado")
	}

	// 3. Authorization Validation: Get contract info and validate ownership
	if contract.ClientID != userID {
		return PaymentResponse{}, errors.New("operación no autorizada: este contrato no pertenece al usuario")
	}

	// 4. Idempotency Check: Validate if a completed payment already exists for this contract
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

	// 5. Orphan References Management: Block new payments if there's a PENDING one (like active OXXO)
	// Unless they are trying to pay with a different immediate method (Card), 
	// but for business integrity, we prefer to block if OXXO is active.
	existingPending, _ := txRepo.GetPaymentByContract(ctx, req.ContractID, PaymentStatusPending)
	if len(existingPending) > 0 {
		p := existingPending[0]
		// If the pending payment is still valid (not expired)
		if time.Now().Before(p.DueDate.Time) {
			return PaymentResponse{}, fmt.Errorf("ya tienes un pago pendiente (OXXO) activo. Por favor págalo o espera a que expire (Expiración: %s)", p.DueDate.Time.Format("15:04"))
		}
	}

	// 6. Security: Validate that the payment amount matches the contract agreed amount
	contractAmount, _ := contract.AgreedAmount.Float64Value()
	if req.Amount != contractAmount.Float64 {
		return PaymentResponse{}, errors.New("el monto del pago no coincide con el monto pactado en el contrato")
	}

	// 7. Sandbox Simulation Logic
	statusID := int32(PaymentStatusCompleted)
	gatewayStatus := "succeeded"
	
	// Rule: Fail if the card number ends in 0000
	if strings.HasSuffix(req.CardNumber, "0000") {
		statusID = PaymentStatusFailed
		gatewayStatus = "declined"
	}
	
	// Rule: Mark as pending if using OXXO (Method 3)
	if req.PaymentMethodID == 3 { 
		statusID = PaymentStatusPending
		gatewayStatus = "pending_payment"
	}

	now := time.Now()
	// Integrity: Set billing_period to the 1st of the month to satisfy DB CHECK constraint
	billingPeriod := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	// Audit: Build metadata for simulation tracking
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

	// 8. Persistence: Register the payment in the database
	payment, err := txRepo.CreatePayment(ctx, sqlcgen.CreatePaymentParams{
		ContractID:       req.ContractID,
		ClientID:         contract.ClientID,
		BillingPeriod:    pgtype.Date{Time: billingPeriod, Valid: true},
		DueDate:          pgtype.Date{Time: now.Add(24 * time.Hour), Valid: true},
		Amount:           pgtype.Numeric{Int: big.NewInt(int64(req.Amount * 100)), Exp: -2, Valid: true},
		PaymentMethodID:  req.PaymentMethodID,
		GatewayID:        req.GatewayID,
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
		PaymentUUID:     payment.PaymentUuid.Bytes,
		StatusID:        payment.StatusID,
		Amount:          req.Amount,
		GatewayID:       payment.GatewayPaymentID.String,
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

	// Security: Only the owner of the payment (client) can confirm it
	if payment.ClientID != userID {
		return errors.New("operación no autorizada: este pago no te pertenece")
	}

	if payment.StatusID != PaymentStatusPending {
		return errors.New("solo se pueden confirmar pagos que estén en estado pendiente")
	}

	// Expiration logic: Check if the reference has expired
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
