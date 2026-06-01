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

	exists bool
	data   sqlcgen.GetContractByUUIDRow
	txData sqlcgen.GetContractDataByTransactionIDRow
	list   []sqlcgen.ListContractsRow

	txDataErr         error
	existsErr         error
	clausesErr        error
	servicesErr       error
	beginErr          error
	contractErr       error
	updateTxErr       error
	updatePropertyErr error
	listErr           error
	detailErr         error

	tx                *mockTx
	createdContractID int32
}

func (m *mockContractRepo) CheckContractExistsByTransactionID(ctx context.Context, txID int32) (bool, error) {
	return m.exists, m.existsErr
}

func (m *mockContractRepo) GetContractDataByTransactionID(ctx context.Context, txID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	return m.txData, m.txDataErr
}

func (m *mockContractRepo) GetPropertyClausesByTransactionID(ctx context.Context, txID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	return nil, m.clausesErr
}

func (m *mockContractRepo) GetPropertyServicesByTransactionID(ctx context.Context, txID int32) ([]string, error) {
	if m.servicesErr != nil {
		return nil, m.servicesErr
	}

	return []string{"wifi", "parking"}, nil
}

func (m *mockContractRepo) FindLatestContractByPropertyAndClient(ctx context.Context, propertyID, clientID int32) (int32, error) {
	return 0, nil
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

func (m *mockContractRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}

	if m.tx != nil {
		return m.tx, nil
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

func (m *mockContractRepo) ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error) {
	return m.list, m.listErr
}

func (m *mockContractRepo) GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error) {
	return m.data, m.detailErr
}

type mockTx struct {
	pgx.Tx
	commitErr error
}

