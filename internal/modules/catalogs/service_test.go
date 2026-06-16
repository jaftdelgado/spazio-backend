package catalogs

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockCatalogsRepository struct {
	listModalitiesFunc                func(ctx context.Context) ([]Modality, error)
	listPropertyTypesFunc             func(ctx context.Context) ([]PropertyType, error)
	listRentPeriodsByPropertyTypeFunc func(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error)
	listOrientationsFunc              func(ctx context.Context) ([]Orientation, error)
}

func (m *mockCatalogsRepository) ListModalities(ctx context.Context) ([]Modality, error) {
	if m.listModalitiesFunc != nil {
		return m.listModalitiesFunc(ctx)
	}

	return nil, nil
}

func (m *mockCatalogsRepository) ListPropertyTypes(ctx context.Context) ([]PropertyType, error) {
	if m.listPropertyTypesFunc != nil {
		return m.listPropertyTypesFunc(ctx)
	}

	return nil, nil
}

func (m *mockCatalogsRepository) ListRentPeriodsByPropertyType(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error) {
	if m.listRentPeriodsByPropertyTypeFunc != nil {
		return m.listRentPeriodsByPropertyTypeFunc(ctx, propertyTypeID)
	}

	return nil, nil
}

func (m *mockCatalogsRepository) ListOrientations(ctx context.Context) ([]Orientation, error) {
	if m.listOrientationsFunc != nil {
		return m.listOrientationsFunc(ctx)
	}

	return nil, nil
}

