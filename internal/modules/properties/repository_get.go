package properties

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) ListPropertyStatusHistory(ctx context.Context, propertyUUID string) ([]PropertyStatusHistoryData, error) {
	var pgUUID pgtype.UUID
	if err := pgUUID.Scan(propertyUUID); err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	rows, err := r.queries.ListPropertyStatusHistory(ctx, pgUUID)
	if err != nil {
		return nil, fmt.Errorf("list property status history: %w", err)
	}

	history := make([]PropertyStatusHistoryData, 0, len(rows))
	for _, row := range rows {
		history = append(history, PropertyStatusHistoryData{
			HistoryID:          row.HistoryID,
			PropertyUUID:       propertyUUID,
			PreviousStatusName: row.PreviousStatusName,
			NewStatusName:      row.NewStatusName,
			ChangedByName:      stringValueFromUnknown(row.ChangedByName),
			ChangedAt:          row.ChangedAt.Time,
		})
	}

	return history, nil
}

func (r *repository) GetPropertyOwnerByUUID(ctx context.Context, propertyUUID string) (int32, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return 0, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return 0, err
	}

	ownerID, err := r.queries.GetPropertyOwnerID(ctx, propertyID)
	if err != nil {
		return 0, fmt.Errorf("get property owner: %w", err)
	}

	return ownerID, nil
}

func (r *repository) ListProperties(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
	statusIDs := input.StatusIDs
	if statusIDs == nil {
		statusIDs = []int32{}
	}

	rows, err := r.queries.ListPropertiesCards(ctx, sqlcgen.ListPropertiesCardsParams{
		SearchQuery:    input.Query,
		StatusIds:      statusIDs,
		PropertyTypeID: input.PropertyTypeID,
		ModalityID:     input.ModalityID,
		CountryID:      input.CountryID,
		StateID:        input.StateID,
		CityID:         input.CityID,
		MinPrice:       float64ToNumeric(input.MinPrice),
		MaxPrice:       float64ToNumeric(input.MaxPrice),
		MinBedrooms:    input.MinBedrooms,
		SortField:      input.Sort,
		SortOrder:      input.Order,
		PageOffset:     resolvePageOffset(input.Page, input.PageSize),
		PageSize:       input.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list properties: %w", err)
	}

	properties := make([]PropertyCardData, 0, len(rows))
	for _, row := range rows {
		property, err := propertyCardDataFromRow(row)
		if err != nil {
			return nil, 0, err
		}
		properties = append(properties, property)
	}

	if len(rows) == 0 {
		return properties, 0, nil
	}

	return properties, rows[0].TotalCount, nil
}

func (r *repository) ListPropertiesForAgent(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
	statusIDs := input.StatusIDs
	if statusIDs == nil {
		statusIDs = []int32{}
	}

	rows, err := r.queries.ListPropertiesCardsForAgent(ctx, sqlcgen.ListPropertiesCardsForAgentParams{
		AgentID:        input.UserID,
		SearchQuery:    input.Query,
		StatusIds:      statusIDs,
		PropertyTypeID: input.PropertyTypeID,
		ModalityID:     input.ModalityID,
		CountryID:      input.CountryID,
		StateID:        input.StateID,
		CityID:         input.CityID,
		MinPrice:       float64ToNumeric(input.MinPrice),
		MaxPrice:       float64ToNumeric(input.MaxPrice),
		MinBedrooms:    input.MinBedrooms,
		SortField:      input.Sort,
		SortOrder:      input.Order,
		PageOffset:     resolvePageOffset(input.Page, input.PageSize),
		PageSize:       input.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list properties for agent: %w", err)
	}

	properties := make([]PropertyCardData, 0, len(rows))
	for _, row := range rows {
		property, err := propertyCardDataFromAgentRow(row)
		if err != nil {
			return nil, 0, err
		}
		properties = append(properties, property)
	}

	if len(rows) == 0 {
		return properties, 0, nil
	}

	return properties, rows[0].TotalCount, nil
}

func float64ToNumeric(val float64) pgtype.Numeric {
	if val == 0 {
		return pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true}
	}
	return pgtype.Numeric{Int: big.NewInt(int64(val * 100)), Exp: -2, Valid: true}
}

func (r *repository) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	return r.GetPropertyByUUID(ctx, propertyUUID)
}

