package contracts

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockContractRepo struct {
	ContractRepository
	exists bool
	data   sqlcgen.GetContractByUUIDRow
	txData sqlcgen.GetContractDataByTransactionIDRow
	list   []sqlcgen.ListContractsRow
	err    error
}

func (m *mockContractRepo) CheckContractExistsByTransactionID(ctx context.Context, txID int32) (bool, error) {
	return m.exists, m.err
}

func (m *mockContractRepo) GetContractDataByTransactionID(ctx context.Context, txID int32) (sqlcgen.GetContractDataByTransactionIDRow, error) {
	return m.txData, m.err
}

func (m *mockContractRepo) GetPropertyClausesByTransactionID(ctx context.Context, txID int32) ([]sqlcgen.GetPropertyClausesByTransactionIDRow, error) {
	return nil, nil
}

func (m *mockContractRepo) CreateContract(ctx context.Context, input CreateContractInput, storageKey string) (sqlcgen.Contract, error) {
	return sqlcgen.Contract{}, nil
}

func (m *mockContractRepo) ListContracts(ctx context.Context, params sqlcgen.ListContractsParams) ([]sqlcgen.ListContractsRow, error) {
	return m.list, m.err
}

func (m *mockContractRepo) GetContractByUUID(ctx context.Context, contractUUID uuid.UUID) (sqlcgen.GetContractByUUIDRow, error) {
	return m.data, m.err
}

type mockStorage struct{}

func (m *mockStorage) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	return nil
}

func (m *mockStorage) PublicURL(ctx context.Context, key string) (string, error) {
	return "http://mock.url/" + key, nil
}

func TestGenerateContract_ServiceLogic(t *testing.T) {
	ctx := context.Background()

	amount := pgtype.Numeric{}
	amount.Scan("1500.00")

	validData := sqlcgen.GetContractDataByTransactionIDRow{
		TransactionID:    1,
		OwnerID:          102,
		PropertyStatusID: 2,
		FinalAmount:      amount,
		TransactionType:  "rent",
		PropertyTitle:    "Test House",
		OwnerFirstName:   "John",
		OwnerLastName:    "Doe",
		ClientFirstName:  "Jane",
		ClientLastName:   "Smith",
	}

	tests := []struct {
		name    string
		userID  int32
		input   CreateContractInput
		repo    *mockContractRepo
		wantErr string
	}{
		{
			name:   "success",
			userID: 102,
			input: CreateContractInput{
				TransactionID: 1,
				AgreedAmount:  1500.00,
				StartDate:     time.Now(),
				Currency:      "MXN",
			},
			repo: &mockContractRepo{txData: validData},
		},
		{
			name:    "not owner",
			userID:  999,
			input:   CreateContractInput{TransactionID: 1},
			repo:    &mockContractRepo{txData: validData},
			wantErr: "solo el propietario de la propiedad puede generar el contrato",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &mockStorage{})

			_, err := svc.GenerateContract(ctx, tt.userID, tt.input)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
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
			repo:   &mockContractRepo{data: sqlcgen.GetContractByUUIDRow{ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true}}},
		},
		{
			name:   "success owner",
			userID: 102,
			role:   3,
			repo:   &mockContractRepo{data: sqlcgen.GetContractByUUIDRow{ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true}, OwnerID: 102}},
		},
		{
			name:    "unauthorized",
			userID:  999,
			role:    3,
			repo:    &mockContractRepo{data: sqlcgen.GetContractByUUIDRow{ContractUuid: pgtype.UUID{Bytes: contractUUID, Valid: true}, OwnerID: 102}},
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
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