func TestService_ListModalities(t *testing.T) {
	tests := []struct {
		name    string
		repo    CatalogsRepository
		want    []Modality
		wantErr bool
	}{
		{
			name: "returns data",
			repo: &mockCatalogsRepository{
				listModalitiesFunc: func(ctx context.Context) ([]Modality, error) {
					return []Modality{{ModalityID: 1, Name: "Rent"}}, nil
				},
			},
			want: []Modality{{ModalityID: 1, Name: "Rent"}},
		},
		{
			name: "returns empty slice",
			repo: &mockCatalogsRepository{
				listModalitiesFunc: func(ctx context.Context) ([]Modality, error) {
					return []Modality{}, nil
				},
			},
			want: []Modality{},
		},
		{
			name: "wraps repository error",
			repo: &mockCatalogsRepository{
				listModalitiesFunc: func(ctx context.Context) ([]Modality, error) {
					return nil, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			result, err := svc.ListModalities(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if result.Data != nil {
					t.Fatalf("expected nil result data on error, got %#v", result.Data)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(result.Data, tt.want) {
				t.Fatalf("unexpected data: got %#v, want %#v", result.Data, tt.want)
			}
		})
	}
}

func TestService_ListPropertyTypes(t *testing.T) {
	tests := []struct {
		name    string
		repo    CatalogsRepository
		want    []PropertyType
		wantErr bool
	}{
		{
			name: "returns data",
			repo: &mockCatalogsRepository{
				listPropertyTypesFunc: func(ctx context.Context) ([]PropertyType, error) {
					icon := "/icons/apartment.svg"
					return []PropertyType{{PropertyTypeID: 1, Name: "Apartment", Icon: &icon, Subtype: "residential"}}, nil
				},
			},
			want: func() []PropertyType {
				icon := "/icons/apartment.svg"
				return []PropertyType{{PropertyTypeID: 1, Name: "Apartment", Icon: &icon, Subtype: "residential"}}
			}(),
		},
		{
			name: "returns empty slice",
			repo: &mockCatalogsRepository{
				listPropertyTypesFunc: func(ctx context.Context) ([]PropertyType, error) {
					return []PropertyType{}, nil
				},
			},
			want: []PropertyType{},
		},
		{
			name: "wraps repository error",
			repo: &mockCatalogsRepository{
				listPropertyTypesFunc: func(ctx context.Context) ([]PropertyType, error) {
					return nil, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			result, err := svc.ListPropertyTypes(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if result.Data != nil {
					t.Fatalf("expected nil result data on error, got %#v", result.Data)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(result.Data, tt.want) {
				t.Fatalf("unexpected data: got %#v, want %#v", result.Data, tt.want)
			}
		})
	}
}

func TestService_ListRentPeriods(t *testing.T) {
	repositoryErr := errors.New("db down")

	tests := []struct {
		name           string
		propertyTypeID int32
		repo           *mockCatalogsRepository
		want           []RentPeriod
		wantErr        bool
		wantRepoID     int32
		repoErr        error
	}{
		{
			name:           "returns rent periods",
			propertyTypeID: 7,
			repo: &mockCatalogsRepository{
				listRentPeriodsByPropertyTypeFunc: func(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error) {
					return []RentPeriod{{PeriodID: 1, Name: "Monthly"}}, nil
				},
			},
			want:       []RentPeriod{{PeriodID: 1, Name: "Monthly"}},
			wantRepoID: 7,
		},
		{
			name:           "returns empty slice",
			propertyTypeID: 7,
			repo: &mockCatalogsRepository{
				listRentPeriodsByPropertyTypeFunc: func(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error) {
					return []RentPeriod{}, nil
				},
			},
			want:       []RentPeriod{},
			wantRepoID: 7,
		},
		{
			name:           "wraps repository error",
			propertyTypeID: 7,
			repo: &mockCatalogsRepository{
				listRentPeriodsByPropertyTypeFunc: func(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error) {
					return nil, repositoryErr
				},
			},
			wantErr:    true,
			wantRepoID: 7,
			repoErr:    repositoryErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRepoID int32
			repo := tt.repo
			originalFunc := repo.listRentPeriodsByPropertyTypeFunc
			repo.listRentPeriodsByPropertyTypeFunc = func(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error) {
				gotRepoID = propertyTypeID
				return originalFunc(ctx, propertyTypeID)
			}

			svc := NewService(repo)
			result, err := svc.ListRentPeriods(context.Background(), tt.propertyTypeID)

			if gotRepoID != tt.wantRepoID {
				t.Fatalf("repository called with propertyTypeID %d, want %d", gotRepoID, tt.wantRepoID)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.repoErr != nil && !errors.Is(err, tt.repoErr) {
					t.Fatalf("expected wrapped repository error %v, got %v", tt.repoErr, err)
				}
				if result.Data != nil {
					t.Fatalf("expected nil result data on error, got %#v", result.Data)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(result.Data, tt.want) {
				t.Fatalf("unexpected data: got %#v, want %#v", result.Data, tt.want)
			}
		})
	}
}

func TestService_ListOrientations(t *testing.T) {
	tests := []struct {
		name    string
		repo    CatalogsRepository
		want    []Orientation
		wantErr bool
	}{
		{
			name: "returns data",
			repo: &mockCatalogsRepository{
				listOrientationsFunc: func(ctx context.Context) ([]Orientation, error) {
					return []Orientation{{OrientationID: 1, Name: "North"}}, nil
				},
			},
			want: []Orientation{{OrientationID: 1, Name: "North"}},
		},
		{
			name: "returns empty slice",
			repo: &mockCatalogsRepository{
				listOrientationsFunc: func(ctx context.Context) ([]Orientation, error) {
					return []Orientation{}, nil
				},
			},
			want: []Orientation{},
		},
		{
			name: "wraps repository error",
			repo: &mockCatalogsRepository{
				listOrientationsFunc: func(ctx context.Context) ([]Orientation, error) {
					return nil, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			result, err := svc.ListOrientations(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if result.Data != nil {
					t.Fatalf("expected nil result data on error, got %#v", result.Data)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(result.Data, tt.want) {
				t.Fatalf("unexpected data: got %#v, want %#v", result.Data, tt.want)
			}
		})
	}
}
