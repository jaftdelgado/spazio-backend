package contracts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type ContractRepository interface {
	CreateContract(ctx context.Context, input CreateContractInput, storageKey string) (sqlcgen.Contract, error)
	GetContractDataByTransactionID(ctx context.Context, transactionID int32) (sqlcgen.GetContractDataByTransactionIDRow, error)
	GetPropertyClausesByTransactionID(ctx context.Context, transactionID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error)
	CheckContractExistsByTransactionID(ctx context.Context, transactionID int32) (bool, error)
	ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error)
	GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error)
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

func (r *repository) CreateContract(ctx context.Context, input CreateContractInput, storageKey string) (sqlcgen.Contract, error) {
	contractUUID := uuid.New()

	amount := pgtype.Numeric{}
	amount.Scan(fmt.Sprintf("%f", input.AgreedAmount))

	var endDate pgtype.Date
	if input.EndDate != nil {
		endDate = pgtype.Date{Time: *input.EndDate, Valid: true}
	}

	return r.queries.CreateContract(ctx, sqlcgen.CreateContractParams{
		ContractUuid:  pgtype.UUID{Bytes: contractUUID, Valid: true},
		TransactionID: input.TransactionID,
		Currency:      input.Currency,
		AgreedAmount:  amount,
		StorageKey:    storageKey,
		StartDate:     pgtype.Date{Time: input.StartDate, Valid: true},
		EndDate:       endDate,
		StatusID:      1,
	})
}

func (r *repository) GetContractDataByTransactionID(ctx context.Context, transactionID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	return r.queries.GetContractDataByTransactionID(ctx, transactionID)
}

func (r *repository) GetPropertyClausesByTransactionID(ctx context.Context, transactionID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	return r.queries.GetPropertyClausesByTransactionID(ctx, transactionID)
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
