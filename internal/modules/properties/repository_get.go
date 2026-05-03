package properties

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) ListProperties(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
	rows, err := r.queries.ListPropertiesCards(ctx, sqlcgen.ListPropertiesCardsParams{
		SearchQuery:    input.Query,
		StatusIds:      input.StatusIDs,
		PropertyTypeID: input.PropertyTypeID,
		ModalityID:     input.ModalityID,
		CountryID:      input.CountryID,
		StateID:        input.StateID,
		CityID:         input.CityID,
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

func (r *repository) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyResult{}, err
	}

	data, err := r.getPropertyDataByID(ctx, propertyID)
	if err != nil {
		return GetPropertyResult{}, err
	}

	return GetPropertyResult{Data: data}, nil
}

func (r *repository) GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyFullResult{}, err
	}

	baseData, err := r.getPropertyDataByID(ctx, propertyID)
	if err != nil {
		return GetPropertyFullResult{}, err
	}

	priceRows, err := r.queries.ListPropertyPriceTimeline(ctx, propertyID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("list property price timeline: %w", err)
	}

	photosRows, err := r.queries.ListPropertyPhotos(ctx, propertyID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("list property photos: %w", err)
	}

	serviceRows, err := r.queries.ListPropertyServiceIDs(ctx, propertyID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("list property services: %w", err)
	}

	clauseRows, err := r.queries.ListPropertyClauses(ctx, propertyID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("list property clauses: %w", err)
	}

	prices, err := buildFullPropertyPrices(baseData.ModalityID, priceRows)
	if err != nil {
		return GetPropertyFullResult{}, err
	}

	photos := make([]PropertyPhotoData, 0, len(photosRows))
	for _, row := range photosRows {
		photos = append(photos, propertyPhotoDataFromRow(row))
	}

	services := make([]int32, 0, len(serviceRows))
	for _, serviceID := range serviceRows {
		services = append(services, serviceID)
	}

	clauses := make([]PropertyClauseData, 0, len(clauseRows))
	for _, row := range clauseRows {
		clause, err := propertyClauseDataFromRow(row)
		if err != nil {
			return GetPropertyFullResult{}, err
		}
		clauses = append(clauses, clause)
	}

	return GetPropertyFullResult{
		Data: GetPropertyFullData{
			PropertyUUID:   baseData.PropertyUUID,
			OwnerID:        baseData.OwnerID,
			Subtype:        baseData.Subtype,
			Title:          baseData.Title,
			Description:    baseData.Description,
			PropertyTypeID: baseData.PropertyTypeID,
			ModalityID:     baseData.ModalityID,
			LotArea:        baseData.LotArea,
			IsFeatured:     baseData.IsFeatured,
			Residential:    baseData.Residential,
			Commercial:     baseData.Commercial,
			Location:       baseData.Location,
			Prices:         prices,
			Photos:         photos,
			Services:       services,
			Clauses:        clauses,
		},
	}, nil
}

func (r *repository) getPropertyDataByID(ctx context.Context, propertyID int32) (GetPropertyData, error) {
	baseRow, err := r.queries.GetPropertyBaseByID(ctx, propertyID)
	if err != nil {
		if errorsIsPgxNoRows(err) {
			return GetPropertyData{}, ErrPropertyNotFound
		}
		return GetPropertyData{}, fmt.Errorf("get property base: %w", err)
	}

	lotValue, err := baseRow.LotArea.Float64Value()
	if err != nil {
		return GetPropertyData{}, fmt.Errorf("convert lot area: %w", err)
	}

	data := GetPropertyData{
		PropertyUUID:   baseRow.PropertyUuid.String(),
		OwnerID:        baseRow.OwnerID,
		Subtype:        baseRow.Subtype,
		Title:          baseRow.Title,
		Description:    baseRow.Description,
		PropertyTypeID: baseRow.PropertyTypeID,
		ModalityID:     baseRow.ModalityID,
		LotArea:        lotValue.Float64,
		IsFeatured:     baseRow.IsFeatured,
	}

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
		} else {
			if !errorsIsPgxNoRows(err) {
				return GetPropertyData{}, fmt.Errorf("get residential: %w", err)
			}
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
		} else {
			if !errorsIsPgxNoRows(err) {
				return GetPropertyData{}, fmt.Errorf("get commercial: %w", err)
			}
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
	} else {
		if !errorsIsPgxNoRows(err) {
			return GetPropertyData{}, fmt.Errorf("get location: %w", err)
		}
	}

	return data, nil
}

func resolvePageOffset(page, pageSize int32) int32 {
	return (page - 1) * pageSize
}

