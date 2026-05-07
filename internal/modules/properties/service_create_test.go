package properties

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// validLocation returns a minimal valid CreateLocationInput for tests.
func validLocation() *CreateLocationInput {
	lat := 19.5
	lon := -96.9
	pub := true
	return &CreateLocationInput{
		CityID:          1,
		Street:          "Av. Principal",
		ExteriorNumber:  "45",
		PostalCode:      "91000",
		Latitude:        &lat,
		Longitude:       &lon,
		IsPublicAddress: &pub,
	}
}

// validSalePrice returns a valid CreateSalePriceInput for tests.
func validSalePrice() *CreateSalePriceInput {
	sp := 1500000.0
	return &CreateSalePriceInput{
		SalePrice:    &sp,
		Currency:     "MXN",
		IsNegotiable: ptrBool(true),
	}
}

// validRentPrice returns a valid CreateRentPriceInput for tests.
func validRentPrice(periodID int32) CreateRentPriceInput {
	rp := 8000.0
	return CreateRentPriceInput{
		PeriodID:     periodID,
		RentPrice:    &rp,
		Currency:     "MXN",
		IsNegotiable: ptrBool(false),
	}
}

func validResidentialInput() *CreateResidentialInput {
	return &CreateResidentialInput{
		Bedrooms:         ptrInt16(3),
		Bathrooms:        ptrInt16(2),
		Beds:             ptrInt16(4),
		Floors:           ptrInt16(2),
		ParkingSpots:     ptrInt16(1),
		BuiltArea:        ptrFloat64(120.5),
		ConstructionYear: ptrInt16(2010),
		OrientationID:    ptrInt32(2),
		IsFurnished:      ptrBool(true),
	}
}

func validCommercialInput() *CreateCommercialInput {
	return &CreateCommercialInput{
		CeilingHeight:   ptrFloat64(4.5),
		LoadingDocks:    ptrInt16(2),
		InternalOffices: ptrInt16(3),
		ThreePhasePower: ptrBool(true),
		LandUse:         ptrString("Retail"),
	}
}

// baseInput returns a minimal CreatePropertyInput ready to pass to the service.
func baseInput(modalityID int32) CreatePropertyInput {
	return CreatePropertyInput{
		OwnerID:        1,
		Title:          "Casa",
		PropertyTypeID: 1,
		ModalityID:     modalityID,
		Location:       validLocation(),
	}
}

