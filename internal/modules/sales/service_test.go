package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockSalesRepository struct {
	getSalePropertyByUUIDFunc           func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error)
	getCurrentSalePriceByPropertyIDFunc func(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error)
	createSaleTransactionFunc           func(ctx context.Context, arg sqlcgen.CreateSaleTransactionParams) (sqlcgen.CreateSaleTransactionRow, error)
	createSalePropertyStatusHistoryFunc func(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error
	beginFunc                           func(ctx context.Context) (pgx.Tx, error)
	withTxFunc                          func(tx pgx.Tx) Repository
}

func (m *mockSalesRepository) GetSalePropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
	return m.getSalePropertyByUUIDFunc(ctx, propertyUUID)
}

func (m *mockSalesRepository) GetCurrentSalePriceByPropertyID(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error) {
	return m.getCurrentSalePriceByPropertyIDFunc(ctx, propertyID)
}

func (m *mockSalesRepository) CreateSaleTransaction(ctx context.Context, arg sqlcgen.CreateSaleTransactionParams) (sqlcgen.CreateSaleTransactionRow, error) {
	return m.createSaleTransactionFunc(ctx, arg)
}

func (m *mockSalesRepository) CreateSalePropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error {
	return m.createSalePropertyStatusHistoryFunc(ctx, arg)
}

func (m *mockSalesRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	return m.beginFunc(ctx)
}

func (m *mockSalesRepository) WithTx(tx pgx.Tx) Repository {
	if m.withTxFunc != nil {
		return m.withTxFunc(tx)
	}

	return m
}

type mockSaleTx struct {
	pgx.Tx
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
}

func (m *mockSaleTx) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}

	return nil
}

func (m *mockSaleTx) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}

	return nil
}

