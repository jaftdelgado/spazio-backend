package contracts

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type ContractService interface {
	GenerateRentContract(ctx context.Context, userID int32, input CreateRentContractInput) (CreateContractResult, error)
	GenerateSaleContract(ctx context.Context, userID int32, input CreateSaleContractInput) (CreateContractResult, error)
	ListContracts(ctx context.Context, userID int32, roleID int32, filter ListContractsFilter) ([]ContractListItem, error)
	GetContractDetail(ctx context.Context, userID int32, roleID int32, contractUUID uuid.UUID) (ContractDetail, error)
}

type ContractStorage interface {
	Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	PublicURL(ctx context.Context, storageKey string) (string, error)
}

type ContractRepository interface {
	CreateContract(ctx context.Context, contractUUID uuid.UUID, input CreateContractInput, parentContractID *int32, storageKey string) (sqlcgen.Contract, error)
	GetContractDataByTransactionID(ctx context.Context, transactionID int32) (sqlcgen.GetContractDataByTransactionIDRow, error)
	GetPropertyClausesByTransactionID(ctx context.Context, transactionID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error)
	GetPropertyServicesByTransactionID(ctx context.Context, transactionID int32) ([]string, error)
	CheckContractExistsByTransactionID(ctx context.Context, transactionID int32) (bool, error)
	ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error)
	GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error)
	FindLatestContractByPropertyAndClient(ctx context.Context, propertyID, clientID int32) (int32, error)
	UpdateTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error
	UpdatePropertyStatus(ctx context.Context, propertyID int32, statusID int32) error
	Begin(ctx context.Context) (pgx.Tx, error)
	WithTx(tx pgx.Tx) ContractRepository
}

type CreateRentContractInput struct {
	TransactionID int32     `json:"transaction_id"`
	PeriodID      int32     `json:"period_id"`
	Currency      string    `json:"currency"`
	AgreedAmount  float64   `json:"agreed_amount"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

type CreateSaleContractInput struct {
	TransactionID int32   `json:"transaction_id"`
	Currency      string  `json:"currency"`
	AgreedAmount  float64 `json:"agreed_amount"`
}

// CreateContractInput is used internally by the repository to persist rent or sale contracts.
type CreateContractInput struct {
	TransactionID int32
	PeriodID      *int32
	Currency      string
	AgreedAmount  float64
	StartDate     time.Time
	EndDate       *time.Time
}

type CreateContractResult struct {
	ContractID   int32  `json:"contract_id"`
	ContractUUID string `json:"contract_uuid"`
	StorageKey   string `json:"storage_key"`
	PDFUrl       string `json:"pdf_url"`
}

type ContractDetail struct {
	ContractID    int32      `json:"contract_id"`
	ContractUUID  string     `json:"contract_uuid"`
	PropertyTitle string     `json:"property_title"`
	OwnerName     string     `json:"owner_name"`
	ClientName    string     `json:"client_name"`
	AgreedAmount  float64    `json:"agreed_amount"`
	Currency      string     `json:"currency"`
	PeriodName    string     `json:"period_name,omitempty"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Status        string     `json:"status"`
	PDFUrl        string     `json:"pdf_url"`
}

type ListContractsFilter struct {
	OwnerID         *int32     `json:"owner_id"`
	TransactionType *string    `json:"transaction_type"`
	StatusID        *int32     `json:"status_id"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	Search          *string    `json:"search"`
	Page            int32      `json:"page"`
	Limit           int32      `json:"limit"`
}

type ContractListItem struct {
	ContractID      int32      `json:"contract_id"`
	ContractUUID    string     `json:"contract_uuid"`
	TransactionType string     `json:"transaction_type"`
	PropertyTitle   string     `json:"property_title"`
	AgreedAmount    float64    `json:"agreed_amount"`
	Currency        string     `json:"currency"`
	StartDate       time.Time  `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	Status          string     `json:"status"`
	ClientName      string     `json:"client_name"`
	CreatedAt       time.Time  `json:"created_at"`
}