// TestService_CreateProperty tests CreateProperty in table-driven format.
func TestService_CreateProperty(t *testing.T) {
	tests := []struct {
		name           string
		buildInput     func() CreatePropertyInput
		setupRepo      func() *mockPropertyRepository
		wantErr        bool
		wantValidation bool // true if we expect a ValidationError specifically
		wantRepoCalled bool
		wantUUID       string
	}{
		// Subtype resolution
		{
			name: "returns error when getting property subtype fails",
			buildInput: func() CreatePropertyInput {
				return baseInput(ModalitySale)
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return "", errors.New("db")
					},
				}
			},
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when residential subtype payload is missing",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeResidential, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when commercial subtype payload is missing",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeCommercial, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when residential subtype includes commercial payload",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Residential = validResidentialInput()
				inp.Commercial = validCommercialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeResidential, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when residential subtype bedrooms are missing",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				res := validResidentialInput()
				res.Bedrooms = nil
				inp.Residential = res
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeResidential, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when residential orientation id is zero",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				res := validResidentialInput()
				res.OrientationID = ptrInt32(0)
				inp.Residential = res
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeResidential, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "creates residential property successfully",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Residential = validResidentialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeResidential, nil
					},
					getAllowedPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
						return map[int32]struct{}{}, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "res-123"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "res-123",
		},
		{
			name: "returns validation error when commercial subtype includes residential payload",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Commercial = validCommercialInput()
				inp.Residential = validResidentialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeCommercial, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when commercial subtype ceiling height is missing",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				com := validCommercialInput()
				com.CeilingHeight = nil
				inp.Commercial = com
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeCommercial, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when commercial subtype land use is empty",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				com := validCommercialInput()
				com.LandUse = ptrString("")
				inp.Commercial = com
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeCommercial, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "creates commercial property successfully",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Commercial = validCommercialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeCommercial, nil
					},
					getAllowedPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
						return map[int32]struct{}{}, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "com-123"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "com-123",
		},
		{
			name: "returns validation error when other subtype includes residential payload",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Residential = validResidentialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when other subtype includes commercial payload",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Commercial = validCommercialInput()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "creates other subtype property successfully",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getAllowedPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
						return map[int32]struct{}{}, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "other-123"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "other-123",
		},
		{
			name: "returns validation error when property subtype is unknown",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return "unknown", nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when sale modality is missing sale price",
			buildInput: func() CreatePropertyInput {
				return baseInput(ModalitySale)
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when sale modality includes rent prices",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.RentPrices = []CreateRentPriceInput{validRentPrice(1)}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when rent modality is missing rent prices",
			buildInput: func() CreatePropertyInput {
				return baseInput(ModalityRent)
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when rent modality includes sale price",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalityRent)
				inp.SalePrice = validSalePrice()
				inp.RentPrices = []CreateRentPriceInput{validRentPrice(1)}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when mixed modality is missing sale price",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalityMixed)
				inp.RentPrices = []CreateRentPriceInput{validRentPrice(1)}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when mixed modality is missing rent prices",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalityMixed)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when modality id is invalid",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(99)
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "returns validation error when rent price period id is not allowed",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalityRent)
				inp.RentPrices = []CreateRentPriceInput{validRentPrice(999)}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getAllowedPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
						return map[int32]struct{}{1: {}, 2: {}}, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "creates property successfully with valid rent price period id",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalityRent)
				inp.RentPrices = []CreateRentPriceInput{validRentPrice(1)}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getAllowedPeriodsFunc: func(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
						return map[int32]struct{}{1: {}}, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "abc-123"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "abc-123",
		},
		{
			name: "returns validation error when clause id is invalid",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Clauses = []CreatePropertyClauseInput{{ClauseID: 999, BooleanValue: ptrBool(true)}}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getClauseValueTypesFunc: func(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
						return map[int32]int32{}, nil
					},
				}
			},
			wantErr:        true,
			wantValidation: true,
			wantRepoCalled: false,
		},
		{
			name: "creates property successfully with valid boolean clause",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				inp.Clauses = []CreatePropertyClauseInput{{ClauseID: 1, BooleanValue: ptrBool(true)}}
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getClauseValueTypesFunc: func(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
						return map[int32]int32{1: ClauseValueTypeBoolean}, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "uuid-clause"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "uuid-clause",
		},
		{
			name: "returns error when repository create property fails",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{}, errors.New("db")
					},
				}
			},
			wantErr:        true,
			wantRepoCalled: true,
		},
		// Happy path
		{
			name: "creates property successfully",
			buildInput: func() CreatePropertyInput {
				inp := baseInput(ModalitySale)
				inp.SalePrice = validSalePrice()
				return inp
			},
			setupRepo: func() *mockPropertyRepository {
				return &mockPropertyRepository{
					getPropertySubtypeFunc: func(ctx context.Context, propertyTypeID int32) (string, error) {
						return SubtypeOther, nil
					},
					getModalityNameFunc: func(ctx context.Context, modalityID int32) (string, error) {
						return "Venta", nil
					},
					createPropertyFunc: func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
						return CreatePropertyResult{Data: CreatePropertyResultData{PropertyUUID: "happy-uuid"}}, nil
					},
				}
			},
			wantErr:        false,
			wantRepoCalled: true,
			wantUUID:       "happy-uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoCreateCalled := false
			repo := tt.setupRepo()

			// Wrap createPropertyFunc to track calls.
			originalCreateFunc := repo.createPropertyFunc
			repo.createPropertyFunc = func(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
				repoCreateCalled = true
				if originalCreateFunc != nil {
					return originalCreateFunc(ctx, input)
				}
				return CreatePropertyResult{}, nil
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			input := tt.buildInput()
			result, err := svc.CreateProperty(context.Background(), input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantValidation {
					var ve ValidationError
					if !errors.As(err, &ve) {
						t.Fatalf("expected ValidationError, got %T: %v", err, err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.wantUUID != "" && result.Data.PropertyUUID != tt.wantUUID {
					t.Fatalf("uuid: got %q want %q", result.Data.PropertyUUID, tt.wantUUID)
				}
			}

			if repoCreateCalled != tt.wantRepoCalled {
				t.Fatalf("repo CreateProperty called: got %v, want %v", repoCreateCalled, tt.wantRepoCalled)
			}
		})
	}
}

