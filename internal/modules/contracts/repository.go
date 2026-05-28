package contracts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

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

type repository struct {
	db      *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) ContractRepository {
	return &repository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *repository) WithTx(tx pgx.Tx) ContractRepository {
	return &repository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

func (r *repository) CreateContract(ctx context.Context, contractUUID uuid.UUID, input CreateContractInput, parentContractID *int32, storageKey string) (sqlcgen.Contract, error) {
	amount := pgtype.Numeric{}
	// Robust numeric scanning using string representation of big.Rat or simple float formatting
	// for now keeping Scan with fixed precision to avoid binary floating point issues during DB ingestion
	amount.Scan(fmt.Sprintf("%.2f", input.AgreedAmount))

	var endDate pgtype.Date
	if input.EndDate != nil {
		endDate = pgtype.Date{Time: *input.EndDate, Valid: true}
	}

	params := sqlcgen.CreateContractParams{
		ContractUuid:  pgtype.UUID{Bytes: contractUUID, Valid: true},
		TransactionID: input.TransactionID,
		Currency:      input.Currency,
		AgreedAmount:  amount,
		StorageKey:    storageKey,
		StartDate:     pgtype.Date{Time: input.StartDate, Valid: true},
		EndDate:       endDate,
		StatusID:      1, // Pending/Draft
	}

	if parentContractID != nil {
		params.ParentContractID = pgtype.Int4{Int32: *parentContractID, Valid: true}
	}

	if input.PeriodID != nil {
		params.PeriodID = pgtype.Int4{Int32: *input.PeriodID, Valid: true}
	}

	return r.queries.CreateContract(ctx, params)
}

func (r *repository) FindLatestContractByPropertyAndClient(ctx context.Context, propertyID, clientID int32) (int32, error) {
	id, err := r.queries.FindLatestContractByPropertyAndClient(ctx, sqlcgen.FindLatestContractByPropertyAndClientParams{
		PropertyID: propertyID,
		ClientID:   clientID,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) GetContractDataByTransactionID(ctx context.Context, transactionID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	return r.queries.GetContractDataByTransactionID(ctx, transactionID)
}

func (r *repository) GetPropertyClausesByTransactionID(ctx context.Context, transactionID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	return r.queries.GetPropertyClausesByTransactionID(ctx, transactionID)
}

func (r *repository) GetPropertyServicesByTransactionID(ctx context.Context, transactionID int32) ([]string, error) {
	return r.queries.GetPropertyServicesByTransactionID(ctx, transactionID)
}

func (r *repository) CheckContractExistsByTransactionID(ctx context.Context, transactionID int32) (bool, error) {
	return r.queries.CheckContractExistsByTransactionID(ctx, transactionID)
}

func (r *repository) ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error) {
	return r.queries.ListContracts(ctx, params)
}

func (r *repository) GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error) {
	return r.queries.GetContractByUUID(ctx, pgtype.UUID{Bytes: contractUUID, Valid: true})
}

func (r *repository) UpdateTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error {
	return r.queries.UpdateTransactionStatus(ctx, sqlcgen.UpdateTransactionStatusParams{
		TransactionID: transactionID,
		StatusID:      statusID,
	})
}

func (r *repository) UpdatePropertyStatus(ctx context.Context, propertyID int32, statusID int32) error {
	return r.queries.UpdatePropertyStatus(ctx, sqlcgen.UpdatePropertyStatusParams{
		PropertyID: propertyID,
		StatusID:   statusID,
	})
}
