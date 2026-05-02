package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *repository) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	// Resolve property id first
	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyResult{}, err
	}

	// Get base
	baseRow, err := r.queries.GetPropertyBaseByID(ctx, propertyID)
	if err != nil {
		if errorsIsPgxNoRows(err) {
			return GetPropertyResult{}, ErrPropertyNotFound
		}
		return GetPropertyResult{}, fmt.Errorf("get property base: %w", err)
	}

	// Map base
	lotValue, err := baseRow.LotArea.Float64Value()
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("convert lot area: %w", err)
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

	// Residential
	if baseRow.Subtype == SubtypeResidential {
		if resRow, err := r.queries.GetResidentialByPropertyID(ctx, propertyID); err == nil {
			builtValue, err := resRow.BuiltArea.Float64Value()
			if err != nil {
				return GetPropertyResult{}, fmt.Errorf("convert built area: %w", err)
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
				return GetPropertyResult{}, fmt.Errorf("get residential: %w", err)
			}
		}
	}

	// Commercial
	if baseRow.Subtype == SubtypeCommercial {
		if comRow, err := r.queries.GetCommercialByPropertyID(ctx, propertyID); err == nil {
			chValue, err := comRow.CeilingHeight.Float64Value()
			if err != nil {
				return GetPropertyResult{}, fmt.Errorf("convert ceiling height: %w", err)
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
				return GetPropertyResult{}, fmt.Errorf("get commercial: %w", err)
			}
		}
	}

	// Location
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
			return GetPropertyResult{}, fmt.Errorf("get location: %w", err)
		}
	}

	return GetPropertyResult{Data: data}, nil
}

func errorsIsPgxNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
