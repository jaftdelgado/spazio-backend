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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin contracts transaction: %w", err)
	}

	return tx, nil
}

func (r *repository) WithTx(tx pgx.Tx) ContractRepository {
	return &repository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

func (r *repository) CreateContract(ctx context.Context, contractUUID uuid.UUID, input CreateContractInput, parentContractID *int32, storageKey string) (sqlcgen.Contract, error) {
	amount := pgtype.Numeric{}
	if err := amount.Scan(fmt.Sprintf("%.2f", input.AgreedAmount)); err != nil {
		return sqlcgen.Contract{}, fmt.Errorf("scan agreed amount: %w", err)
	}

	deposit := pgtype.Numeric{}
	if err := deposit.Scan(fmt.Sprintf("%.2f", input.SecurityDeposit)); err != nil {
		return sqlcgen.Contract{}, fmt.Errorf("scan security deposit: %w", err)
	}

	var endDate pgtype.Date
	if input.EndDate != nil {
		endDate = pgtype.Date{Time: *input.EndDate, Valid: true}
	}

	params := sqlcgen.CreateContractParams{
		ContractUuid:    pgtype.UUID{Bytes: contractUUID, Valid: true},
		TransactionID:   input.TransactionID,
		Currency:        input.Currency,
		AgreedAmount:    amount,
		SecurityDeposit: deposit,
		StorageKey:      storageKey,
		StartDate:       pgtype.Date{Time: input.StartDate, Valid: true},
		EndDate:         endDate,
		StatusID:        1, // Pending/Draft
	}

	if parentContractID != nil {
		params.ParentContractID = pgtype.Int4{Int32: *parentContractID, Valid: true}
	}

	if input.PeriodID != nil {
		params.PeriodID = pgtype.Int4{Int32: *input.PeriodID, Valid: true}
	}

	contract, err := r.queries.CreateContract(ctx, params)
	if err != nil {
		return sqlcgen.Contract{}, fmt.Errorf("create contract: %w", err)
	}

	return contract, nil
}

func (r *repository) FindLatestContractByPropertyAndClient(ctx context.Context, propertyID, clientID int32) (int32, error) {
	id, err := r.queries.FindLatestContractByPropertyAndClient(ctx, sqlcgen.FindLatestContractByPropertyAndClientParams{
		PropertyID: propertyID,
		ClientID:   clientID,
	})
	if err != nil {
		return 0, fmt.Errorf("find latest contract by property and client: %w", err)
	}

	return id, nil
}

func (r *repository) GetContractDataByTransactionID(ctx context.Context, transactionID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	row, err := r.queries.GetContractDataByTransactionID(ctx, transactionID)
	if err != nil {
		return sqlcgen.GetContractDataByTransactionIDRow{}, fmt.Errorf("get contract data by transaction id: %w", err)
	}

	return row, nil
}

func (r *repository) GetPropertyClausesByTransactionID(ctx context.Context, transactionID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	rows, err := r.queries.GetPropertyClausesByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("get property clauses by transaction id: %w", err)
	}

	return rows, nil
}

func (r *repository) GetPropertyServicesByTransactionID(ctx context.Context, transactionID int32) ([]string, error) {
	rows, err := r.queries.GetPropertyServicesByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("get property services by transaction id: %w", err)
	}

	return rows, nil
}

func (r *repository) CheckContractExistsByTransactionID(ctx context.Context, transactionID int32) (bool, error) {
	exists, err := r.queries.CheckContractExistsByTransactionID(ctx, transactionID)
	if err != nil {
		return false, fmt.Errorf("check contract exists by transaction id: %w", err)
	}

	return exists, nil
}

func (r *repository) ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error) {
	rows, err := r.queries.ListContracts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}

	return rows, nil
}

func (r *repository) GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error) {
	row, err := r.queries.GetContractByUUID(ctx, pgtype.UUID{Bytes: contractUUID, Valid: true})
	if err != nil {
		return sqlcgen.GetContractByUUIDRow{}, fmt.Errorf("get contract by uuid: %w", err)
	}

	return row, nil
}

func (r *repository) UpdateTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error {
	err := r.queries.UpdateTransactionStatus(ctx, sqlcgen.UpdateTransactionStatusParams{
		TransactionID: transactionID,
		StatusID:      statusID,
	})
	if err != nil {
		return fmt.Errorf("update transaction status: %w", err)
	}

	return nil
}

func (r *repository) UpdatePropertyStatus(ctx context.Context, propertyID int32, statusID int32) error {
	err := r.queries.UpdatePropertyStatus(ctx, sqlcgen.UpdatePropertyStatusParams{
		PropertyID: propertyID,
		StatusID:   statusID,
	})
	if err != nil {
		return fmt.Errorf("update property status: %w", err)
	}

	return nil
}
