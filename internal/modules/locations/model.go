package locations

import "context"

type ListStatesInput struct {
	CountryID int32
}

type ListCitiesInput struct {
	StateID  int32
	Page     int32
	PageSize int32
}

type Country struct {
	CountryID int32  `json:"country_id"`
	Iso2Code  string `json:"iso2_code"`
	Name      string `json:"name"`
}

type State struct {
	StateID int32   `json:"state_id"`
	IsoCode *string `json:"iso_code"`
	Name    string  `json:"name"`
}

type City struct {
	CityID int32  `json:"city_id"`
	Name   string `json:"name"`
}

type ListCountriesResult struct {
	Data []Country `json:"data"`
}

type ListStatesResult struct {
	Data []State `json:"data"`
}

type ListCitiesResult struct {
	Data []City         `json:"data"`
	Meta ListCitiesMeta `json:"meta"`
}

type ListCitiesMeta struct {
	Total      int64 `json:"total"`
	Page       int32 `json:"page"`
	PageSize   int32 `json:"page_size"`
	TotalPages int32 `json:"total_pages"`
}

type LocationsRepository interface {
	ListCountries(ctx context.Context) ([]Country, error)
	ListStates(ctx context.Context, countryID int32) ([]State, error)
	ListCities(ctx context.Context, input ListCitiesInput) ([]City, int64, error)
}

type LocationsService interface {
	ListCountries(ctx context.Context) (ListCountriesResult, error)
	ListStates(ctx context.Context, input ListStatesInput) (ListStatesResult, error)
	ListCities(ctx context.Context, input ListCitiesInput) (ListCitiesResult, error)
}