func (m *mockTx) Commit(ctx context.Context) error {
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	return nil
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

func validRentContractData(t *testing.T) sqlcgen.GetContractDataByTransactionIDRow {
	t.Helper()

	return sqlcgen.GetContractDataByTransactionIDRow{
		TransactionID:        1,
		TransactionType:      sqlcgen.TransactionTypeRent,
		FinalAmount:          numericFromString(t, "1500.00"),
		PropertyID:           10,
		OwnerID:              102,
		ClientID:             201,
		PropertyStatusID:     2,
		PropertyTitle:        "Test House",
		PropertyDescription:  "Nice house",
		OwnerFirstName:       "John",
		OwnerLastName:        "Doe",
		ClientFirstName:      "Jane",
		ClientLastName:       "Smith",
		Street:               "Main Street",
		ExteriorNumber:       "123",
		Neighborhood:         "Centro",
		CityName:             "Xalapa",
		StateName:            "Veracruz",
		PropertyTypeName:     "Casa",
		LotArea:              numericFromString(t, "120.00"),
		BuiltArea:            numericFromString(t, "90.00"),
		Bedrooms:             pgtype.Int2{Int16: 2, Valid: true},
		Bathrooms:            pgtype.Int2{Int16: 1, Valid: true},
		Floors:               pgtype.Int2{Int16: 1, Valid: true},
		PeriodName:           pgtype.Text{String: "Mensual", Valid: true},
	}
}

func validSaleContractData(t *testing.T) sqlcgen.GetContractDataByTransactionIDRow {
	t.Helper()

	closingDate := time.Now()

	return sqlcgen.GetContractDataByTransactionIDRow{
		TransactionID:        2,
		TransactionType:      sqlcgen.TransactionTypeSale,
		FinalAmount:          numericFromString(t, "850000.00"),
		ClosingDate:          pgtype.Date{Time: closingDate, Valid: true},
		PropertyID:           20,
		OwnerID:              102,
		ClientID:             201,
		PropertyStatusID:     2,
		PropertyTitle:        "Sale House",
		PropertyDescription:  "House for sale",
		OwnerFirstName:       "John",
		OwnerLastName:        "Doe",
		ClientFirstName:      "Jane",
		ClientLastName:       "Smith",
		Street:               "Second Street",
		ExteriorNumber:       "456",
		Neighborhood:         "Centro",
		CityName:             "Xalapa",
		StateName:            "Veracruz",
		PropertyTypeName:     "Casa",
		LotArea:              numericFromString(t, "200.00"),
		BuiltArea:            numericFromString(t, "150.00"),
		Bedrooms:             pgtype.Int2{Int16: 3, Valid: true},
		Bathrooms:            pgtype.Int2{Int16: 2, Valid: true},
		Floors:               pgtype.Int2{Int16: 2, Valid: true},
	}
}

func TestGenerateRentContract_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	validInput := CreateRentContractInput{
		TransactionID: 1,
		PeriodID:      3,
		AgreedAmount:  1500.00,
		StartDate:     startDate,
		EndDate:       endDate,
		Currency:      "MXN",
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
			name:    "success when authenticated user is client",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{},
		},
		{
			name:    "rejects when transaction data cannot be fetched",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txDataErr: errors.New("database error")},
			storage: &mockStorage{},
			wantErr: "fetch transaction data",
		},
		{
			name:    "rejects sale transaction on rent endpoint",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: sqlcgen.GetContractDataByTransactionIDRow{TransactionType: sqlcgen.TransactionTypeSale}},
			storage: &mockStorage{},
			wantErr: "la transacción no corresponde a una renta",
		},
		{
			name:    "rejects user that is not client",
			userID:  999,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{},
			wantErr: "no tiene permiso para generar el contrato de esta renta",
		},
		{
			name: "rejects invalid date range",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1500.00,
				StartDate:     startDate,
				EndDate:       startDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{},
			wantErr: "la fecha de finalización debe ser posterior a la fecha de inicio",
		},
		{
			name:    "rejects when contract existence check fails",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), existsErr: errors.New("exists check failed")},
			storage: &mockStorage{},
			wantErr: "check contract existence",
		},
		{
			name:    "rejects already generated contract",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), exists: true},
			storage: &mockStorage{},
			wantErr: "ya existe un contrato generado para esta transacción",
		},
		{
			name: "rejects amount mismatch",
			userID: 201,
			input: CreateRentContractInput{
				TransactionID: 1,
				PeriodID:      3,
				AgreedAmount:  1499.99,
				StartDate:     startDate,
				EndDate:       endDate,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{},
			wantErr: "no coincide",
		},
		{
			name:    "rejects when property clauses cannot be fetched",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), clausesErr: errors.New("clauses failed")},
			storage: &mockStorage{},
			wantErr: "fetch property clauses",
		},
		{
			name:    "continues when services cannot be fetched",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), servicesErr: errors.New("services failed")},
			storage: &mockStorage{},
		},
		{
			name:    "rejects when transaction cannot begin",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), beginErr: errors.New("begin failed")},
			storage: &mockStorage{},
			wantErr: "begin transaction",
		},
		{
			name:    "rejects when contract record cannot be created",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), contractErr: errors.New("create failed")},
			storage: &mockStorage{},
			wantErr: "create contract record",
		},
		{
			name:    "rejects when storage upload fails",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{uploadErr: errors.New("storage unavailable")},
			wantErr: "upload pdf to storage",
		},
		{
			name:    "rejects when transaction status update fails",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), updateTxErr: errors.New("status update failed")},
			storage: &mockStorage{},
			wantErr: "update transaction status",
		},
		{
			name:    "rejects when transaction commit fails",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t), tx: &mockTx{commitErr: errors.New("commit failed")}},
			storage: &mockStorage{},
			wantErr: "commit transaction",
		},
		{
			name:    "returns success even when public url generation fails",
			userID:  201,
			input:   validInput,
			repo:    &mockContractRepo{txData: validRentContractData(t)},
			storage: &mockStorage{urlErr: errors.New("url failed")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.storage)

			result, err := svc.GenerateRentContract(ctx, tt.userID, tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.ContractID == 0 {
				t.Errorf("expected contract id to be assigned")
			}
		})
	}
}

