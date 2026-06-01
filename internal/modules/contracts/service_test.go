package contracts

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockContractRepo struct {
	ContractRepository
	exists            bool
	data              sqlcgen.GetContractByUUIDRow
	txData            sqlcgen.GetContractDataByTransactionIDRow
	list              []sqlcgen.ListContractsRow
	err               error
	contractErr       error
	updateTxErr       error
	updatePropertyErr error
	createdContractID int32
}

func (m *mockContractRepo) CheckContractExistsByTransactionID(ctx context.Context, txID int32) (bool, error) {
	return m.exists, m.err
}

func (m *mockContractRepo) GetContractDataByTransactionID(ctx context.Context, txID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	return m.txData, m.err
}

func (m *mockContractRepo) GetPropertyClausesByTransactionID(ctx context.Context, txID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	return nil, m.err
}

func (m *mockContractRepo) CreateContract(ctx context.Context, contractUUID uuid.UUID, input CreateContractInput, parentContractID *int32, storageKey string) (sqlcgen.Contract, error) {
	if m.contractErr != nil {
		return sqlcgen.Contract{}, m.contractErr
	}

	contractID := m.createdContractID
	if contractID == 0 {
		contractID = 1
	}

	return sqlcgen.Contract{ContractID: contractID}, nil
}

func (m *mockContractRepo) GetPropertyServicesByTransactionID(ctx context.Context, txID int32) ([]string, error) {
	return nil, nil
}

func (m *mockContractRepo) FindLatestContractByPropertyAndClient(ctx context.Context, propertyID, clientID int32) (int32, error) {
	return 0, nil
}

func (m *mockContractRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &mockTx{}, nil
}

func (m *mockContractRepo) WithTx(tx pgx.Tx) ContractRepository {
	return m
}

func (m *mockContractRepo) UpdateTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error {
	return m.updateTxErr
}

func (m *mockContractRepo) UpdatePropertyStatus(ctx context.Context, propertyID int32, statusID int32) error {
	return m.updatePropertyErr
}

type mockTx struct {
	pgx.Tx
}

func (m *mockTx) Commit(ctx context.Context) error {
	return nil
}

func (m *mockTx) Rollback(ctx context.Context) error {
	return nil
}

func (m *mockContractRepo) ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error) {
	return m.list, m.err
}

func (m *mockContractRepo) GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error) {
	return m.data, m.err
}

type mockStorage struct {
	uploadErr error
	urlErr    error
}

func (m *mockStorage) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	return m.uploadErr
}

func (m *mockStorage) PublicURL(ctx context.Context, key string) (string, error) {
	if m.urlErr != nil {
		return "", m.urlErr
	}

	return "http://mock.url/" + key, nil
}

func numericFromString(t *testing.T, value string) pgtype.Numeric {
	t.Helper()

	amount := pgtype.Numeric{}
	if err := amount.Scan(value); err != nil {
		t.Fatalf("failed to scan numeric %s: %v", value, err)
	}

	return amount
}

