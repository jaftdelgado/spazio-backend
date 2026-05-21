package payments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (s *service) ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	paymentRecord, err := txRepo.GetPaymentByUUID(ctx, paymentUUID)
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

	err = txRepo.UpdatePaymentStatus(ctx, sqlcgen.UpdatePaymentStatusParams{
		PaymentID:        paymentRecord.PaymentID,
		StatusID:         PaymentStatusCompleted,
		GatewayPaymentID: paymentRecord.GatewayPaymentID,
		GatewayStatus:    pgtype.Text{String: "confirmed", Valid: true},
		PaymentDate:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("actualizar estado de pago: %w", err)
	}

	// F2: Atomic state update in manual confirmation
	contract, err := txRepo.GetContractForPayment(ctx, paymentRecord.ContractID)
	if err != nil {
		return fmt.Errorf("obtener contrato para finalización: %w", err)
	}

	if err := s.finalizeContractState(ctx, txRepo, paymentRecord.ContractID, string(contract.TransactionType)); err != nil {
		return fmt.Errorf("finalizar estado de contrato: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *service) finalizeContractState(ctx context.Context, repo Repository, contractID int32, tType string) error {
	count, err := repo.CountCompletedPaymentsForContract(ctx, contractID)
	if err != nil {
		return err
	}

	if count != 1 {
		return nil
	}

	if err := repo.UpdateTransactionStatusByContract(ctx, contractID, 3); err != nil {
		return fmt.Errorf("close transaction: %w", err)
	}

	newPropStatus := int32(4) // Rented
	if tType == "sale" {
		newPropStatus = 3 // Sold
	}

	if err := repo.UpdatePropertyStatusByContract(ctx, contractID, newPropStatus); err != nil {
		return fmt.Errorf("update property status: %w", err)
	}

	if err := repo.UpdateContractStatus(ctx, contractID, 2); err != nil {
		return fmt.Errorf("activate contract: %w", err)
	}

	return nil
}
