package properties

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestService_GetPrices tests the GetPrices service method in table-driven format.
func TestService_GetPrices(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		repoResult GetPropertyPricesResult
		repoErr    error
		wantErr    bool
		wantErrIs  error
	}{
		{
			name: "returns property prices when prices are found",
			repoResult: GetPropertyPricesResult{
				Data: GetPropertyPricesData{
					SalePrice: &ActiveSalePriceData{
						SalePrice:    1500000,
						Currency:     "MXN",
						IsNegotiable: true,
					},
					RentPrices: []ActiveRentPriceData{
						{PeriodID: 3, RentPrice: 8000, Currency: "MXN"},
					},
				},
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "returns empty property prices when no prices are found",
			repoResult: GetPropertyPricesResult{
				Data: GetPropertyPricesData{
					SalePrice:  nil,
					RentPrices: []ActiveRentPriceData{},
				},
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:       "returns error when property is not found",
			repoResult: GetPropertyPricesResult{},
			repoErr:    ErrPropertyNotFound,
			wantErr:    true,
			wantErrIs:  ErrPropertyNotFound,
		},
		{
			name:       "returns error when repository fails",
			repoResult: GetPropertyPricesResult{},
			repoErr:    errors.New("db"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyPricesFunc: func(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error) {
					return tt.repoResult, tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetPrices(context.Background(), validUUID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.repoResult) {
				t.Fatalf("result mismatch: got %#v want %#v", result, tt.repoResult)
			}
		})
	}
}

// TestService_UpdatePrices tests the UpdatePrices service method in table-driven format.
func TestService_UpdatePrices(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name           string
		input          UpdatePropertyPricesInput
		repoErr        error
		wantErr        bool
		wantRepoCalled bool
		wantInput      UpdatePropertyPricesInput
	}{
		// validatePriceInputs failure cases
		{
			name: "returns error when sale price is zero",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{
					SalePrice: 0,
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when sale price is negative",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{
					SalePrice: -100,
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when rent price period id is zero",
			input: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{
					{PeriodID: 0, RentPrice: 8000},
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when rent price amount is zero",
			input: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{
					{PeriodID: 3, RentPrice: 0},
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "updates sale price successfully",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{
					SalePrice:    1500000,
					IsNegotiable: true,
				},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{
					SalePrice:    1500000,
					IsNegotiable: true,
				},
			},
		},
		{
			name: "updates rent price successfully",
			input: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{
					{PeriodID: 3, RentPrice: 8000, IsNegotiable: false},
				},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{
					{PeriodID: 3, RentPrice: 8000, IsNegotiable: false},
				},
			},
		},
		{
			name: "returns validation error when repository validation fails",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
			repoErr:        ValidationError{Message: "no active price found"},
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
		},
		{
			name: "returns error when property is not found",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
			repoErr:        ErrPropertyNotFound,
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
		},
		{
			name: "returns error when repository fails",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
			repoErr:        errors.New("db"),
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 1500000},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUpdateCalled := false
			var calledInput UpdatePropertyPricesInput

			repo := &mockPropertyRepository{
				updatePropertyPricesFunc: func(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error {
					repoUpdateCalled = true
					calledInput = input
					return tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			err := svc.UpdatePrices(context.Background(), validUUID, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if repoUpdateCalled != tt.wantRepoCalled {
				t.Fatalf("repo called: got %v, want %v", repoUpdateCalled, tt.wantRepoCalled)
			}

			if tt.wantRepoCalled {
				if !reflect.DeepEqual(calledInput, tt.wantInput) {
					t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
				}
			}
		})
	}
}

// TestService_UpdatePrices_ValidationError_IsValidationError verifies that
// validatePriceInputs failures are surfaced as ValidationError, not a generic error.
func TestService_UpdatePrices_ValidationError_IsValidationError(t *testing.T) {
	repo := &mockPropertyRepository{}
	svc := NewService(repo, &mockPropertyPhotoStorage{})

	invalidInputs := []struct {
		name  string
		input UpdatePropertyPricesInput
	}{
		{
			name: "TestService_SalePriceAmountZero",
			input: UpdatePropertyPricesInput{
				SalePrice: &UpdateSalePriceInput{SalePrice: 0},
			},
		},
		{
			name: "TestService_RentPricesPeriodIDZero",
			input: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{{PeriodID: 0, RentPrice: 5000}},
			},
		},
		{
			name: "TestService_RentPricesAmountZero",
			input: UpdatePropertyPricesInput{
				RentPrices: []UpdateRentPriceInput{{PeriodID: 1, RentPrice: 0}},
			},
		},
	}

	for _, tt := range invalidInputs {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdatePrices(context.Background(), "123e4567-e89b-12d3-a456-426614174000", tt.input)
			if err == nil {
				t.Fatal("expected ValidationError, got nil")
			}
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
