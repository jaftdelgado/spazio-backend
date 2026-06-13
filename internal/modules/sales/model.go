package sales

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

const (
	roleAgentID = int32(2)

	modalitySale  = int32(1)
	modalityMixed = int32(3)

	propertyStatusAvailable = int32(2)
	propertyStatusSold      = int32(3)

	transactionStatusPending    = int32(1)
	transactionStatusFormalized = int32(3)
)

// Service defines the business logic for property sales.
type Service interface {
	ConfirmSale(ctx context.Context, auth AuthContext, input SaleInput) (SaleResponse, error)
}

// Repository defines the data access required by sales.
type Repository interface {
	GetSalePropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error)
	GetCurrentSalePriceByPropertyID(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error)
	CreateSaleTransaction(ctx context.Context, arg sqlcgen.CreateSaleTransactionParams) (sqlcgen.CreateSaleTransactionRow, error)
	CreateSalePropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error
	Begin(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) Repository
}

// ContractsClient defines the internal contract creation dependency.
type ContractsClient interface {
	CreateSaleContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error)
}

type AuthContext struct {
	UserID     int32
	RoleID     int32
	AuthHeader string
}

// SaleRequest holds the data needed to formalize a property sale.
type SaleRequest struct {
	PropertyUUID string  `json:"property_uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	AgreedAmount float64 `json:"agreed_amount" example:"1500000.00"`
}

type SaleInput struct {
	PropertyUUID uuid.UUID
	AgreedAmount float64
}

// SaleResponse represents the final outcome of a successful property sale.
type SaleResponse struct {
	TransactionUUID string  `json:"transaction_uuid" format:"uuid" example:"0f8fad5b-d9cb-469f-a165-70867728950e"`
	ContractUUID    string  `json:"contract_uuid" format:"uuid" example:"f68c08c5-e7f0-4aae-b3b1-0d81acf41c09"`
	PropertyUUID    string  `json:"property_uuid" format:"uuid" example:"8d12b3f2-6c8c-4b5f-92c9-2fd58f4f8c01"`
	Status          string  `json:"status" example:"formalized"`
	FinalAmount     float64 `json:"final_amount" example:"1500000.00"`
	Currency        string  `json:"currency" example:"MXN"`
}

type ContractCreateInput struct {
	TransactionID int32   `json:"transaction_id"`
	Currency      string  `json:"currency"`
	AgreedAmount  float64 `json:"agreed_amount"`
}

type ContractCreateResult struct {
	ContractID   int32  `json:"contract_id"`
	ContractUUID string `json:"contract_uuid"`
	StorageKey   string `json:"storage_key"`
	PDFURL       string `json:"pdf_url"`
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

type httpContractsClient struct {
	baseURL string
	client  *http.Client
}