func TestGenerateSaleContract_ServiceLogic(t *testing.T) {
	ctx := context.Background()

	validInput := CreateSaleContractInput{
		TransactionID: 2,
		AgreedAmount:  850000.00,
		Currency:      "MXN",
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
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t)},
			storage: &mockStorage{},
		},
		{
			name:    "rejects when transaction data cannot be fetched",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txDataErr: errors.New("database error")},
			storage: &mockStorage{},
			wantErr: "fetch transaction data",
		},
		{
			name:    "rejects rent transaction on sale endpoint",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: sqlcgen.GetContractDataByTransactionIDRow{TransactionType: sqlcgen.TransactionTypeRent}},
			storage: &mockStorage{},
			wantErr: "la transacción no corresponde a una venta",
		},
		{
			name:    "rejects user that is not owner agent",
			userID:  999,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t)},
			storage: &mockStorage{},
			wantErr: "no tiene permiso para generar el contrato de esta venta",
		},
		{
			name: "rejects amount mismatch",
			userID: 102,
			input: CreateSaleContractInput{
				TransactionID: 2,
				AgreedAmount:  849999.99,
				Currency:      "MXN",
			},
			repo:    &mockContractRepo{txData: validSaleContractData(t)},
			storage: &mockStorage{},
			wantErr: "no coincide",
		},
		{
			name:    "rejects already generated contract",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t), exists: true},
			storage: &mockStorage{},
			wantErr: "ya existe un contrato generado para esta transacción",
		},
		{
			name:    "rejects when contract record cannot be created",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t), contractErr: errors.New("create failed")},
			storage: &mockStorage{},
			wantErr: "create contract record",
		},
		{
			name:    "rejects when storage upload fails",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t)},
			storage: &mockStorage{uploadErr: errors.New("storage unavailable")},
			wantErr: "upload pdf to storage",
		},
		{
			name:    "rejects when transaction status update fails",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t), updateTxErr: errors.New("transaction update failed")},
			storage: &mockStorage{},
			wantErr: "update transaction status",
		},
		{
			name:    "rejects when property status update fails",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t), updatePropertyErr: errors.New("property update failed")},
			storage: &mockStorage{},
			wantErr: "update property status",
		},
		{
			name:    "rejects when transaction commit fails",
			userID:  102,
			input:   validInput,
			repo:    &mockContractRepo{txData: validSaleContractData(t), tx: &mockTx{commitErr: errors.New("commit failed")}},
			storage: &mockStorage{},
			wantErr: "commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.storage)

			result, err := svc.GenerateSaleContract(ctx, tt.userID, tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.ContractID == 0 {
				t.Errorf("expected contract id to be assigned")
			}
		})
	}
}

