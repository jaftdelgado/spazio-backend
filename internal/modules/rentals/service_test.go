package rentals

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type mockRentalsRepository struct {
	getRentalPropertyByUUIDFunc           func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error)
	getAllowedRentalPeriodsFunc           func(ctx context.Context, propertyTypeID int32) ([]int32, error)
	listRentalActivePricesFunc            func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error)
	listRentalBlockedDatesFunc            func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error)
	getPrimaryRentalAgentForPropertyFunc  func(ctx context.Context, propertyID int32) (int32, error)
	createRentalTransactionFunc           func(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error)
	updateRentalPropertyStatusFunc        func(ctx context.Context, propertyID int32, statusID int32) error
	createRentalPropertyStatusHistoryFunc func(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error
	updateRentalTransactionStatusFunc     func(ctx context.Context, transactionID int32, statusID int32) error
	beginFunc                             func(ctx context.Context) (pgx.Tx, error)
	withTxFunc                            func(tx pgx.Tx) Repository
}

func (m *mockRentalsRepository) GetRentalPropertyByUUID(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
	return m.getRentalPropertyByUUIDFunc(ctx, propertyUUID)
}

func (m *mockRentalsRepository) GetAllowedRentalPeriods(ctx context.Context, propertyTypeID int32) ([]int32, error) {
	return m.getAllowedRentalPeriodsFunc(ctx, propertyTypeID)
}

func (m *mockRentalsRepository) ListRentalActivePrices(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
	return m.listRentalActivePricesFunc(ctx, propertyID)
}

func (m *mockRentalsRepository) ListRentalBlockedDates(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
	return m.listRentalBlockedDatesFunc(ctx, propertyID, startDate, endDate)
}

func (m *mockRentalsRepository) GetPrimaryRentalAgentForProperty(ctx context.Context, propertyID int32) (int32, error) {
	return m.getPrimaryRentalAgentForPropertyFunc(ctx, propertyID)
}

func (m *mockRentalsRepository) CreateRentalTransaction(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
	return m.createRentalTransactionFunc(ctx, arg)
}

func (m *mockRentalsRepository) UpdateRentalPropertyStatus(ctx context.Context, propertyID int32, statusID int32) error {
	return m.updateRentalPropertyStatusFunc(ctx, propertyID, statusID)
}

func (m *mockRentalsRepository) CreateRentalPropertyStatusHistory(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error {
	return m.createRentalPropertyStatusHistoryFunc(ctx, arg)
}

func (m *mockRentalsRepository) UpdateRentalTransactionStatus(ctx context.Context, transactionID int32, statusID int32) error {
	return m.updateRentalTransactionStatusFunc(ctx, transactionID, statusID)
}

func (m *mockRentalsRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	return m.beginFunc(ctx)
}

func (m *mockRentalsRepository) WithTx(tx pgx.Tx) Repository {
	if m.withTxFunc != nil {
		return m.withTxFunc(tx)
	}
	return m
}

type mockContractsClient struct {
	createContractFunc func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error)
}

func (m *mockContractsClient) CreateContract(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
	return m.createContractFunc(ctx, authHeader, input)
}

type mockRentalTx struct {
	pgx.Tx
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
}

func (m *mockRentalTx) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockRentalTx) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return nil
}

