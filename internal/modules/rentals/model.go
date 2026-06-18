package rentals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

const (
	PeriodNightly int32 = 1
	PeriodWeekly  int32 = 2
	PeriodMonthly int32 = 3
	PeriodYearly  int32 = 4

	ModalitySale  int32 = 1
	ModalityRent  int32 = 2
	ModalityMixed int32 = 3

	PropertyStatusReserved  int32 = 1
	PropertyStatusAvailable int32 = 2
	PropertyStatusRented    int32 = 4

	TransactionStatusPending   int32 = 1
	TransactionStatusCompleted int32 = 2
)

// Service defines the business logic for rentals.
type Service interface {
	PreviewRental(ctx context.Context, auth AuthContext, input RentalPreviewInput) (RentalPreviewResponse, error)
	ConfirmRental(ctx context.Context, auth AuthContext, input RentalConfirmInput) (RentalResponse, error)
}

// Repository defines the data access required by rentals.
type Repository interface {
	GetRentalPropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error)
	GetAllowedRentalPeriods(ctx context.Context, propertyTypeID int32) ([]int32, error)
	ListRentalActivePrices(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error)
	ListRentalBlockedDates(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error)
	GetPrimaryRentalAgentForProperty(ctx context.Context, propertyID int32) (int32, error)
	CreateRentalTransaction(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error)
	UpdateRentalPropertyStatus(ctx context.Context, propertyID int32, statusID int32) error
	CreateRentalPropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error
	UpdateRentalTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error
	Begin(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) Repository
}

// ContractsClient defines the internal contract creation dependency.
type ContractsClient interface {
	CreateContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error)
}

type AuthContext struct {
	UserID     int32
	RoleID     int32
	UserUUID   uuid.UUID
	AuthHeader string
}

type RentalPreviewRequest struct {
	PropertyUUID string `json:"property_uuid"`
	PeriodID     int32  `json:"period_id"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

type RentalConfirmRequest struct {
	PropertyUUID string `json:"property_uuid"`
	ClientUUID   string `json:"client_uuid"`
	PeriodID     int32  `json:"period_id"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

type RentalPreviewInput struct {
	PropertyUUID uuid.UUID
	PeriodID     int32
	StartDate    time.Time
	EndDate      time.Time
}

type RentalConfirmInput struct {
	PropertyUUID uuid.UUID
	ClientUUID   uuid.UUID
	PeriodID     int32
	StartDate    time.Time
	EndDate      time.Time
}

type RentalBreakdown struct {
	Years  int32 `json:"years"`
	Months int32 `json:"months"`
	Weeks  int32 `json:"weeks"`
	Nights int32 `json:"nights"`
}

type RentalPriceComponent struct {
	PeriodID  int32  `json:"period_id"`
	Period    string `json:"period"`
	Units     int32  `json:"units"`
	UnitPrice string `json:"unit_price"`
	LineTotal string `json:"line_total"`
}

type RentalPreviewResponse struct {
	PropertyUUID    string                 `json:"property_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Period          string                 `json:"period" example:"Monthly"`
	PeriodID        int32                  `json:"period_id" example:"3"`
	StartDate       string                 `json:"start_date" example:"2026-07-01"`
	EndDate         string                 `json:"end_date" example:"2026-09-30"`
	Units           int32                  `json:"units" example:"3"`
	UnitPrice       string                 `json:"unit_price" example:"5000.00"`
	Currency        string                 `json:"currency" example:"MXN"`
	Subtotal        string                 `json:"subtotal" example:"15000.00"`
	Deposit         string                 `json:"deposit" example:"5000.00"`
	Total           string                 `json:"total" example:"20000.00"`
	IsNegotiable    bool                   `json:"is_negotiable"`
	BlockedDates    []string               `json:"blocked_dates"`
	Breakdown       RentalBreakdown        `json:"breakdown"`
	PriceComponents []RentalPriceComponent `json:"price_components,omitempty"`
}

type RentalResponse struct {
	TransactionUUID string                 `json:"transaction_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	ContractUUID    string                 `json:"contract_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	PropertyUUID    string                 `json:"property_uuid" format:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Status          string                 `json:"status" example:"Completed"`
	Period          string                 `json:"period" example:"Monthly"`
	PeriodID        int32                  `json:"period_id" example:"3"`
	StartDate       string                 `json:"start_date" example:"2026-07-01"`
	EndDate         string                 `json:"end_date" example:"2026-09-30"`
	Currency        string                 `json:"currency" example:"MXN"`
	Subtotal        string                 `json:"subtotal" example:"15000.00"`
	Deposit         string                 `json:"deposit" example:"5000.00"`
	Total           string                 `json:"total" example:"20000.00"`
	IsNegotiable    bool                   `json:"is_negotiable"`
	Breakdown       RentalBreakdown        `json:"breakdown"`
	PriceComponents []RentalPriceComponent `json:"price_components,omitempty"`
}

type ContractCreateInput struct {
	TransactionID   int32   `json:"transaction_id"`
	PeriodID        int32   `json:"period_id"`
	Currency        string  `json:"currency"`
	AgreedAmount    float64 `json:"agreed_amount"`
	SecurityDeposit float64 `json:"security_deposit"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
}

type ContractCreateResult struct {
	ContractID   int32  `json:"contract_id"`
	ContractUUID string `json:"contract_uuid"`
	StorageKey   string `json:"storage_key"`
	PDFUrl       string `json:"pdf_url"`
}

type statusError struct {
	StatusCode int
	Message    string
}

func (e *statusError) Error() string {
	return e.Message
}

func newStatusError(statusCode int, format string, args ...any) error {
	return &statusError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}

type pricingDetails struct {
	RequestedPeriodID   int32
	RequestedPeriodName string
	Units               int32
	UnitPriceCents      int64
	Currency            string
	DepositCents        int64
	SubtotalCents       int64
	TotalCents          int64
	IsNegotiable        bool
	Breakdown           RentalBreakdown
	PriceComponents     []RentalPriceComponent
}