// TestResolveModalityRequirements tests the pure function resolveModalityRequirements.
func TestResolveModalityRequirements(t *testing.T) {
	tests := []struct {
		modalityID int32
		want       modalityRequirements
		wantErr    bool
	}{
		{
			modalityID: ModalitySale,
			want:       modalityRequirements{RequiresSale: true, RequiresRent: false},
		},
		{
			modalityID: ModalityRent,
			want:       modalityRequirements{RequiresSale: false, RequiresRent: true},
		},
		{
			modalityID: ModalityMixed,
			want:       modalityRequirements{RequiresSale: true, RequiresRent: true},
		},
		{
			modalityID: 99,
			want:       modalityRequirements{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		got, err := resolveModalityRequirements(tt.modalityID)

		if tt.wantErr {
			if err == nil {
				t.Fatalf("resolveModalityRequirements(%d): expected error, got nil", tt.modalityID)
			}
			continue
		}
		if err != nil {
			t.Fatalf("resolveModalityRequirements(%d): unexpected error: %v", tt.modalityID, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("resolveModalityRequirements(%d): got %+v, want %+v", tt.modalityID, got, tt.want)
		}
	}
}

// TestValidateModalityPricing tests the pure function validateModalityPricing.
func TestValidateModalityPricing(t *testing.T) {
	tests := []struct {
		name       string
		modalityID int32
		salePrice  *CreateSalePriceInput
		rentPrices []CreateRentPriceInput
		wantErr    bool
	}{
		{
			name:       "returns nil when sale modality is valid",
			modalityID: ModalitySale,
			salePrice:  validSalePrice(),
			rentPrices: nil,
			wantErr:    false,
		},
		{
			name:       "returns validation error when sale modality is missing sale price",
			modalityID: ModalitySale,
			salePrice:  nil,
			rentPrices: nil,
			wantErr:    true,
		},
		{
			name:       "returns validation error when sale modality includes rent prices",
			modalityID: ModalitySale,
			salePrice:  validSalePrice(),
			rentPrices: []CreateRentPriceInput{validRentPrice(1)},
			wantErr:    true,
		},
		{
			name:       "returns nil when rent modality is valid",
			modalityID: ModalityRent,
			salePrice:  nil,
			rentPrices: []CreateRentPriceInput{validRentPrice(1)},
			wantErr:    false,
		},
		{
			name:       "returns validation error when rent modality is missing rent prices",
			modalityID: ModalityRent,
			salePrice:  nil,
			rentPrices: nil,
			wantErr:    true,
		},
		{
			name:       "returns validation error when rent modality includes sale price",
			modalityID: ModalityRent,
			salePrice:  validSalePrice(),
			rentPrices: []CreateRentPriceInput{validRentPrice(1)},
			wantErr:    true,
		},
		{
			name:       "returns nil when mixed modality is valid",
			modalityID: ModalityMixed,
			salePrice:  validSalePrice(),
			rentPrices: []CreateRentPriceInput{validRentPrice(1)},
			wantErr:    false,
		},
		{
			name:       "returns validation error when mixed modality is missing sale price",
			modalityID: ModalityMixed,
			salePrice:  nil,
			rentPrices: []CreateRentPriceInput{validRentPrice(1)},
			wantErr:    true,
		},
		{
			name:       "returns validation error when mixed modality is missing rent prices",
			modalityID: ModalityMixed,
			salePrice:  validSalePrice(),
			rentPrices: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := resolveModalityRequirements(tt.modalityID)
			if err != nil {
				if tt.wantErr {
					return // invalid modality ID → expected error
				}
				t.Fatalf("resolveModalityRequirements(%d): unexpected error: %v", tt.modalityID, err)
			}

			input := CreatePropertyInput{
				ModalityID: tt.modalityID,
				SalePrice:  tt.salePrice,
				RentPrices: tt.rentPrices,
			}

			err = validateModalityPricing(input, req)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