func (r *repository) GetPropertyByUUID(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	var pgUUID pgtype.UUID
	if err := pgUUID.Scan(propertyUUID); err != nil {
		return GetPropertyResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	baseRow, err := r.queries.GetPropertyBaseByUUID(ctx, pgUUID)
	if err != nil {
		if errorsIsPgxNoRows(err) {
			return GetPropertyResult{}, ErrPropertyNotFound
		}
		return GetPropertyResult{}, fmt.Errorf("get property base: %w", err)
	}

	data, err := r.getPropertyDataFromBaseRow(ctx, baseRow)
	if err != nil {
		return GetPropertyResult{}, err
	}

	return GetPropertyResult{Data: data}, nil
}

func (r *repository) IsPropertyAssignedToAgent(ctx context.Context, propertyID int32, agentID int32) (bool, error) {
	assigned, err := r.queries.IsPropertyAssignedToAgent(ctx, sqlcgen.IsPropertyAssignedToAgentParams{
		PropertyID: propertyID,
		AgentID:    agentID,
	})
	if err != nil {
		return false, fmt.Errorf("check property assignment: %w", err)
	}

	return assigned, nil
}

func (r *repository) GetPropertyPricesHistory(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error) {
	var pgUUID pgtype.UUID
	if err := pgUUID.Scan(propertyUUID); err != nil {
		return GetPropertyPricesHistoryResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	rows, err := r.queries.GetPropertyPricesHistory(ctx, pgUUID)
	if err != nil {
		return GetPropertyPricesHistoryResult{}, fmt.Errorf("get property prices history: %w", err)
	}

	items := make([]PropertyPriceHistoryData, 0, len(rows))
	for _, row := range rows {
		item, err := propertyPriceHistoryDataFromHistoryRow(row)
		if err != nil {
			return GetPropertyPricesHistoryResult{}, err
		}
		items = append(items, item)
	}

	return GetPropertyPricesHistoryResult{Data: items}, nil
}

func (r *repository) getPropertyDataFromBaseRow(ctx context.Context, baseRow sqlcgen.GetPropertyBaseByUUIDRow) (GetPropertyData, error) {
	lotValue, err := baseRow.LotArea.Float64Value()
	if err != nil {
		return GetPropertyData{}, fmt.Errorf("convert lot area: %w", err)
	}

	data := GetPropertyData{
		PropertyID:     baseRow.PropertyID,
		PropertyUUID:   baseRow.PropertyUuid.String(),
		OwnerID:        baseRow.OwnerID,
		Subtype:        baseRow.Subtype,
		Title:          baseRow.Title,
		Description:    baseRow.Description,
		PropertyTypeID: baseRow.PropertyTypeID,
		ModalityID:     baseRow.ModalityID,
		StatusID:       baseRow.StatusID,
		LotArea:        lotValue.Float64,
		IsFeatured:     baseRow.IsFeatured,
		RegisteredBy:   stringValueFromUnknown(baseRow.RegisteredBy),
	}

	propertyID := baseRow.PropertyID

	if baseRow.Subtype == SubtypeResidential {
		if resRow, err := r.queries.GetResidentialByPropertyID(ctx, propertyID); err == nil {
			builtValue, err := resRow.BuiltArea.Float64Value()
			if err != nil {
				return GetPropertyData{}, fmt.Errorf("convert built area: %w", err)
			}

			data.Residential = &ResidentialData{
				Bedrooms:         resRow.Bedrooms,
				Bathrooms:        resRow.Bathrooms,
				Beds:             resRow.Beds,
				Floors:           resRow.Floors,
				ParkingSpots:     resRow.ParkingSpots,
				BuiltArea:        builtValue.Float64,
				ConstructionYear: resRow.ConstructionYear,
				OrientationID:    resRow.OrientationID,
				IsFurnished:      resRow.IsFurnished,
			}
		} else if !errorsIsPgxNoRows(err) {
			return GetPropertyData{}, fmt.Errorf("get residential: %w", err)
		}
	}

	if baseRow.Subtype == SubtypeCommercial {
		if comRow, err := r.queries.GetCommercialByPropertyID(ctx, propertyID); err == nil {
			chValue, err := comRow.CeilingHeight.Float64Value()
			if err != nil {
				return GetPropertyData{}, fmt.Errorf("convert ceiling height: %w", err)
			}

			data.Commercial = &CommercialData{
				CeilingHeight:   chValue.Float64,
				LoadingDocks:    comRow.LoadingDocks,
				InternalOffices: comRow.InternalOffices,
				ThreePhasePower: comRow.ThreePhasePower,
				LandUse:         comRow.LandUse,
			}
		} else if !errorsIsPgxNoRows(err) {
			return GetPropertyData{}, fmt.Errorf("get commercial: %w", err)
		}
	}

	if locRow, err := r.queries.GetLocationByPropertyID(ctx, propertyID); err == nil {
		data.Location = &LocationData{
			CityID:          locRow.CityID,
			Neighborhood:    locRow.Neighborhood,
			Street:          locRow.Street,
			ExteriorNumber:  locRow.ExteriorNumber,
			InteriorNumber:  stringPointerFromText(locRow.InteriorNumber),
			PostalCode:      locRow.PostalCode,
			Latitude:        locRow.Latitude,
			Longitude:       locRow.Longitude,
			IsPublicAddress: locRow.IsPublicAddress,
		}
	} else if !errorsIsPgxNoRows(err) {
		return GetPropertyData{}, fmt.Errorf("get location: %w", err)
	}

	return data, nil
}

func resolvePageOffset(page, pageSize int32) int32 {
	return (page - 1) * pageSize
}

func propertyCardDataFromRow(row sqlcgen.ListPropertiesCardsRow) (PropertyCardData, error) {
	return propertyCardDataFromValues(
		row.PropertyUuid,
		row.Title,
		row.CoverPhotoUrl,
		row.PropertyTypeID,
		row.PropertyTypeName,
		row.PropertyTypeIcon,
		row.ModalityID,
		row.ModalityName,
		row.StatusID,
		row.StatusName,
		row.DisplayPriceAmount,
		row.DisplayPriceCurrency,
		row.DisplayPriceType,
		row.DisplayPeriodName,
		row.Bedrooms,
		row.Bathrooms,
		row.BuiltArea,
	)
}

func propertyCardDataFromAgentRow(row sqlcgen.ListPropertiesCardsForAgentRow) (PropertyCardData, error) {
	return propertyCardDataFromValues(
		row.PropertyUuid,
		row.Title,
		row.CoverPhotoUrl,
		row.PropertyTypeID,
		row.PropertyTypeName,
		row.PropertyTypeIcon,
		row.ModalityID,
		row.ModalityName,
		row.StatusID,
		row.StatusName,
		row.DisplayPriceAmount,
		row.DisplayPriceCurrency,
		row.DisplayPriceType,
		row.DisplayPeriodName,
		row.Bedrooms,
		row.Bathrooms,
		row.BuiltArea,
	)
}

func propertyCardDataFromValues(
	propertyUUID pgtype.UUID,
	title string,
	coverPhotoURL pgtype.Text,
	propertyTypeID int32,
	propertyTypeName string,
	propertyTypeIcon pgtype.Text,
	modalityID int32,
	modalityName string,
	statusID int32,
	statusName string,
	displayPriceAmount pgtype.Numeric,
	displayPriceCurrency string,
	displayPriceType string,
	displayPeriodName string,
	bedrooms pgtype.Int2,
	bathrooms pgtype.Int2,
	builtArea pgtype.Numeric,
) (PropertyCardData, error) {
	card := PropertyCardData{
		PropertyUUID:  propertyUUID.String(),
		Title:         title,
		CoverPhotoURL: stringPointerFromText(coverPhotoURL),
		PropertyType: PropertyCardTypeData{
			PropertyTypeID: propertyTypeID,
			Name:           propertyTypeName,
			Icon:           stringPointerFromText(propertyTypeIcon),
		},
		Modality: PropertyCardModalityData{
			ModalityID: modalityID,
			Name:       modalityName,
		},
		Status: PropertyCardStatusData{
			StatusID: statusID,
			Name:     statusName,
		},
	}

	if bedrooms.Valid {
		v := bedrooms.Int16
		card.Bedrooms = &v
	}
	if bathrooms.Valid {
		v := bathrooms.Int16
		card.Bathrooms = &v
	}
	if builtArea.Valid {
		v, _ := builtArea.Float64Value()
		card.BuiltArea = &v.Float64
	}

	if displayPriceAmount.Valid {
		amount, err := displayPriceAmount.Float64Value()
		if err != nil {
			return PropertyCardData{}, fmt.Errorf("convert display price amount: %w", err)
		}
		if amount.Valid {
			var periodName *string
			if displayPeriodName != "" {
				periodName = &displayPeriodName
			}

			card.Price = &PropertyCardPriceData{
				Amount:     amount.Float64,
				Currency:   displayPriceCurrency,
				PriceType:  displayPriceType,
				PeriodName: periodName,
			}
		}
	}

	return card, nil
}

func propertyPriceHistoryDataFromHistoryRow(row sqlcgen.GetPropertyPricesHistoryRow) (PropertyPriceHistoryData, error) {
	amount, err := row.Amount.Float64Value()
	if err != nil {
		return PropertyPriceHistoryData{}, fmt.Errorf("convert property price amount: %w", err)
	}
	if !amount.Valid {
		return PropertyPriceHistoryData{}, fmt.Errorf("property price amount is invalid")
	}

	data := PropertyPriceHistoryData{
		PriceType:    row.PriceType,
		Amount:       amount.Float64,
		Currency:     row.Currency,
		PeriodName:   stringPointerFromText(row.PeriodName),
		IsNegotiable: row.IsNegotiable,
		ValidFrom:    row.ValidFrom.Time,
		ValidUntil:   timePointerFromTimestamptz(row.ValidUntil),
		IsCurrent:    row.IsCurrent,
	}

	if row.Deposit.Valid {
		deposit, err := row.Deposit.Float64Value()
		if err != nil {
			return PropertyPriceHistoryData{}, fmt.Errorf("convert property price deposit: %w", err)
		}
		if deposit.Valid {
			value := deposit.Float64
			data.Deposit = &value
		}
	}

	return data, nil
}

func stringValueFromUnknown(value interface{}) string {
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return s
	}

	return ""
}

func timePointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	timeValue := value.Time
	return &timeValue
}

func errorsIsPgxNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
