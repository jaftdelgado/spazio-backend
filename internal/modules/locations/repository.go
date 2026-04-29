package locations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	db      *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) LocationsRepository {
	return &repository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *repository) ListCountries(ctx context.Context) ([]Country, error) {
	rows, err := r.queries.ListCountries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list countries: %w", err)
	}

	countries := make([]Country, 0, len(rows))
	for _, row := range rows {
		countries = append(countries, Country{
			CountryID: row.CountryID,
			Iso2Code:  row.Iso2Code,
			Name:      row.Name,
		})
	}

	return countries, nil
}

func (r *repository) ListStates(ctx context.Context, countryID int32) ([]State, error) {
	rows, err := r.queries.ListStates(ctx, countryID)
	if err != nil {
		return nil, fmt.Errorf("list states: %w", err)
	}

	states := make([]State, 0, len(rows))
	for _, row := range rows {
		var isoCode *string
		if row.IsoCode.Valid {
			isoCode = &row.IsoCode.String
		}

		states = append(states, State{
			StateID: row.StateID,
			IsoCode: isoCode,
			Name:    row.Name,
		})
	}

	return states, nil
}

func (r *repository) ListCities(ctx context.Context, input ListCitiesInput) ([]City, int64, error) {
	offset := (input.Page - 1) * input.PageSize

	rows, err := r.queries.ListCities(ctx, sqlcgen.ListCitiesParams{
		StateID: input.StateID,
		Limit:   input.PageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list cities: %w", err)
	}

	cities := make([]City, 0, len(rows))
	for _, row := range rows {
		cities = append(cities, City{
			CityID: row.CityID,
			Name:   row.Name,
		})
	}

	if len(rows) == 0 {
		return cities, 0, nil
	}

	return cities, rows[0].TotalCount, nil
}
