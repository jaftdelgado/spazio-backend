package properties

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	db      *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository builds a property repository implementation.
func NewRepository(db *pgxpool.Pool) PropertyRepository {
	return &repository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *repository) GetModalityName(ctx context.Context, modalityID int32) (string, error) {
	name, err := r.queries.GetModalityName(ctx, modalityID)
	if err != nil {
		return "", fmt.Errorf("get modality name: %w", err)
	}

	return name, nil
}

func (r *repository) GetAllowedPeriods(ctx context.Context, propertyTypeID int32) (map[int32]struct{}, error) {
	rows, err := r.queries.GetAllowedPeriods(ctx, propertyTypeID)
	if err != nil {
		return nil, fmt.Errorf("get allowed periods: %w", err)
	}

	allowedPeriods := make(map[int32]struct{}, len(rows))
	for _, periodID := range rows {
		allowedPeriods[periodID] = struct{}{}
	}

	return allowedPeriods, nil
}

func (r *repository) GetClauseValueTypes(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
	rows, err := r.queries.GetClauseValueTypes(ctx, clauseIDs)
	if err != nil {
		return nil, fmt.Errorf("get clause value types: %w", err)
	}

	valueTypes := make(map[int32]int32, len(rows))
	for _, row := range rows {
		valueTypes[row.ClauseID] = row.ValueTypeID
	}

	return valueTypes, nil
}

func (r *repository) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := sqlcgen.New(tx)

	propertyUUID := uuid.New()
	pgUUID := pgtype.UUID{Bytes: propertyUUID, Valid: true}

	lotArea, err := numericFromFloat64(input.LotArea)
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("convert lot area: %w", err)
	}

	propertyRow, err := queries.CreateProperty(ctx, sqlcgen.CreatePropertyParams{
		PropertyUuid:   pgUUID,
		OwnerID:        input.OwnerID,
		Category:       input.Category,
		Title:          input.Title,
		Description:    input.Description,
		PropertyTypeID: input.PropertyTypeID,
		ModalityID:     input.ModalityID,
		LotArea:        lotArea,
		IsFeatured:     input.IsFeatured,
	})
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("create property: %w", err)
	}

	if err := r.createSubtype(ctx, queries, propertyRow.PropertyID, input); err != nil {
		return CreatePropertyResult{}, err
	}

	if err := r.createLocation(ctx, queries, propertyRow.PropertyID, input.Location); err != nil {
		return CreatePropertyResult{}, err
	}

	if input.SalePrice != nil {
		if err := r.createSalePrice(ctx, queries, propertyRow.PropertyID, input.OwnerID, input.SalePrice); err != nil {
			return CreatePropertyResult{}, err
		}
	}

	for _, rentPrice := range input.RentPrices {
		if err := r.createRentPrice(ctx, queries, propertyRow.PropertyID, input.OwnerID, rentPrice); err != nil {
			return CreatePropertyResult{}, err
		}
	}

	for _, serviceID := range input.Services {
		if err := queries.CreatePropertyService(ctx, sqlcgen.CreatePropertyServiceParams{
			PropertyID: propertyRow.PropertyID,
			ServiceID:  serviceID,
		}); err != nil {
			return CreatePropertyResult{}, fmt.Errorf("create property service %d: %w", serviceID, err)
		}
	}

	for _, clause := range input.Clauses {
		if err := r.createPropertyClause(ctx, queries, propertyRow.PropertyID, clause); err != nil {
			return CreatePropertyResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CreatePropertyResult{}, fmt.Errorf("commit transaction: %w", err)
	}

	return CreatePropertyResult{
		Data: CreatePropertyResultData{
			PropertyUUID: propertyRow.PropertyUuid.String(),
		},
	}, nil
}

func (r *repository) createSubtype(ctx context.Context, queries *sqlcgen.Queries, propertyID int32, input CreatePropertyInput) error {
	switch input.Category {
	case CategoryResidential:
		builtArea, err := numericFromFloat64(*input.Residential.BuiltArea)
		if err != nil {
			return fmt.Errorf("convert residential built area: %w", err)
		}

		if err := queries.CreateResidentialProperty(ctx, sqlcgen.CreateResidentialPropertyParams{
			PropertyID:       propertyID,
			Bedrooms:         *input.Residential.Bedrooms,
			Bathrooms:        *input.Residential.Bathrooms,
			Beds:             *input.Residential.Beds,
			Floors:           *input.Residential.Floors,
			ParkingSpots:     *input.Residential.ParkingSpots,
			BuiltArea:        builtArea,
			ConstructionYear: *input.Residential.ConstructionYear,
			OrientationID:    *input.Residential.OrientationID,
			IsFurnished:      *input.Residential.IsFurnished,
		}); err != nil {
			return fmt.Errorf("create residential property: %w", err)
		}
	case CategoryCommercial:
		ceilingHeight, err := numericFromFloat64(*input.Commercial.CeilingHeight)
		if err != nil {
			return fmt.Errorf("convert commercial ceiling height: %w", err)
		}

		if err := queries.CreateCommercialProperty(ctx, sqlcgen.CreateCommercialPropertyParams{
			PropertyID:      propertyID,
			CeilingHeight:   ceilingHeight,
			LoadingDocks:    *input.Commercial.LoadingDocks,
			InternalOffices: *input.Commercial.InternalOffices,
			ThreePhasePower: *input.Commercial.ThreePhasePower,
			LandUse:         *input.Commercial.LandUse,
		}); err != nil {
			return fmt.Errorf("create commercial property: %w", err)
		}
	}

	return nil
}

