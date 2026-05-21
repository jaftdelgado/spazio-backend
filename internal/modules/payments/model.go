package payments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Service defines the business logic for the payments module.
type Service interface {
	ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error
	HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error
	ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) (ListPaymentsResult, error)
	GetPaymentByID(ctx context.Context, userID int32, roleID int32, paymentID int32) (PaymentDetail, error)
}

const (
	roleAdminID  int32 = 1
	roleAgentID  int32 = 2
	roleClientID int32 = 3
)

const (
	PaymentStatusPending   int32 = 1
	PaymentStatusCompleted int32 = 2
	PaymentStatusFailed    int32 = 3
	PaymentStatusRefunded  int32 = 4
)

const (
	ContractStatusActive     int32 = 2
	ContractStatusBlocked    int32 = 3
	ContractStatusTerminated int32 = 4
)

var (
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrPaymentForbidden = errors.New("forbidden")
	ErrUnsupportedRole  = errors.New("unsupported user role")
)

type RegisterPaymentRequest struct {
	ContractID      int32   `json:"contract_id" binding:"required"`
	PaymentMethodID int32   `json:"payment_method_id" binding:"required"`
	GatewayID       int32   `json:"gateway_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required"`
	Currency        string  `json:"currency" binding:"required" example:"MXN"`

	Token           string `json:"token,omitempty"`
	GatewayMethodID string `json:"gateway_method_id,omitempty"`
	IssuerID        string `json:"issuer_id,omitempty"`
	Installments    int    `json:"installments,omitempty"`
	PayerEmail      string `json:"payer_email" binding:"required"`
}

type PaymentResponse struct {
	PaymentUUID     uuid.UUID  `json:"payment_uuid"`
	Status          string     `json:"status"`
	StatusID        int32      `json:"status_id"`
	Amount          float64    `json:"amount"`
	PaymentDate     *time.Time `json:"payment_date,omitempty"`
	GatewayID       string     `json:"gateway_payment_id,omitempty"`
	ReferenceNumber *string    `json:"reference_number,omitempty"`
}

type ListPaymentsInput struct {
	PropertyID *int32
	StatusID   *int32
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int32
	Offset     int32
}

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

type PaymentsPagination struct {
	Limit  int32 `json:"limit" example:"20"`
	Offset int32 `json:"offset" example:"0"`
	Total  int64 `json:"total" example:"84"`
}

type ListPaymentsResult struct {
	Data       []PaymentListItem  `json:"data"`
	Pagination PaymentsPagination `json:"pagination"`
}

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
