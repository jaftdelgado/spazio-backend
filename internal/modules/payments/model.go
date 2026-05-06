package payments

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrPaymentNotFound is returned when a payment does not exist.
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrPaymentForbidden is returned when the authenticated user cannot access the payment.
	ErrPaymentForbidden = errors.New("forbidden")
	// ErrUnsupportedRole is returned when the user role is not supported by this module.
	ErrUnsupportedRole = errors.New("unsupported user role")
)

// --- UC-16 & UC-17 ---

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

// --- UC-17: Consulting Payments ---

// ListPaymentsInput defines filters for listing payments.
type ListPaymentsInput struct {
	PropertyID *int32
	StatusID   *int32
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int32
	Offset     int32
}

// PaymentListItem represents one item in the paginated payments list.
type PaymentListItem struct {
	PaymentID     int32      `json:"payment_id" example:"1"`
	ContractID    int32      `json:"contract_id" example:"10"`
	PropertyID    int32      `json:"property_id" example:"5"`
	BillingPeriod string     `json:"billing_period" example:"2024-03-01"`
	DueDate       string     `json:"due_date" example:"2024-03-10"`
	Amount        string     `json:"amount" example:"1500.00"`
	Currency      string     `json:"currency" example:"MXN"`
	PaymentMethod string     `json:"payment_method" example:"Transferencia bancaria"`
	Gateway       *string    `json:"gateway" example:"Stripe"`
	Status        string     `json:"status" example:"Pagado"`
	PaymentDate   *time.Time `json:"payment_date" example:"2024-03-08T14:32:00Z"`
	TotalCount    int64      `json:"-"`
}

// PaymentsPagination represents pagination metadata returned by the payments list endpoint.
type PaymentsPagination struct {
	Limit  int32 `json:"limit" example:"20"`
	Offset int32 `json:"offset" example:"0"`
	Total  int64 `json:"total" example:"84"`
}

// ListPaymentsResult is the response payload returned by the payments list use case.
type ListPaymentsResult struct {
	Data       []PaymentListItem  `json:"data"`
	Pagination PaymentsPagination `json:"pagination"`
}

// PaymentDetail represents the serialized payment detail response.
type PaymentDetail struct {
	PaymentID       int32      `json:"payment_id" example:"1"`
	ContractID      int32      `json:"contract_id" example:"10"`
	PropertyID      int32      `json:"property_id" example:"5"`
	TransactionID   int32      `json:"transaction_id" example:"3"`
	TransactionType string     `json:"transaction_type" example:"rent"`
	BillingPeriod   string     `json:"billing_period" example:"2024-03-01"`
	DueDate         string     `json:"due_date" example:"2024-03-10"`
	AgreedAmount    string     `json:"agreed_amount" example:"15000.00"`
	Amount          string     `json:"amount" example:"1500.00"`
	Currency        string     `json:"currency" example:"MXN"`
	PaymentMethod   string     `json:"payment_method" example:"Transferencia bancaria"`
	Gateway         *string    `json:"gateway" example:"Stripe"`
	Status          string     `json:"status" example:"Pagado"`
	PaymentDate     *time.Time `json:"payment_date" example:"2024-03-08T14:32:00Z"`
	ClientID        int32      `json:"client_id" example:"7"`
	AgentID         int32      `json:"agent_id" example:"2"`
}