func TestListContracts_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	contractUUID := uuid.New()
	search := "casa"
	transactionType := "rent"
	statusID := int32(1)
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)
	ownerID := int32(102)

	row := sqlcgen.ListContractsRow{
		ContractID:      1,
		ContractUuid:    pgtype.UUID{Bytes: contractUUID, Valid: true},
		TransactionType: sqlcgen.TransactionTypeRent,
		PropertyTitle:   "Casa bonita",
		AgreedAmount:    numericFromString(t, "1500.00"),
		Currency:        "MXN",
		StartDate:       pgtype.Date{Time: startDate, Valid: true},
		StatusName:      "Generated",
		ClientName:      "Jane Smith",
		CreatedAt:       pgtype.Timestamptz{Time: startDate, Valid: true},
		OwnerID:         102,
	}

	tests := []struct {
		name    string
		userID  int32
		roleID  int32
		filter  ListContractsFilter
		repo    *mockContractRepo
		wantErr string
	}{
		{
			name:   "admin sees all",
			userID: 1,
			roleID: roleAdminID,
			filter: ListContractsFilter{Page: 1, Limit: 10},
			repo:   &mockContractRepo{list: []sqlcgen.ListContractsRow{row}},
		},
		{
			name:   "agent can use owner filter",
			userID: 2,
			roleID: roleAgentID,
			filter: ListContractsFilter{
				Page:    1,
				Limit:   10,
				OwnerID: &ownerID,
			},
			repo: &mockContractRepo{list: []sqlcgen.ListContractsRow{row}},
		},
		{
			name:   "non admin or agent sees own contracts",
			userID: 201,
			roleID: 3,
			filter: ListContractsFilter{Page: 1, Limit: 10},
			repo:   &mockContractRepo{list: []sqlcgen.ListContractsRow{row}},
		},
		{
			name:   "applies all filters",
			userID: 1,
			roleID: roleAdminID,
			filter: ListContractsFilter{
				Page:            2,
				Limit:           5,
				TransactionType: &transactionType,
				StatusID:        &statusID,
				StartDate:       &startDate,
				EndDate:         &endDate,
				Search:          &search,
			},
			repo: &mockContractRepo{list: []sqlcgen.ListContractsRow{row}},
		},
		{
			name:    "returns repository error",
			userID:  1,
			roleID:  roleAdminID,
			filter:  ListContractsFilter{Page: 1, Limit: 10},
			repo:    &mockContractRepo{listErr: errors.New("database error")},
			wantErr: "list contracts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &mockStorage{})

			result, err := svc.ListContracts(ctx, tt.userID, tt.roleID, tt.filter)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != 1 {
				t.Errorf("expected 1 contract, got %d", len(result))
			}
		})
	}
}

func TestGetContractDetail_ServiceLogic(t *testing.T) {
	ctx := context.Background()
	contractUUID := uuid.New()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	baseRow := sqlcgen.GetContractByUUIDRow{
		ContractUuid:    pgtype.UUID{Bytes: contractUUID, Valid: true},
		StorageKey:      "contracts/test.pdf",
		PropertyTitle:   "Casa bonita",
		OwnerFirstName:  "John",
		OwnerLastName:   "Doe",
		ClientFirstName: "Jane",
		ClientLastName:  "Smith",
		AgreedAmount:    numericFromString(t, "1500.00"),
		Currency:        "MXN",
		PeriodName:      pgtype.Text{String: "Mensual", Valid: true},
		StartDate:       pgtype.Date{Time: startDate, Valid: true},
		EndDate:         pgtype.Date{Time: endDate, Valid: true},
		StatusName:      "Generated",
		OwnerID:         102,
		ClientID:        201,
	}

	tests := []struct {
		name    string
		userID  int32
		roleID  int32
		repo    *mockContractRepo
		storage *mockStorage
		wantErr string
	}{
		{
			name:    "success admin",
			userID:  1,
			roleID:  roleAdminID,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{},
		},
		{
			name:    "success agent",
			userID:  2,
			roleID:  roleAgentID,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{},
		},
		{
			name:    "success owner",
			userID:  102,
			roleID:  3,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{},
		},
		{
			name:    "success client",
			userID:  201,
			roleID:  3,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{},
		},
		{
			name:    "rejects unauthorized user",
			userID:  999,
			roleID:  3,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{},
			wantErr: "no tiene permiso para ver este contrato",
		},
		{
			name:    "returns repository error",
			userID:  1,
			roleID:  roleAdminID,
			repo:    &mockContractRepo{detailErr: errors.New("contract not found")},
			storage: &mockStorage{},
			wantErr: "get contract by uuid",
		},
		{
			name:    "returns public url error",
			userID:  1,
			roleID:  roleAdminID,
			repo:    &mockContractRepo{data: baseRow},
			storage: &mockStorage{urlErr: errors.New("url error")},
			wantErr: "generate contract public url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.storage)

			result, err := svc.GetContractDetail(ctx, tt.userID, tt.roleID, contractUUID)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.PDFUrl == "" {
				t.Errorf("expected pdf url")
			}
		})
	}
}