func (r *repository) createLocation(ctx context.Context, queries *sqlcgen.Queries, propertyID int32, location *CreateLocationInput) error {
	if err := queries.CreateLocation(ctx, sqlcgen.CreateLocationParams{
		PropertyID:      propertyID,
		CityID:          location.CityID,
		Neighborhood:    location.Neighborhood,
		Street:          location.Street,
		ExteriorNumber:  location.ExteriorNumber,
		InteriorNumber:  textFromPointer(location.InteriorNumber),
		PostalCode:      location.PostalCode,
		StMakepoint:     *location.Longitude,
		StMakepoint_2:   *location.Latitude,
		IsPublicAddress: *location.IsPublicAddress,
	}); err != nil {
		return fmt.Errorf("create location: %w", err)
	}

	return nil
}

func (r *repository) createSalePrice(ctx context.Context, queries *sqlcgen.Queries, propertyID, ownerID int32, salePrice *CreateSalePriceInput) error {
	amount, err := numericFromFloat64(*salePrice.SalePrice)
	if err != nil {
		return fmt.Errorf("convert sale price: %w", err)
	}

	if err := queries.CreateSalePrice(ctx, sqlcgen.CreateSalePriceParams{
		PropertyID:      propertyID,
		SalePrice:       amount,
		Currency:        salePrice.Currency,
		IsNegotiable:    *salePrice.IsNegotiable,
		ChangedByUserID: ownerID,
	}); err != nil {
		return fmt.Errorf("create sale price: %w", err)
	}

	return nil
}

func (r *repository) createRentPrice(ctx context.Context, queries *sqlcgen.Queries, propertyID, ownerID int32, rentPrice CreateRentPriceInput) error {
	amount, err := numericFromFloat64(*rentPrice.RentPrice)
	if err != nil {
		return fmt.Errorf("convert rent price: %w", err)
	}

	deposit, err := numericFromPointer(rentPrice.Deposit)
	if err != nil {
		return fmt.Errorf("convert deposit: %w", err)
	}

	if err := queries.CreateRentPrice(ctx, sqlcgen.CreateRentPriceParams{
		PropertyID:      propertyID,
		PeriodID:        rentPrice.PeriodID,
		RentPrice:       amount,
		Deposit:         deposit,
		Currency:        rentPrice.Currency,
		IsNegotiable:    *rentPrice.IsNegotiable,
		ChangedByUserID: ownerID,
	}); err != nil {
		return fmt.Errorf("create rent price for period_id %d: %w", rentPrice.PeriodID, err)
	}

	return nil
}

func (r *repository) createPropertyClause(ctx context.Context, queries *sqlcgen.Queries, propertyID int32, clause CreatePropertyClauseInput) error {
	minValue, err := numericFromPointer(clause.MinValue)
	if err != nil {
		return fmt.Errorf("convert clause min value: %w", err)
	}

	maxValue, err := numericFromPointer(clause.MaxValue)
	if err != nil {
		return fmt.Errorf("convert clause max value: %w", err)
	}

	if err := queries.CreatePropertyClause(ctx, sqlcgen.CreatePropertyClauseParams{
		PropertyID:   propertyID,
		ClauseID:     clause.ClauseID,
		BooleanValue: boolFromPointer(clause.BooleanValue),
		IntegerValue: int4FromPointer(clause.IntegerValue),
		MinValue:     minValue,
		MaxValue:     maxValue,
	}); err != nil {
		return fmt.Errorf("create property clause %d: %w", clause.ClauseID, err)
	}

	return nil
}

func numericFromFloat64(value float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}

	return numeric, nil
}

func numericFromPointer(value *float64) (pgtype.Numeric, error) {
	if value == nil {
		return pgtype.Numeric{}, nil
	}

	return numericFromFloat64(*value)
}

func textFromPointer(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func boolFromPointer(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}

	return pgtype.Bool{Bool: *value, Valid: true}
}

func int4FromPointer(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: *value, Valid: true}
}