func TestService_PreviewRental(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	auth := AuthContext{UserID: 7, RoleID: 3, UserUUID: uuid.New()}
	input := RentalPreviewInput{
		PropertyUUID: propertyUUID,
		PeriodID:     3,
		StartDate:    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC),
	}

	makeRepo := func() *mockRentalsRepository {
		return &mockRentalsRepository{
			getRentalPropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
				return sqlcgen.GetRentalPropertyByUUIDRow{
					PropertyID:     10,
					PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
					PropertyTypeID: 2,
					ModalityID:     2,
					StatusID:       2,
				}, nil
			},
			getAllowedRentalPeriodsFunc: func(ctx context.Context, propertyTypeID int32) ([]int32, error) {
				return []int32{1, 2, 3, 4}, nil
			},
			listRentalActivePricesFunc: func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
				return []sqlcgen.ListRentalActivePricesRow{
					priceRow(1, "Nightly", "500.00", "5000.00", "MXN", false),
					priceRow(2, "Weekly", "2500.00", "5000.00", "MXN", false),
					priceRow(3, "Monthly", "5000.00", "5000.00", "MXN", false),
				}, nil
			},
			listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
				return nil, nil
			},
			getPrimaryRentalAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) {
				return 5, nil
			},
			createRentalTransactionFunc: func(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
				return sqlcgen.Transaction{}, nil
			},
			updateRentalPropertyStatusFunc:        func(ctx context.Context, propertyID int32, statusID int32) error { return nil },
			createRentalPropertyStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error { return nil },
			updateRentalTransactionStatusFunc:     func(ctx context.Context, transactionID int32, statusID int32) error { return nil },
			beginFunc:                             func(ctx context.Context) (pgx.Tx, error) { return &mockRentalTx{}, nil },
		}
	}

	tests := []struct {
		name       string
		auth       AuthContext
		mutateRepo func(repo *mockRentalsRepository)
		wantCode   int
		wantTotal  string
		wantMonths int32
		wantWeeks  int32
		wantNights int32
	}{
		{
			name:       "role different than client returns 403",
			auth:       AuthContext{UserID: 7, RoleID: 2, UserUUID: uuid.New()},
			mutateRepo: func(repo *mockRentalsRepository) {},
			wantCode:   http.StatusForbidden,
		},
		{
			name: "property uuid not found returns 404",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.getRentalPropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
					return sqlcgen.GetRentalPropertyByUUIDRow{}, errors.New("no rows")
				}
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "property with sale modality returns 422",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.getRentalPropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
					return sqlcgen.GetRentalPropertyByUUIDRow{
						PropertyID:     10,
						PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
						PropertyTypeID: 2,
						ModalityID:     1,
						StatusID:       2,
					}, nil
				}
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "property not available returns 422",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.getRentalPropertyByUUIDFunc = func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
					return sqlcgen.GetRentalPropertyByUUIDRow{
						PropertyID:     10,
						PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
						PropertyTypeID: 2,
						ModalityID:     2,
						StatusID:       4,
					}, nil
				}
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "period not allowed returns 422",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.getAllowedRentalPeriodsFunc = func(ctx context.Context, propertyTypeID int32) ([]int32, error) {
					return []int32{1, 2}, nil
				}
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "missing current requested price returns 422",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.listRentalActivePricesFunc = func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
					return []sqlcgen.ListRentalActivePricesRow{
						priceRow(1, "Nightly", "500.00", "5000.00", "MXN", false),
						priceRow(2, "Weekly", "2500.00", "5000.00", "MXN", false),
					}, nil
				}
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "blocked dates return 422",
			auth: auth,
			mutateRepo: func(repo *mockRentalsRepository) {
				repo.listRentalBlockedDatesFunc = func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
					return []sqlcgen.ListRentalBlockedDatesRow{{ExceptionDate: pgtype.Date{Time: startDate.AddDate(0, 0, 3), Valid: true}}}, nil
				}
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:       "successful monthly flow decomposes into month weeks and nights",
			auth:       auth,
			mutateRepo: func(repo *mockRentalsRepository) {},
			wantCode:   http.StatusOK,
			wantTotal:  "15500.00",
			wantMonths: 1,
			wantWeeks:  2,
			wantNights: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := makeRepo()
			tt.mutateRepo(repo)
			svc := NewService(repo, &mockContractsClient{})

			result, err := svc.PreviewRental(ctx, tt.auth, input)
			if tt.wantCode != http.StatusOK {
				var statusErr *statusError
				if !errors.As(err, &statusErr) {
					t.Fatalf("expected status error, got %v", err)
				}
				if statusErr.StatusCode != tt.wantCode {
					t.Fatalf("status code = %d, want %d", statusErr.StatusCode, tt.wantCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Fatalf("total = %s, want %s", result.Total, tt.wantTotal)
			}
			if result.Breakdown.Months != tt.wantMonths || result.Breakdown.Weeks != tt.wantWeeks || result.Breakdown.Nights != tt.wantNights {
				t.Fatalf("breakdown = %+v, want months=%d weeks=%d nights=%d", result.Breakdown, tt.wantMonths, tt.wantWeeks, tt.wantNights)
			}
		})
	}
}