func propertyCardDataFromRow(row sqlcgen.ListPropertiesCardsRow) (PropertyCardData, error) {
	card := PropertyCardData{
		PropertyUUID:  row.PropertyUuid.String(),
		Title:         row.Title,
		CoverPhotoURL: stringPointerFromText(row.CoverPhotoUrl),
		PropertyType: PropertyCardTypeData{
			PropertyTypeID: row.PropertyTypeID,
			Name:           row.PropertyTypeName,
			Icon:           stringPointerFromText(row.PropertyTypeIcon),
		},
		Modality: PropertyCardModalityData{
			ModalityID: row.ModalityID,
			Name:       row.ModalityName,
		},
		Status: PropertyCardStatusData{
			StatusID: row.StatusID,
			Name:     row.StatusName,
		},
	}

	if row.DisplayPriceAmount.Valid {
		amount, err := row.DisplayPriceAmount.Float64Value()
		if err != nil {
			return PropertyCardData{}, fmt.Errorf("convert display price amount: %w", err)
		}
		if amount.Valid {
			var periodName *string
			if row.DisplayPeriodName != "" {
				periodName = &row.DisplayPeriodName
			}

			card.Price = &PropertyCardPriceData{
				Amount:     amount.Float64,
				Currency:   row.DisplayPriceCurrency,
				PriceType:  row.DisplayPriceType,
				PeriodName: periodName,
			}
		}
	}

	return card, nil
}

func buildFullPropertyPrices(modalityID int32, rows []sqlcgen.ListPropertyPriceTimelineRow) (PropertyFullPricesData, error) {
	result := PropertyFullPricesData{
		Current: PropertyCurrentPricesData{
			Rent: make([]CurrentRentPriceDetailData, 0),
		},
		History: make([]PropertyPriceHistoryData, 0, len(rows)),
	}

	for _, row := range rows {
		historyItem, err := propertyPriceHistoryDataFromRow(row)
		if err != nil {
			return PropertyFullPricesData{}, err
		}
		result.History = append(result.History, historyItem)

		if !row.IsCurrent {
			continue
		}

		switch row.PriceType {
		case "sale":
			if modalityID == ModalityRent {
				continue
			}

			result.Current.Sale = &CurrentSalePriceDetailData{
				Amount:       historyItem.Amount,
				Currency:     historyItem.Currency,
				IsNegotiable: historyItem.IsNegotiable,
				ValidFrom:    historyItem.ValidFrom,
				ValidUntil:   historyItem.ValidUntil,
				IsCurrent:    historyItem.IsCurrent,
			}
		case "rent":
			if modalityID == ModalitySale {
				continue
			}

			result.Current.Rent = append(result.Current.Rent, CurrentRentPriceDetailData{
				Amount:       historyItem.Amount,
				Currency:     historyItem.Currency,
				PeriodName:   historyItem.PeriodName,
				IsNegotiable: historyItem.IsNegotiable,
				Deposit:      historyItem.Deposit,
				ValidFrom:    historyItem.ValidFrom,
				ValidUntil:   historyItem.ValidUntil,
				IsCurrent:    historyItem.IsCurrent,
			})
		}
	}

	sort.Slice(result.Current.Rent, func(i, j int) bool {
		return periodNamePriority(result.Current.Rent[i].PeriodName) < periodNamePriority(result.Current.Rent[j].PeriodName)
	})

	return result, nil
}

func propertyPriceHistoryDataFromRow(row sqlcgen.ListPropertyPriceTimelineRow) (PropertyPriceHistoryData, error) {
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

func timePointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	timeValue := value.Time
	return &timeValue
}

func periodNamePriority(periodName *string) int {
	if periodName == nil {
		return 100
	}

	normalized := strings.ToLower(strings.TrimSpace(*periodName))
	if normalized == "" {
		return 100
	}

	words := strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	switch {
	case hasWordWithPrefix(words, "month"), hasWordWithPrefix(words, "mens"), hasExactWord(words, "mes"):
		return 1
	case hasExactWord(words, "year"), hasExactWord(words, "año"), hasExactWord(words, "anual"):
		return 2
	case hasWordWithPrefix(words, "week"), hasWordWithPrefix(words, "seman"):
		return 3
	case hasWordWithPrefix(words, "day"), hasWordWithPrefix(words, "dia"):
		return 4
	default:
		return 100
	}
}

func hasExactWord(words []string, target string) bool {
	for _, word := range words {
		if word == target {
			return true
		}
	}

	return false
}

func hasWordWithPrefix(words []string, prefix string) bool {
	for _, word := range words {
		if strings.HasPrefix(word, prefix) {
			return true
		}
	}

	return false
}

func errorsIsPgxNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