func TestGenerateRentContract_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	validData := sqlcgen.GetContractDataByTransactionIDRow{
		TransactionID:    1,
		TransactionType:  sqlcgen.TransactionTypeRent,
		FinalAmount:      numericFromString(t, "1500.00"),
		PropertyID:       10,
		OwnerID:          102,
		ClientID:         201,
		PropertyStatusID: 2,
		PropertyTitle:    "Test House",
		OwnerFirstName:   "John",
		OwnerLastName:    "Doe",
		ClientFirstName:  "Jane",
		ClientLastName:   "Smith",
	}

	tests := []struct {
		name    string
		userID  int32
		input   CreateRentContractInput
		repo    *mockContractRepo
		storage *mockStorage
		wantErr string
	}{
		{
			name:   "success when authenticated user is client",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
		},
		{
			name:   "rejects sale transaction on rent endpoint",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo: &mockContractRepo{
				txData: sqlcgen.GetContractDataByTransactionIDRow{
					TransactionType: sqlcgen.TransactionTypeSale,
				},
			},
			storage: &mockStorage{},
			wantErr: "la transacción no corresponde a una renta",
		},
		{
			name:   "rejects user that is not client",
			userID: 999,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
			wantErr: "no tiene permiso para generar el contrato de esta renta",
		},
		{
			name:   "rejects invalid date range",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       startDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
			wantErr: "la fecha de finalización debe ser posterior a la fecha de inicio",
		},
		{
			name:   "rejects already generated contract",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validData, exists: true},
			storage: &mockStorage{},
			wantErr: "ya existe un contrato generado para esta transacción",
		},
		{
			name:   "rejects amount mismatch",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1499.99,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
			wantErr: "no coincide",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.storage)

			_, err := svc.GenerateRentContract(ctx, tt.userID, tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateSaleContract_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	closingDate := time.Now()

	validData := sqlcgen.GetContractDataByTransactionIDRow{
		TransactionID:    2,
		TransactionType:  sqlcgen.TransactionTypeSale,
		FinalAmount:      numericFromString(t, "850000.00"),
		ClosingDate:      pgtype.Date{Time: closingDate, Valid: true},
		PropertyID:       20,
		OwnerID:          102,
		ClientID:         201,
		PropertyStatusID: 2,
		PropertyTitle:    "Sale House",
		OwnerFirstName:   "John",
		OwnerLastName:    "Doe",
		ClientFirstName:  "Jane",
		ClientLastName:   "Smith",
	}

	tests := []struct {
		name    string
		userID  int32
		input   CreateSaleContractInput
		repo    *mockContractRepo
		storage *mockStorage
		wantErr string
	}{
		{
			name:    "success when authenticated user is owner agent",
			userID:  102,
			input:   CreateSaleContractInput{TransactionID: 2, AgreedAmount: 850000.00, Currency: "MXN"},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
		},
		{
			name:    "rejects rent transaction on sale endpoint",
			userID:  102,
			input:   CreateSaleContractInput{TransactionID: 2, AgreedAmount: 850000.00, Currency: "MXN"},
			repo:    &mockContractRepo{txData: sqlcgen.GetContractDataByTransactionIDRow{TransactionType: sqlcgen.TransactionTypeRent}},
			storage: &mockStorage{},
			wantErr: "la transacción no corresponde a una venta",
		},
		{
			name:    "rejects user that is not owner agent",
			userID:  999,
			input:   CreateSaleContractInput{TransactionID: 2, AgreedAmount: 850000.00, Currency: "MXN"},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{},
			wantErr: "no tiene permiso para generar el contrato de esta venta",
		},
		{
			name:    "rolls back when storage upload fails",
			userID:  102,
			input:   CreateSaleContractInput{TransactionID: 2, AgreedAmount: 850000.00, Currency: "MXN"},
			repo:    &mockContractRepo{txData: validData},
			storage: &mockStorage{uploadErr: errors.New("storage unavailable")},
			wantErr: "upload pdf to storage",
		},
		{
			name:    "returns error when property status update fails",
			userID:  102,
			input:   CreateSaleContractInput{TransactionID: 2, AgreedAmount: 850000.00, Currency: "MXN"},
			repo:    &mockContractRepo{txData: validData, updatePropertyErr: errors.New("property update failed")},
			storage: &mockStorage{},
			wantErr: "update property status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.storage)

			_, err := svc.GenerateSaleContract(ctx, tt.userID, tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestListContracts_ServiceLogic(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		userID int32
		role   int32
		repo   *mockContractRepo
	}{
		{
			name:   "admin sees all",
			userID: 1,
			role:   1,
			repo:   &mockContractRepo{list: []sqlcgen.ListContractsRow{{ContractID: 1}}},
		},
		{
			name:   "owner sees own",
			userID: 102,
			role:   3,
			repo:   &mockContractRepo{list: []sqlcgen.ListContractsRow{{ContractID: 1, OwnerID: 102}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &mockStorage{})
			filter := ListContractsFilter{Page: 1, Limit: 10}

			_, err := svc.ListContracts(ctx, tt.userID, tt.role, filter)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetContractDetail_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	contractUUID := uuid.New()

	tests := []struct {
		name    string
		userID  int32
		role    int32
		repo    *mockContractRepo
		wantErr string
	}{
		{
			name:   "success admin",
			userID: 1,
			role:   1,
			repo: &mockContractRepo{
				data: sqlcgen.GetContractByUUIDRow{
					ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true},
				},
			},
		},
		{
			name:   "success owner",
			userID: 102,
			role:   3,
			repo: &mockContractRepo{
				data: sqlcgen.GetContractByUUIDRow{
					ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true},
					OwnerID:      102,
				},
			},
		},
		{
			name:   "success client",
			userID: 201,
			role:   3,
			repo: &mockContractRepo{
				data: sqlcgen.GetContractByUUIDRow{
					ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true},
					ClientID:     201,
				},
			},
		},
		{
			name:   "unauthorized",
			userID: 999,
			role:   3,
			repo: &mockContractRepo{
				data: sqlcgen.GetContractByUUIDRow{
					ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true},
					OwnerID:      102,
					ClientID:     201,
				},
			},
			wantErr: "no tiene permiso para ver este contrato",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &mockStorage{})

			_, err := svc.GetContractDetail(ctx, tt.userID, tt.role, contractUUID)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