func TestService_ConfirmSale(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	auth := AuthContext{UserID: 11, RoleID: roleAgentID, AuthHeader: "Bearer token"}
	input := SaleInput{
		PropertyUUID: propertyUUID,
		AgreedAmount: 1500000,
	}

	makeRepo := func() *mockSalesRepository {
		return &mockSalesRepository{
			getSalePropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
				return sqlcgen.GetSalePropertyByUUIDRow{
					PropertyID:   15,
					PropertyUuid: pgtype.UUID{Bytes: propertyUUID, Valid: true},
					ModalityID:   modalitySale,
					StatusID:     propertyStatusAvailable,
				}, nil
			},
			getCurrentSalePriceByPropertyIDFunc: func(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error) {
				return sqlcgen.GetCurrentSalePriceByPropertyIDRow{
					SalePrice: numericFromCents(150000000),
					Currency:  "MXN",
				}, nil
			},
			createSaleTransactionFunc: func(ctx context.Context, arg sqlcgen.CreateSaleTransactionParams) (sqlcgen.CreateSaleTransactionRow, error) {
				return sqlcgen.CreateSaleTransactionRow{
					TransactionID:   18,
					TransactionUuid: pgtype.UUID{Bytes: uuid.MustParse("0f8fad5b-d9cb-469f-a165-70867728950e"), Valid: true},
					PropertyID:      15,
					AgentID:         auth.UserID,
					StatusID:        transactionStatusPending,
					FinalAmount:     arg.FinalAmount,
				}, nil
			},
			createSalePropertyStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error {
				return nil
			},
			beginFunc: func(ctx context.Context) (pgx.Tx, error) {
				return &mockSaleTx{}, nil
			},
		}
	}

	tests := []struct {
		name          string
		auth          AuthContext
		mutateRepo    func(repo *mockSalesRepository)
		contractsFunc func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error)
		wantCode      int
		wantContract  string
		wantStatus    string
		wantErrSubstr string
		checkAfter    func(t *testing.T, repo *mockSalesRepository)
	}{
		{
			name:       "non agent role returns 403",
			auth:       AuthContext{UserID: 11, RoleID: 1, AuthHeader: "Bearer token"},
			mutateRepo: func(repo *mockSalesRepository) {},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode: http.StatusForbidden,
		},
		{
			name: "property uuid not found returns 404",
			auth: auth,
			mutateRepo: func(repo *mockSalesRepository) {
				repo.getSalePropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
					return sqlcgen.GetSalePropertyByUUIDRow{}, pgx.ErrNoRows
				}
			},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "property with rent modality returns 422",
			auth: auth,
			mutateRepo: func(repo *mockSalesRepository) {
				repo.getSalePropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
					return sqlcgen.GetSalePropertyByUUIDRow{
						PropertyID:   15,
						PropertyUuid: pgtype.UUID{Bytes: propertyUUID, Valid: true},
						ModalityID:   2,
						StatusID:     propertyStatusAvailable,
					}, nil
				}
			},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "property not available returns 422",
			auth: auth,
			mutateRepo: func(repo *mockSalesRepository) {
				repo.getSalePropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetSalePropertyByUUIDRow, error) {
					return sqlcgen.GetSalePropertyByUUIDRow{
						PropertyID:   15,
						PropertyUuid: pgtype.UUID{Bytes: propertyUUID, Valid: true},
						ModalityID:   modalitySale,
						StatusID:     1,
					}, nil
				}
			},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "missing current sale price returns 422",
			auth: auth,
			mutateRepo: func(repo *mockSalesRepository) {
				repo.getCurrentSalePriceByPropertyIDFunc = func(ctx context.Context, propertyID int32) (sqlcgen.GetCurrentSalePriceByPropertyIDRow, error) {
					return sqlcgen.GetCurrentSalePriceByPropertyIDRow{}, pgx.ErrNoRows
				}
			},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:       "agreed amount mismatch returns 422 with expected price",
			auth:       auth,
			mutateRepo: func(repo *mockSalesRepository) {},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, nil
			},
			wantCode:      http.StatusUnprocessableEntity,
			wantErrSubstr: "expected 1500000.00 MXN",
		},
		{
			name:       "successful flow returns 201 and writes history",
			auth:       auth,
			mutateRepo: func(repo *mockSalesRepository) {},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{
					ContractID:   26,
					ContractUUID: "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09",
					StorageKey:   "contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf",
					PDFURL:       "https://example.com/contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf",
				}, nil
			},
			wantCode:     http.StatusCreated,
			wantContract: "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09",
			wantStatus:   "formalized",
			checkAfter: func(t *testing.T, repo *mockSalesRepository) {
				t.Helper()
			},
		},
		{
			name:       "contracts endpoint failure returns 500 and logs transaction id",
			auth:       auth,
			mutateRepo: func(repo *mockSalesRepository) {},
			contractsFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
				return ContractCreateResult{}, errors.New("contracts endpoint returned status 500")
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := makeRepo()
			tt.mutateRepo(repo)

			var historyArgs []sqlcgen.CreateSalePropertyStatusHistoryParams
			repo.createSalePropertyStatusHistoryFunc = func(ctx context.Context, arg sqlcgen.CreateSalePropertyStatusHistoryParams) error {
				historyArgs = append(historyArgs, arg)
				return nil
			}

			svc := NewService(repo, &mockContractsClient{createSaleContractFunc: tt.contractsFunc})
			amount := input.AgreedAmount
			if strings.Contains(tt.name, "mismatch") {
				amount = 1499999.99
			}

			var logBuffer bytes.Buffer
			originalWriter := log.Writer()
			log.SetOutput(&logBuffer)
			defer log.SetOutput(originalWriter)

			result, err := svc.ConfirmSale(ctx, tt.auth, SaleInput{
				PropertyUUID: input.PropertyUUID,
				AgreedAmount: amount,
			})

			var statusErr *statusError
			if tt.wantCode != http.StatusCreated {
				if !errors.As(err, &statusErr) {
					t.Fatalf("expected statusError, got %v", err)
				}
				if statusErr.StatusCode != tt.wantCode {
					t.Fatalf("status code = %d, want %d", statusErr.StatusCode, tt.wantCode)
				}
				if tt.wantErrSubstr != "" && !strings.Contains(statusErr.Message, tt.wantErrSubstr) {
					t.Fatalf("error message = %q, want substring %q", statusErr.Message, tt.wantErrSubstr)
				}
				if strings.Contains(tt.name, "logs transaction id") {
					logOutput := logBuffer.String()
					if !strings.Contains(logOutput, "transaction_id=18") {
						t.Fatalf("log output = %q, want transaction_id=18", logOutput)
					}
					if !strings.Contains(logOutput, "transaction_uuid=0f8fad5b-d9cb-469f-a165-70867728950e") {
						t.Fatalf("log output = %q, want transaction uuid", logOutput)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ContractUUID != tt.wantContract {
				t.Fatalf("contract_uuid = %s, want %s", result.ContractUUID, tt.wantContract)
			}
			if result.TransactionUUID != "0f8fad5b-d9cb-469f-a165-70867728950e" {
				t.Fatalf("transaction_uuid = %s, want 0f8fad5b-d9cb-469f-a165-70867728950e", result.TransactionUUID)
			}
			if result.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", result.Status, tt.wantStatus)
			}
			if len(historyArgs) != 1 {
				t.Fatalf("history inserts = %d, want 1", len(historyArgs))
			}
			if historyArgs[0].PreviousStatusID != propertyStatusAvailable || historyArgs[0].NewStatusID != propertyStatusSold {
				t.Fatalf("history = %+v, want previous=%d new=%d", historyArgs[0], propertyStatusAvailable, propertyStatusSold)
			}
		})
	}
}

type mockContractsClient struct {
	createSaleContractFunc func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error)
}

func (m *mockContractsClient) CreateSaleContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
	return m.createSaleContractFunc(ctx, authHeader, input)
}

func TestHTTPContractsClient_CreateSaleContract_UsesSaleEndpointAndBody(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotBody ContractCreateInput

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"contract_id":26,"contract_uuid":"f68c08c5-e7f0-4aae-b3b1-0d81acf41c09","storage_key":"contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf","pdf_url":"https://example.com/contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf"}`))
	}))
	defer server.Close()

	client := NewHTTPContractsClient(server.URL)
	result, err := client.CreateSaleContract(context.Background(), "Bearer token", ContractCreateInput{
		TransactionID: 18,
		Currency:      "MXN",
		AgreedAmount:  1500000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/v1/contracts/sale" {
		t.Fatalf("path = %s, want /api/v1/contracts/sale", gotPath)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("authorization = %s, want Bearer token", gotAuth)
	}
	if gotBody.TransactionID != 18 {
		t.Fatalf("transaction_id = %d, want 18", gotBody.TransactionID)
	}
	if gotBody.Currency != "MXN" {
		t.Fatalf("currency = %s, want MXN", gotBody.Currency)
	}
	if gotBody.AgreedAmount != 1500000 {
		t.Fatalf("agreed_amount = %.2f, want 1500000.00", gotBody.AgreedAmount)
	}
	if result.ContractUUID != "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09" {
		t.Fatalf("contract_uuid = %s, want f68c08c5-e7f0-4aae-b3b1-0d81acf41c09", result.ContractUUID)
	}
}
