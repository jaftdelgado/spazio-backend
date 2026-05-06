package locations

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockLocationsRepository struct {
	listCountriesFunc func(ctx context.Context) ([]Country, error)
	listStatesFunc    func(ctx context.Context, countryID int32) ([]State, error)
	listCitiesFunc    func(ctx context.Context, input ListCitiesInput) ([]City, int64, error)
}

func (m *mockLocationsRepository) ListCountries(ctx context.Context) ([]Country, error) {
	if m.listCountriesFunc != nil {
		return m.listCountriesFunc(ctx)
	}
	return nil, nil
}

func (m *mockLocationsRepository) ListStates(ctx context.Context, countryID int32) ([]State, error) {
	if m.listStatesFunc != nil {
		return m.listStatesFunc(ctx, countryID)
	}
	return nil, nil
}

func (m *mockLocationsRepository) ListCities(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
	if m.listCitiesFunc != nil {
		return m.listCitiesFunc(ctx, input)
	}
	return nil, 0, nil
}

func TestService_ListCountries(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name     string
		repo     *mockLocationsRepository
		wantData []Country
		wantErr  bool
	}{
		{
			name: "returns data",
			repo: &mockLocationsRepository{
				listCountriesFunc: func(ctx context.Context) ([]Country, error) {
					return []Country{{CountryID: 1, Iso2Code: "US", Name: "United States"}}, nil
				},
			},
			wantData: []Country{{CountryID: 1, Iso2Code: "US", Name: "United States"}},
		},
		{
			name: "repository returns nil slice converts to empty slice",
			repo: &mockLocationsRepository{
				listCountriesFunc: func(ctx context.Context) ([]Country, error) {
					return nil, nil
				},
			},
			wantData: []Country{},
		},
		{
			name: "repository returns empty slice",
			repo: &mockLocationsRepository{
				listCountriesFunc: func(ctx context.Context) ([]Country, error) {
					return []Country{}, nil
				},
			},
			wantData: []Country{},
		},
		{
			name: "repository error is wrapped",
			repo: &mockLocationsRepository{
				listCountriesFunc: func(ctx context.Context) ([]Country, error) {
					return nil, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			got, err := svc.ListCountries(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
		})
	}
}

func TestService_ListStates(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name     string
		input    ListStatesInput
		repo     *mockLocationsRepository
		wantData []State
		wantErr  bool
	}{
		{
			name:  "returns data with non-nil iso_code",
			input: ListStatesInput{CountryID: 1},
			repo: &mockLocationsRepository{
				listStatesFunc: func(ctx context.Context, countryID int32) ([]State, error) {
					code := "CA"
					return []State{{StateID: 1, IsoCode: &code, Name: "California"}}, nil
				},
			},
			wantData: func() []State {
				code := "CA"
				return []State{{StateID: 1, IsoCode: &code, Name: "California"}}
			}(),
		},
		{
			name:  "returns data with nil iso_code",
			input: ListStatesInput{CountryID: 1},
			repo: &mockLocationsRepository{
				listStatesFunc: func(ctx context.Context, countryID int32) ([]State, error) {
					return []State{{StateID: 2, IsoCode: nil, Name: "State Without Code"}}, nil
				},
			},
			wantData: []State{{StateID: 2, IsoCode: nil, Name: "State Without Code"}},
		},
		{
			name:  "repository returns nil slice converts to empty slice",
			input: ListStatesInput{CountryID: 1},
			repo: &mockLocationsRepository{
				listStatesFunc: func(ctx context.Context, countryID int32) ([]State, error) {
					return nil, nil
				},
			},
			wantData: []State{},
		},
		{
			name:  "repository error is wrapped",
			input: ListStatesInput{CountryID: 1},
			repo: &mockLocationsRepository{
				listStatesFunc: func(ctx context.Context, countryID int32) ([]State, error) {
					return nil, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			got, err := svc.ListStates(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
		})
	}
}

func TestService_ListCities(t *testing.T) {
	repositoryErr := errors.New("repo fail")

	tests := []struct {
		name      string
		input     ListCitiesInput
		repo      *mockLocationsRepository
		wantData  []City
		wantMeta  ListCitiesMeta
		wantErr   bool
		wantInput ListCitiesInput
	}{
		{
			name:  "returns data with correct meta",
			input: ListCitiesInput{StateID: 1, Page: 2, PageSize: 50},
			repo: &mockLocationsRepository{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
					return []City{{CityID: 1, Name: "Los Angeles"}}, 150, nil
				},
			},
			wantData:  []City{{CityID: 1, Name: "Los Angeles"}},
			wantMeta:  ListCitiesMeta{Total: 150, Page: 2, PageSize: 50, TotalPages: 3},
			wantInput: ListCitiesInput{StateID: 1, Page: 2, PageSize: 50},
		},
		{
			name:  "empty result returns empty slice and zero meta",
			input: ListCitiesInput{StateID: 1, Page: 1, PageSize: 50},
			repo: &mockLocationsRepository{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
					return []City{}, 0, nil
				},
			},
			wantData:  []City{},
			wantMeta:  ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0},
			wantInput: ListCitiesInput{StateID: 1, Page: 1, PageSize: 50},
		},
		{
			name:  "repository returns nil slice converts to empty slice",
			input: ListCitiesInput{StateID: 1, Page: 1, PageSize: 50},
			repo: &mockLocationsRepository{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
					return nil, 0, nil
				},
			},
			wantData:  []City{},
			wantMeta:  ListCitiesMeta{Total: 0, Page: 1, PageSize: 50, TotalPages: 0},
			wantInput: ListCitiesInput{StateID: 1, Page: 1, PageSize: 50},
		},
		{
			name:  "repository error is wrapped",
			input: ListCitiesInput{StateID: 1, Page: 1, PageSize: 50},
			repo: &mockLocationsRepository{
				listCitiesFunc: func(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
					return nil, 0, repositoryErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledInput ListCitiesInput
			repo := tt.repo
			orig := repo.listCitiesFunc
			repo.listCitiesFunc = func(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
				calledInput = input
				return orig(ctx, input)
			}

			svc := NewService(repo)
			got, err := svc.ListCities(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Data == nil {
				t.Fatal("expected non-nil data slice")
			}
			if !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", got.Data, tt.wantData)
			}
			if !reflect.DeepEqual(got.Meta, tt.wantMeta) {
				t.Fatalf("meta mismatch: got %#v want %#v", got.Meta, tt.wantMeta)
			}
			if !reflect.DeepEqual(calledInput, tt.wantInput) {
				t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
			}
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	if got := calculateTotalPages(0, 50); got != 0 {
		t.Fatalf("calculateTotalPages(0,50) = %d, want 0", got)
	}
	if got := calculateTotalPages(50, 50); got != 1 {
		t.Fatalf("calculateTotalPages(50,50) = %d, want 1", got)
	}
	if got := calculateTotalPages(51, 50); got != 2 {
		t.Fatalf("calculateTotalPages(51,50) = %d, want 2", got)
	}
	if got := calculateTotalPages(1, 50); got != 1 {
		t.Fatalf("calculateTotalPages(1,50) = %d, want 1", got)
	}
	if got := calculateTotalPages(100, 50); got != 2 {
		t.Fatalf("calculateTotalPages(100,50) = %d, want 2", got)
	}
	if got := calculateTotalPages(101, 50); got != 3 {
		t.Fatalf("calculateTotalPages(101,50) = %d, want 3", got)
	}
	if got := calculateTotalPages(10, 0); got != 0 {
		t.Fatalf("calculateTotalPages(10,0) = %d, want 0", got)
	}
}
