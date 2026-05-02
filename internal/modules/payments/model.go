package payments

import (
	"time"

	"github.com/google/uuid"
)

type RegisterPaymentRequest struct {
	ContractID      int32   `json:"contract_id" binding:"required"`
	PaymentMethodID int32   `json:"payment_method_id" binding:"required"`
	GatewayID       int32   `json:"gateway_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required"`
	CardNumber      string  `json:"card_number"` // Para simulación
}

type PaymentResponse struct {
	PaymentUUID     uuid.UUID `json:"payment_uuid"`
	Status          string    `json:"status"`
	StatusID        int32     `json:"status_id"`
	Amount          float64   `json:"amount"`
	PaymentDate     *time.Time `json:"payment_date,omitempty"`
	GatewayID       string    `json:"gateway_payment_id,omitempty"`
	ReferenceNumber *string    `json:"reference_number,omitempty"`
}
