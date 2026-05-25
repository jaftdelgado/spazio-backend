package payments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
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

var (
	roleAdminID  = shared.RoleAdminID
	roleAgentID  = shared.RoleAgentID
	roleClientID = shared.RoleClientID
)

type RegisterPaymentRequest struct {
	ContractID      int32  `json:"contract_id" binding:"required" example:"10"`
	PaymentMethodID int32  `json:"payment_method_id" binding:"required" example:"1"`
	GatewayID       int32  `json:"gateway_id" binding:"required" example:"1"`
	Amount          int64  `json:"amount" binding:"required" example:"150000"` // Amount in cents
	Currency        string `json:"currency" binding:"required" example:"MXN"`

	Token           string `json:"token,omitempty" example:"tok_123"`
	GatewayMethodID string `json:"gateway_method_id,omitempty" example:"card_456"`
	IssuerID        string `json:"issuer_id,omitempty" example:"bank_789"`
	Installments    int    `json:"installments,omitempty" example:"1"`
	PayerEmail      string `json:"payer_email" binding:"required" example:"user@example.com"`
}

type PaymentResponse struct {
	PaymentUUID     uuid.UUID  `json:"payment_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Status          string     `json:"status" example:"Pending"`
	StatusID        int32      `json:"status_id" example:"1"`
	Amount          int64      `json:"amount" example:"150000"` // Amount in cents
	PaymentDate     *time.Time `json:"payment_date,omitempty" example:"2024-03-08T14:32:00Z"`
	GatewayID       string     `json:"gateway_payment_id,omitempty" example:"pi_123"`
	ReferenceNumber *string    `json:"reference_number,omitempty" example:"REF-123"`
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
	PaymentUUID   uuid.UUID  `json:"payment_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	PaymentID     int32      `json:"-"`
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

type ListPaymentsMeta struct {
	Total int64 `json:"total" example:"84"`
	Shown int   `json:"shown" example:"20"`
}

type ListPaymentsResult struct {
	Data []PaymentListItem `json:"data"`
	Meta ListPaymentsMeta  `json:"meta"`
}

type PaymentDetailRecord struct {
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

type PaymentDetailResponse struct {
	PaymentUUID     uuid.UUID  `json:"payment_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
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
	ClientID        *int32     `json:"client_id,omitempty" example:"7"`
	AgentID         *int32     `json:"agent_id,omitempty" example:"2"`
}

type Repository interface {
	GetPaymentByContract(ctx context.Context, contractID int32, statusID int32) ([]sqlcgen.Payment, error)
	CreatePayment(ctx context.Context, arg sqlcgen.CreatePaymentParams) (sqlcgen.Payment, error)
	GetContractForPayment(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentRow, error)
	GetContractForPaymentWithLock(ctx context.Context, contractID int32) (sqlcgen.GetContractForPaymentWithLockRow, error)
	GetPaymentByUUID(ctx context.Context, paymentUUID uuid.UUID) (sqlcgen.GetPaymentByUUIDRow, error)
	GetPaymentByGatewayID(ctx context.Context, gatewayID string) (sqlcgen.GetPaymentByGatewayIDRow, error)
	GetLastPaidPeriod(ctx context.Context, contractID int32) (pgtype.Date, error)
	GetPendingPayments(ctx context.Context, contractID int32) ([]sqlcgen.GetPendingPaymentsRow, error)
	UpdatePaymentStatus(ctx context.Context, arg sqlcgen.UpdatePaymentStatusParams) error
	WithTx(tx pgx.Tx) Repository
	Begin(ctx context.Context) (pgx.Tx, error)

	ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) ([]PaymentListItem, error)
	GetPaymentDetailByUUID(ctx context.Context, paymentUUID uuid.UUID) (PaymentDetailRecord, error)

	CountCompletedPaymentsForContract(ctx context.Context, contractID int32) (int64, error)
	UpdateTransactionStatusByContract(ctx context.Context, contractID int32, statusID int32) error
	UpdatePropertyStatusByContract(ctx context.Context, contractID int32, statusID int32) error
	UpdateContractStatus(ctx context.Context, contractID int32, statusID int32) error
}

// Service defines the business logic for the payments module.
type Service interface {
	ProcessPayment(ctx context.Context, userID int32, req RegisterPaymentRequest) (PaymentResponse, error)
	ConfirmPendingPayment(ctx context.Context, userID int32, paymentUUID uuid.UUID) error
	HandleWebhook(ctx context.Context, xSignature string, xRequestID string, body []byte) error
	ListPayments(ctx context.Context, userID int32, roleID int32, input ListPaymentsInput) (ListPaymentsResult, error)
	GetPaymentByUUID(ctx context.Context, userID int32, roleID int32, paymentUUID uuid.UUID) (PaymentDetailResponse, error)
}
