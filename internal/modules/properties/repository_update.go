package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) UpdateProperty(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := sqlcgen.New(tx)

	propertyID, err := getPropertyIDByUUID(ctx, queries, parsedUUID)
	if err != nil {
		return UpdatePropertyResult{}, err
	}

	// Read current state
	baseRow, err := queries.GetPropertyBaseByUUID(ctx, pgtype.UUID{Bytes: parsedUUID, Valid: true})
	if err != nil {
		if errorsIsPgxNoRows(err) {
			return UpdatePropertyResult{}, ErrPropertyNotFound
		}
		return UpdatePropertyResult{}, fmt.Errorf("get property base: %w", err)
	}

	var (
		baseChanged        bool
		subtypeChanged     bool
		locationChanged    bool
		agentChanged       bool
		currentAssignedID  int32
		hasCurrentAssigned bool
	)

	// Compare base fields
	if input.Title != nil && *input.Title != baseRow.Title {
		baseChanged = true
	}
	if input.Description != nil && *input.Description != baseRow.Description {
		baseChanged = true
	}
	if input.LotArea != nil {
		changed, err := numericEqualsFloat64(baseRow.LotArea, *input.LotArea)
		if err != nil {
			return UpdatePropertyResult{}, fmt.Errorf("compare current lot area: %w", err)
		}
		if !changed {
			baseChanged = true
		}
	}
	if input.IsFeatured != nil && *input.IsFeatured != baseRow.IsFeatured {
		baseChanged = true
	}
	if input.AgentID != nil {
		currentAgent, err := queries.GetPrimaryPropertyAgentByPropertyID(ctx, propertyID)
		if err == nil {
			currentAssignedID = currentAgent.AgentID
			hasCurrentAssigned = true
		} else if !errorsIsPgxNoRows(err) {
			return UpdatePropertyResult{}, fmt.Errorf("get current property agent: %w", err)
		}

		if !hasCurrentAssigned || currentAssignedID != *input.AgentID {
			agentChanged = true
		}
	}

	// Compare/prepare subtype updates
	switch baseRow.Subtype {
	case SubtypeResidential:
		if input.Residential != nil {
			// read existing residential
			resRow, err := queries.GetResidentialByPropertyID(ctx, propertyID)
			if err != nil && !errorsIsPgxNoRows(err) {
				return UpdatePropertyResult{}, fmt.Errorf("get residential: %w", err)
			}

			// compare each field (safe to assume exists when subtype is residential)
			if *input.Residential.Bedrooms != resRow.Bedrooms || *input.Residential.Bathrooms != resRow.Bathrooms || *input.Residential.Beds != resRow.Beds || *input.Residential.Floors != resRow.Floors || *input.Residential.ParkingSpots != resRow.ParkingSpots {
				subtypeChanged = true
			}
			builtAreaChanged, err := numericEqualsFloat64(resRow.BuiltArea, *input.Residential.BuiltArea)
			if err != nil {
				return UpdatePropertyResult{}, fmt.Errorf("compare current built area: %w", err)
			}
			if !builtAreaChanged || *input.Residential.ConstructionYear != resRow.ConstructionYear || *input.Residential.OrientationID != resRow.OrientationID || *input.Residential.IsFurnished != resRow.IsFurnished {
				subtypeChanged = true
			}

			if subtypeChanged {
				builtNumeric, err := numericFromFloat64(*input.Residential.BuiltArea)
				if err != nil {
					return UpdatePropertyResult{}, fmt.Errorf("convert built area: %w", err)
				}

				if err := queries.UpdateResidentialPropertyByID(ctx, sqlcgen.UpdateResidentialPropertyByIDParams{
					PropertyID:       propertyID,
					Bedrooms:         *input.Residential.Bedrooms,
					Bathrooms:        *input.Residential.Bathrooms,
					Beds:             *input.Residential.Beds,
					Floors:           *input.Residential.Floors,
					ParkingSpots:     *input.Residential.ParkingSpots,
					BuiltArea:        builtNumeric,
					ConstructionYear: *input.Residential.ConstructionYear,
					OrientationID:    *input.Residential.OrientationID,
					IsFurnished:      *input.Residential.IsFurnished,
				}); err != nil {
					return UpdatePropertyResult{}, fmt.Errorf("update residential: %w", err)
				}
			}
		}
	case SubtypeCommercial:
		if input.Commercial != nil {
			comRow, err := queries.GetCommercialByPropertyID(ctx, propertyID)
			if err != nil && !errorsIsPgxNoRows(err) {
				return UpdatePropertyResult{}, fmt.Errorf("get commercial: %w", err)
			}

			ceilingHeightChanged, err := numericEqualsFloat64(comRow.CeilingHeight, *input.Commercial.CeilingHeight)
			if err != nil {
				return UpdatePropertyResult{}, fmt.Errorf("compare current ceiling height: %w", err)
			}

			if !ceilingHeightChanged || *input.Commercial.LoadingDocks != comRow.LoadingDocks || *input.Commercial.InternalOffices != comRow.InternalOffices || *input.Commercial.ThreePhasePower != comRow.ThreePhasePower || *input.Commercial.LandUse != comRow.LandUse {
				subtypeChanged = true
			}

			if subtypeChanged {
				chNumeric, err := numericFromFloat64(*input.Commercial.CeilingHeight)
				if err != nil {
					return UpdatePropertyResult{}, fmt.Errorf("convert ceiling height: %w", err)
				}

				if err := queries.UpdateCommercialPropertyByID(ctx, sqlcgen.UpdateCommercialPropertyByIDParams{
					PropertyID:      propertyID,
					CeilingHeight:   chNumeric,
					LoadingDocks:    *input.Commercial.LoadingDocks,
					InternalOffices: *input.Commercial.InternalOffices,
					ThreePhasePower: *input.Commercial.ThreePhasePower,
					LandUse:         *input.Commercial.LandUse,
				}); err != nil {
					return UpdatePropertyResult{}, fmt.Errorf("update commercial: %w", err)
				}
			}
		}
	}

	// Location compare
	if input.Location != nil {
		locRow, err := queries.GetLocationByPropertyID(ctx, propertyID)
		if err != nil && !errorsIsPgxNoRows(err) {
			return UpdatePropertyResult{}, fmt.Errorf("get location: %w", err)
		}

		if locRow.CityID != *input.Location.CityID || locRow.Neighborhood != *input.Location.Neighborhood || locRow.Street != *input.Location.Street || locRow.ExteriorNumber != *input.Location.ExteriorNumber || !optionalStringEqual(locRow.InteriorNumber, input.Location.InteriorNumber) || locRow.PostalCode != *input.Location.PostalCode || locRow.IsPublicAddress != *input.Location.IsPublicAddress {
			locationChanged = true
		} else {
			// compare coords
			if locRow.Latitude != *input.Location.Latitude || locRow.Longitude != *input.Location.Longitude {
				locationChanged = true
			}
		}

		if locationChanged {
			if err := queries.UpdateLocationByID(ctx, sqlcgen.UpdateLocationByIDParams{
				CityID:          *input.Location.CityID,
				Neighborhood:    *input.Location.Neighborhood,
				Street:          *input.Location.Street,
				ExteriorNumber:  *input.Location.ExteriorNumber,
				InteriorNumber:  textFromPointer(input.Location.InteriorNumber),
				PostalCode:      *input.Location.PostalCode,
				Longitude:       *input.Location.Longitude,
				Latitude:        *input.Location.Latitude,
				IsPublicAddress: *input.Location.IsPublicAddress,
				PropertyID:      propertyID,
			}); err != nil {
				return UpdatePropertyResult{}, fmt.Errorf("update location: %w", err)
			}
		}
	}

	// Apply base update if needed
	if baseChanged {
		var lotNumeric pgtype.Numeric
		if input.LotArea != nil {
			v, err := numericFromFloat64(*input.LotArea)
			if err != nil {
				return UpdatePropertyResult{}, fmt.Errorf("convert lot area: %w", err)
			}
			lotNumeric = v
		} else {
			lotNumeric = baseRow.LotArea
		}

		title := baseRow.Title
		if input.Title != nil {
			title = *input.Title
		}
		description := baseRow.Description
		if input.Description != nil {
			description = *input.Description
		}
		isFeatured := baseRow.IsFeatured
		if input.IsFeatured != nil {
			isFeatured = *input.IsFeatured
		}

		if err := queries.UpdatePropertyBaseByID(ctx, sqlcgen.UpdatePropertyBaseByIDParams{
			PropertyID:  propertyID,
			Title:       title,
			Description: description,
			LotArea:     lotNumeric,
			IsFeatured:  isFeatured,
		}); err != nil {
			return UpdatePropertyResult{}, fmt.Errorf("update property base: %w", err)
		}
	}

	if agentChanged {
		if err := queries.DeletePropertyAgents(ctx, propertyID); err != nil {
			return UpdatePropertyResult{}, fmt.Errorf("delete property agents: %w", err)
		}
		if err := queries.CreatePropertyAgent(ctx, sqlcgen.CreatePropertyAgentParams{
			PropertyID: propertyID,
			AgentID:    *input.AgentID,
		}); err != nil {
			return UpdatePropertyResult{}, fmt.Errorf("create property agent: %w", err)
		}
	}

	if !baseChanged && !subtypeChanged && !locationChanged && !agentChanged {
		if err := tx.Commit(ctx); err != nil {
			return UpdatePropertyResult{}, fmt.Errorf("commit transaction: %w", err)
		}
		return UpdatePropertyResult{Message: "no changes detected"}, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return UpdatePropertyResult{}, fmt.Errorf("commit transaction: %w", err)
	}

	return UpdatePropertyResult{Message: "property updated successfully"}, nil
}