func TestService_PreviewRental_StartDateGreaterOrEqualEndDate_Returns400(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	svc := NewService(&mockRentalsRepository{
		getRentalPropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
			return sqlcgen.GetRentalPropertyByUUIDRow{
				PropertyID:     10,
				PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
				PropertyTypeID: 2,
				ModalityID:     2,
				StatusID:       2,
			}, nil
		},
		getAllowedRentalPeriodsFunc: func(ctx context.Context, propertyTypeID int32) ([]int32, error) { return []int32{3}, nil },
		listRentalActivePricesFunc: func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
			return []sqlcgen.ListRentalActivePricesRow{priceRow(3, "Monthly", "5000.00", "5000.00", "MXN", false)}, nil
		},
		listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
			return nil, nil
		},
	}, &mockContractsClient{})

	_, err := svc.PreviewRental(ctx, AuthContext{RoleID: 3}, RentalPreviewInput{
		PropertyUUID: propertyUUID,
		PeriodID:     3,
		StartDate:    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	var statusErr *statusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", err)
	}
}

func TestService_PreviewRental_YearlyIncludesMonthlyRemainder(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	svc := NewService(&mockRentalsRepository{
		getRentalPropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
			return sqlcgen.GetRentalPropertyByUUIDRow{
				PropertyID:     10,
				PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
				PropertyTypeID: 2,
				ModalityID:     2,
				StatusID:       2,
			}, nil
		},
		getAllowedRentalPeriodsFunc: func(ctx context.Context, propertyTypeID int32) ([]int32, error) {
			return []int32{1, 2, 3, 4}, nil
		},
		listRentalActivePricesFunc: func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
			return []sqlcgen.ListRentalActivePricesRow{
				priceRow(1, "Nightly", "500.00", "30000.00", "MXN", false),
				priceRow(3, "Monthly", "12000.00", "30000.00", "MXN", false),
				priceRow(4, "Yearly", "160000.00", "30000.00", "MXN", false),
			}, nil
		},
		listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
			return nil, nil
		},
		getPrimaryRentalAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) {
			return 5, nil
		},
		createRentalTransactionFunc: func(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
			return sqlcgen.Transaction{}, nil
		},
		updateRentalPropertyStatusFunc:        func(ctx context.Context, propertyID int32, statusID int32) error { return nil },
		createRentalPropertyStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error { return nil },
		updateRentalTransactionStatusFunc:     func(ctx context.Context, transactionID int32, statusID int32) error { return nil },
		beginFunc:                             func(ctx context.Context) (pgx.Tx, error) { return &mockRentalTx{}, nil },
	}, &mockContractsClient{})

	result, err := svc.PreviewRental(ctx, AuthContext{RoleID: 3}, RentalPreviewInput{
		PropertyUUID: propertyUUID,
		PeriodID:     4,
		StartDate:    time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2027, 11, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subtotal != "196000.00" {
		t.Fatalf("subtotal = %s, want %s", result.Subtotal, "196000.00")
	}
	if result.Total != "226000.00" {
		t.Fatalf("total = %s, want %s", result.Total, "226000.00")
	}
	if result.Breakdown.Years != 1 || result.Breakdown.Months != 3 {
		t.Fatalf("breakdown = %+v, want 1 year and 3 months", result.Breakdown)
	}
	if len(result.PriceComponents) != 2 {
		t.Fatalf("price_components len = %d, want 2", len(result.PriceComponents))
	}
}

func TestService_ConfirmRental(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	clientUUID := uuid.New()
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	contractCalled := false
	propertyStatusUpdated := false
	historyCreated := false
	transactionCompleted := false
	var createdTransactionAmount pgtype.Numeric
	var contractInput ContractCreateInput
	beginCalls := 0

	createTx := &mockRentalTx{}
	updateTx := &mockRentalTx{}

	repo := &mockRentalsRepository{
		getRentalPropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
			return sqlcgen.GetRentalPropertyByUUIDRow{
				PropertyID:     10,
				PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
				PropertyTypeID: 2,
				ModalityID:     2,
				StatusID:       2,
			}, nil
		},
		getAllowedRentalPeriodsFunc: func(ctx context.Context, propertyTypeID int32) ([]int32, error) { return []int32{1, 2, 3, 4}, nil },
		listRentalActivePricesFunc: func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
			return []sqlcgen.ListRentalActivePricesRow{
				priceRow(1, "Nightly", "500.00", "5000.00", "MXN", false),
				priceRow(2, "Weekly", "2500.00", "5000.00", "MXN", false),
				priceRow(3, "Monthly", "5000.00", "5000.00", "MXN", false),
			}, nil
		},
		listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
			return nil, nil
		},
		getPrimaryRentalAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 88, nil },
		createRentalTransactionFunc: func(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
			createdTransactionAmount = arg.FinalAmount
			return sqlcgen.Transaction{
				TransactionID:   44,
				TransactionUuid: pgtype.UUID{Bytes: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true},
				FinalAmount:     arg.FinalAmount,
			}, nil
		},
		updateRentalPropertyStatusFunc: func(ctx context.Context, propertyID int32, statusID int32) error {
			propertyStatusUpdated = true
			return nil
		},
		createRentalPropertyStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error {
			historyCreated = true
			return nil
		},
		updateRentalTransactionStatusFunc: func(ctx context.Context, transactionID int32, statusID int32) error {
			transactionCompleted = true
			return nil
		},
		beginFunc: func(ctx context.Context) (pgx.Tx, error) {
			beginCalls++
			if beginCalls == 1 {
				return createTx, nil
			}
			return updateTx, nil
		},
	}

	contractsClient := &mockContractsClient{
		createContractFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
			contractCalled = true
			contractInput = input
			return ContractCreateResult{
				ContractUUID: "223e4567-e89b-12d3-a456-426614174000",
			}, nil
		},
	}

	svc := NewService(repo, contractsClient)
	result, err := svc.ConfirmRental(ctx, AuthContext{
		UserID:     7,
		RoleID:     3,
		UserUUID:   clientUUID,
		AuthHeader: "Bearer token",
	}, RentalConfirmInput{
		PropertyUUID: propertyUUID,
		ClientUUID:   clientUUID,
		PeriodID:     3,
		StartDate:    startDate,
		EndDate:      endDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TransactionUUID == "" || result.ContractUUID == "" {
		t.Fatalf("expected transaction and contract UUIDs, got %+v", result)
	}
	if !contractCalled || !propertyStatusUpdated || !historyCreated || !transactionCompleted {
		t.Fatalf("expected post-contract local updates to be executed")
	}
	if beginCalls != 2 {
		t.Fatalf("begin calls = %d, want 2", beginCalls)
	}
	createdAmount, err := numericToFloat(createdTransactionAmount)
	if err != nil {
		t.Fatalf("numericToFloat(createdTransactionAmount) error = %v", err)
	}
	if contractInput.TransactionID != 44 {
		t.Fatalf("contract input transaction_id = %d, want %d", contractInput.TransactionID, 44)
	}
	if contractInput.PeriodID != 3 {
		t.Fatalf("contract input period_id = %d, want %d", contractInput.PeriodID, 3)
	}
	if contractInput.Currency != "MXN" {
		t.Fatalf("contract input currency = %s, want MXN", contractInput.Currency)
	}
	if contractInput.AgreedAmount != createdAmount {
		t.Fatalf("contract agreed_amount = %.2f, want %.2f", contractInput.AgreedAmount, createdAmount)
	}
	if !contractInput.StartDate.Equal(startDate) {
		t.Fatalf("contract input start_date = %s, want %s", contractInput.StartDate, startDate)
	}
	if !contractInput.EndDate.Equal(endDate) {
		t.Fatalf("contract input end_date = %s, want %s", contractInput.EndDate, endDate)
	}
}

func TestService_ConfirmRental_ContractsFailureKeepsPropertyAvailable(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	clientUUID := uuid.New()

	propertyStatusUpdated := false
	beginCalls := 0
	repo := &mockRentalsRepository{
		getRentalPropertyByUUIDFunc: func(ctx context.Context, propertyUUID uuid.UUID) (sqlcgen.GetRentalPropertyByUUIDRow, error) {
			return sqlcgen.GetRentalPropertyByUUIDRow{
				PropertyID:     10,
				PropertyUuid:   pgtype.UUID{Bytes: propertyUUID, Valid: true},
				PropertyTypeID: 2,
				ModalityID:     2,
				StatusID:       2,
			}, nil
		},
		getAllowedRentalPeriodsFunc: func(ctx context.Context, propertyTypeID int32) ([]int32, error) { return []int32{3}, nil },
		listRentalActivePricesFunc: func(ctx context.Context, propertyID int32) ([]sqlcgen.ListRentalActivePricesRow, error) {
			return []sqlcgen.ListRentalActivePricesRow{priceRow(3, "Monthly", "5000.00", "5000.00", "MXN", false)}, nil
		},
		listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
			return nil, nil
		},
		getPrimaryRentalAgentForPropertyFunc: func(ctx context.Context, propertyID int32) (int32, error) { return 88, nil },
		createRentalTransactionFunc: func(ctx context.Context, arg sqlcgen.CreateRentalTransactionParams) (sqlcgen.Transaction, error) {
			return sqlcgen.Transaction{
				TransactionID:   44,
				TransactionUuid: pgtype.UUID{Bytes: uuid.New(), Valid: true},
				FinalAmount:     arg.FinalAmount,
			}, nil
		},
		updateRentalPropertyStatusFunc: func(ctx context.Context, propertyID int32, statusID int32) error {
			propertyStatusUpdated = true
			return nil
		},
		createRentalPropertyStatusHistoryFunc: func(ctx context.Context, arg sqlcgen.CreateRentalPropertyStatusHistoryParams) error { return nil },
		updateRentalTransactionStatusFunc:     func(ctx context.Context, transactionID int32, statusID int32) error { return nil },
		beginFunc: func(ctx context.Context) (pgx.Tx, error) {
			beginCalls++
			return &mockRentalTx{}, nil
		},
	}

	svc := NewService(repo, &mockContractsClient{
		createContractFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
			return ContractCreateResult{}, errors.New("boom")
		},
	})

	_, err := svc.ConfirmRental(ctx, AuthContext{
		UserID:     7,
		RoleID:     3,
		UserUUID:   clientUUID,
		AuthHeader: "Bearer token",
	}, RentalConfirmInput{
		PropertyUUID: propertyUUID,
		ClientUUID:   clientUUID,
		PeriodID:     3,
		StartDate:    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	})
	var statusErr *statusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %v", err)
	}
	if propertyStatusUpdated {
		t.Fatalf("property status should remain available when contracts fails")
	}
	if beginCalls != 1 {
		t.Fatalf("begin calls = %d, want %d", beginCalls, 1)
	}
}

