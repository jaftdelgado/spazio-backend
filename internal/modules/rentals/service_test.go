package rentals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestService_PreviewRental_NightlyOnlyRange_IsAccepted(t *testing.T) {
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
				priceRow(1, "Nightly", "500.00", "1500.00", "MXN", false),
				priceRow(2, "Weekly", "2500.00", "1500.00", "MXN", false),
				priceRow(3, "Monthly", "5000.00", "1500.00", "MXN", false),
			}, nil
		},
		listRentalBlockedDatesFunc: func(ctx context.Context, propertyID int32, startDate, endDate time.Time) ([]sqlcgen.ListRentalBlockedDatesRow, error) {
			return nil, nil
		},
	}, &mockContractsClient{})

	result, err := svc.PreviewRental(ctx, AuthContext{RoleID: 3}, RentalPreviewInput{
		PropertyUUID: propertyUUID,
		PeriodID:     1,
		StartDate:    time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PeriodID != 1 {
		t.Fatalf("period_id = %d, want 1", result.PeriodID)
	}
	if result.Units != 5 {
		t.Fatalf("units = %d, want 5", result.Units)
	}
	if result.Subtotal != "2500.00" {
		t.Fatalf("subtotal = %s, want 2500.00", result.Subtotal)
	}
	if result.Total != "4000.00" {
		t.Fatalf("total = %s, want 4000.00", result.Total)
	}
	if result.Breakdown.Nights != 5 || result.Breakdown.Weeks != 0 || result.Breakdown.Months != 0 || result.Breakdown.Years != 0 {
		t.Fatalf("breakdown = %+v, want only 5 nights", result.Breakdown)
	}
	if len(result.PriceComponents) != 1 {
		t.Fatalf("price_components len = %d, want 1", len(result.PriceComponents))
	}
	if result.PriceComponents[0].PeriodID != 1 || result.PriceComponents[0].Units != 5 || result.PriceComponents[0].LineTotal != "2500.00" {
		t.Fatalf("price_component = %+v, want nightly component for 5 nights", result.PriceComponents[0])
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
	beginCalls := 0
	createTransactionCalls := 0
	insertTxCommitted := false
	postContractTxCommitted := false

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
			createTransactionCalls++
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
			switch beginCalls {
			case 1:
				return &mockRentalTx{
					commitFunc: func(ctx context.Context) error {
						insertTxCommitted = true
						return nil
					},
				}, nil
			case 2:
				return &mockRentalTx{
					commitFunc: func(ctx context.Context) error {
						postContractTxCommitted = true
						return nil
					},
				}, nil
			default:
				t.Fatalf("unexpected Begin call #%d", beginCalls)
				return nil, nil
			}
		},
	}

	contractsClient := &mockContractsClient{
		createContractFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
			contractCalled = true
			if !insertTxCommitted {
				t.Fatalf("contracts should be called only after the transaction insert commit")
			}
			if input.TransactionID != 44 {
				t.Fatalf("transaction_id = %d, want 44", input.TransactionID)
			}
			if input.PeriodID != 3 {
				t.Fatalf("period_id = %d, want 3", input.PeriodID)
			}
			if input.Currency != "MXN" {
				t.Fatalf("currency = %s, want MXN", input.Currency)
			}
			if input.AgreedAmount != 10250.00 { // based on exact mock date range calculation
				t.Fatalf("agreed_amount = %.2f, want %.2f", input.AgreedAmount, 10250.00)
			}
			if input.SecurityDeposit != 5000.00 { // 5000 deposit from db mock
				t.Fatalf("security_deposit = %.2f, want %.2f", input.SecurityDeposit, 5000.00)
			}
			if !input.StartDate.Equal(startDate) {
				t.Fatalf("start_date = %s, want %s", input.StartDate, startDate)
			}
			if !input.EndDate.Equal(endDate) {
				t.Fatalf("end_date = %s, want %s", input.EndDate, endDate)
			}
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
	if createTransactionCalls != 1 {
		t.Fatalf("create transaction calls = %d, want 1", createTransactionCalls)
	}
	if beginCalls != 2 {
		t.Fatalf("begin calls = %d, want 2", beginCalls)
	}
	if !contractCalled || !propertyStatusUpdated || !historyCreated || !transactionCompleted {
		t.Fatalf("expected post-contract local updates to be executed")
	}
	if !postContractTxCommitted {
		t.Fatalf("expected post-contract transaction to commit")
	}
}

func TestService_ConfirmRental_ContractsFailureKeepsPropertyAvailable(t *testing.T) {
	ctx := context.Background()
	propertyUUID := uuid.New()
	clientUUID := uuid.New()

	propertyStatusUpdated := false
	beginCalls := 0
	insertTxCommitted := false
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
				FinalAmount:     numericFromCents(10000),
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
			if beginCalls > 1 {
				t.Fatalf("unexpected second Begin call when contracts creation fails")
			}
			return &mockRentalTx{
				commitFunc: func(ctx context.Context) error {
					insertTxCommitted = true
					return nil
				},
			}, nil
		},
	}

	svc := NewService(repo, &mockContractsClient{
		createContractFunc: func(ctx context.Context, authHeader string, input ContractCreateInput) (ContractCreateResult, error) {
			if !insertTxCommitted {
				t.Fatalf("contracts should be called only after committing the transaction insert")
			}
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
}

func TestHTTPContractsClient_CreateContract_UsesRentEndpointAndBody(t *testing.T) {
	t.Parallel()

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
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := client.CreateContract(context.Background(), "Bearer token", ContractCreateInput{
		TransactionID: 123,
		PeriodID:      3,
		Currency:      "MXN",
		AgreedAmount:  15000,
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/v1/contracts/rent" {
		t.Fatalf("path = %s, want /api/v1/contracts/rent", gotPath)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("authorization = %s, want Bearer token", gotAuth)
	}
	if gotBody.TransactionID != 123 {
		t.Fatalf("transaction_id = %d, want 123", gotBody.TransactionID)
	}
	if gotBody.PeriodID != 3 {
		t.Fatalf("period_id = %d, want 3", gotBody.PeriodID)
	}
	if gotBody.Currency != "MXN" {
		t.Fatalf("currency = %s, want MXN", gotBody.Currency)
	}
	if gotBody.AgreedAmount != 15000 {
		t.Fatalf("agreed_amount = %.2f, want 15000.00", gotBody.AgreedAmount)
	}
	if !gotBody.StartDate.Equal(startDate) {
		t.Fatalf("start_date = %s, want %s", gotBody.StartDate, startDate)
	}
	if !gotBody.EndDate.Equal(endDate) {
		t.Fatalf("end_date = %s, want %s", gotBody.EndDate, endDate)
	}
	if result.ContractUUID != "f68c08c5-e7f0-4aae-b3b1-0d81acf41c09" {
		t.Fatalf("contract_uuid = %s, want f68c08c5-e7f0-4aae-b3b1-0d81acf41c09", result.ContractUUID)
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