func TestHTTPContractsClient_CreateContract_UsesRentEndpointAndBody(t *testing.T) {
	var receivedPath string
	var receivedAuth string
	var receivedBody ContractCreateInput

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")

		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(`{"contract_id":26,"contract_uuid":"f68c08c5-e7f0-4aae-b3b1-0d81acf41c09","storage_key":"contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf","pdf_url":"https://example.com/contracts/f68c08c5-e7f0-4aae-b3b1-0d81acf41c09.pdf"}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewHTTPContractsClient(server.URL)
	input := ContractCreateInput{
		TransactionID: 123,
		PeriodID:      3,
		Currency:      "MXN",
		AgreedAmount:  15000.00,
		StartDate:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}

	result, err := client.CreateContract(context.Background(), "Bearer token", input)
	if err != nil {
		t.Fatalf("CreateContract() error = %v", err)
	}

	if receivedPath != "/api/v1/contracts/rent" {
		t.Fatalf("path = %s, want %s", receivedPath, "/api/v1/contracts/rent")
	}
	if receivedAuth != "Bearer token" {
		t.Fatalf("authorization = %s, want Bearer token", receivedAuth)
	}
	if receivedBody != input {
		t.Fatalf("request body = %+v, want %+v", receivedBody, input)
	}
	if result.ContractUUID != "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09" {
		t.Fatalf("contract_uuid = %s, want %s", result.ContractUUID, "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09")
	}
}

func TestHTTPContractsClient_CreateContract_ReturnsErrorForNonCreatedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewHTTPContractsClient(server.URL)
	_, err := client.CreateContract(context.Background(), "Bearer token", ContractCreateInput{
		TransactionID: 1,
		PeriodID:      3,
		Currency:      "MXN",
		AgreedAmount:  15500.00,
		StartDate:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func priceRow(periodID int32, periodName, rentPrice, deposit, currency string, isNegotiable bool) sqlcgen.ListRentalActivePricesRow {
	var priceNumeric pgtype.Numeric
	priceNumeric.Scan(rentPrice)
	var depositNumeric pgtype.Numeric
	depositNumeric.Scan(deposit)
	return sqlcgen.ListRentalActivePricesRow{
		PeriodID:     periodID,
		PeriodName:   periodName,
		RentPrice:    priceNumeric,
		Deposit:      depositNumeric,
		Currency:     currency,
		IsNegotiable: isNegotiable,
	}
}
