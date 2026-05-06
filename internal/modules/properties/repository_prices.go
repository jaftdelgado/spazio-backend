package properties

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPropertyPrices(ctx context.Context, propertyUUID string) (GetPropertyPricesResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyPricesResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyPricesResult{}, err
	}

	// Get active sale price
	saleRow, err := r.queries.ListActiveSalePrice(ctx, propertyID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return GetPropertyPricesResult{}, fmt.Errorf("list active sale price: %w", err)
	}

	var salePrice *ActiveSalePriceData
	if err == nil {
		floatValue, err := saleRow.SalePrice.Float64Value()
		if err != nil {
			return GetPropertyPricesResult{}, fmt.Errorf("convert sale price: %w", err)
		}

		salePrice = &ActiveSalePriceData{
			SalePrice:    floatValue.Float64,
			Currency:     saleRow.Currency,
			IsNegotiable: saleRow.IsNegotiable,
		}
	}

	// Get active rent prices
	rentRows, err := r.queries.ListActiveRentPrices(ctx, propertyID)
	if err != nil {
		return GetPropertyPricesResult{}, fmt.Errorf("list active rent prices: %w", err)
	}

	rentPrices := make([]ActiveRentPriceData, 0, len(rentRows))
	for _, row := range rentRows {
		floatValue, err := row.RentPrice.Float64Value()
		if err != nil {
			return GetPropertyPricesResult{}, fmt.Errorf("convert rent price: %w", err)
		}

		data := ActiveRentPriceData{
			PeriodID:     row.PeriodID,
			RentPrice:    floatValue.Float64,
			Currency:     row.Currency,
			IsNegotiable: row.IsNegotiable,
		}

		if row.Deposit.Valid {
			depositValue, err := row.Deposit.Float64Value()
			if err != nil {
				return GetPropertyPricesResult{}, fmt.Errorf("convert deposit: %w", err)
			}
			if depositValue.Valid {
				deposit := depositValue.Float64
				data.Deposit = &deposit
			}
		}

		rentPrices = append(rentPrices, data)
	}

	return GetPropertyPricesResult{
		Data: GetPropertyPricesData{
			SalePrice:  salePrice,
			RentPrices: rentPrices,
		},
	}, nil
}

func (r *repository) UpdatePropertyPrices(ctx context.Context, propertyUUID string, input UpdatePropertyPricesInput) error {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return fmt.Errorf("parse property uuid: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := sqlcgen.New(tx)

	propertyID, err := getPropertyIDByUUID(ctx, queries, parsedUUID)
	if err != nil {
		return err
	}

	// Get owner ID for changed_by_user_id
	ownerID, err := r.getOwnerIDFromProperty(ctx, queries, propertyID)
	if err != nil {
		return err
	}

	// Update sale price if provided
	if input.SalePrice != nil {
		if err := r.updateSalePriceWithHistory(ctx, queries, propertyID, ownerID, *input.SalePrice); err != nil {
			return err
		}
	}

	// Update rent prices if provided
	for _, rentPriceUpdate := range input.RentPrices {
		if err := r.updateRentPriceWithHistory(ctx, queries, propertyID, ownerID, rentPriceUpdate); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) getOwnerIDFromProperty(ctx context.Context, queries *sqlcgen.Queries, propertyID int32) (int32, error) {
	ownerID, err := queries.GetPropertyOwnerID(ctx, propertyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPropertyNotFound
		}
		return 0, fmt.Errorf("get property owner id: %w", err)
	}

	return ownerID, nil
}

func (r *repository) updateSalePriceWithHistory(ctx context.Context, queries *sqlcgen.Queries, propertyID, ownerID int32, update UpdateSalePriceInput) error {
	// Get current active sale price
	currentRow, err := queries.ListActiveSalePrice(ctx, propertyID)
	hasActiveSalePrice := true
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("get current sale price: %w", err)
		}
		hasActiveSalePrice = false
	}

	newAmount, err := numericFromFloat64(update.SalePrice)
	if err != nil {
		return fmt.Errorf("convert new sale price: %w", err)
	}

	// If there's an active price, check if amount changed
	if hasActiveSalePrice {
		currentAmount, err := currentRow.SalePrice.Float64Value()
		if err != nil {
			return fmt.Errorf("convert current sale price: %w", err)
		}

		if currentAmount.Float64 != update.SalePrice {
			// Amount changed: close old record and create new one
			if err := queries.UpdateSalePriceToInactive(ctx, propertyID); err != nil {
				return fmt.Errorf("mark sale price inactive: %w", err)
			}

			if err := queries.CreateSalePriceHistoryRecord(ctx, sqlcgen.CreateSalePriceHistoryRecordParams{
				PropertyID:      propertyID,
				SalePrice:       newAmount,
				Currency:        currentRow.Currency,
				IsNegotiable:    update.IsNegotiable,
				ChangedByUserID: ownerID,
			}); err != nil {
				return fmt.Errorf("create sale price history record: %w", err)
			}
		} else {
			// Amount didn't change: just update is_negotiable
			if err := queries.UpdateSalePriceIsNegotiable(ctx, sqlcgen.UpdateSalePriceIsNegotiableParams{
				PropertyID:   propertyID,
				IsNegotiable: update.IsNegotiable,
			}); err != nil {
				return fmt.Errorf("update sale price is_negotiable: %w", err)
			}
		}
	} else {
		return ValidationError{Message: "no active sale_price found for this property"}
	}

	return nil
}

func (r *repository) updateRentPriceWithHistory(ctx context.Context, queries *sqlcgen.Queries, propertyID, ownerID int32, update UpdateRentPriceInput) error {
	// Get current active rent price for this period
	currentRow, err := queries.ListActiveRentPriceByPeriod(ctx, sqlcgen.ListActiveRentPriceByPeriodParams{
		PropertyID: propertyID,
		PeriodID:   update.PeriodID,
	})
	hasActiveRentPrice := true
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("get current rent price: %w", err)
		}
		hasActiveRentPrice = false
	}

	newAmount, err := numericFromFloat64(update.RentPrice)
	if err != nil {
		return fmt.Errorf("convert new rent price: %w", err)
	}

	newDeposit, err := numericFromPointer(update.Deposit)
	if err != nil {
		return fmt.Errorf("convert new deposit: %w", err)
	}

	// If there's an active price, check if amount changed
	if hasActiveRentPrice {
		currentAmount, err := currentRow.RentPrice.Float64Value()
		if err != nil {
			return fmt.Errorf("convert current rent price: %w", err)
		}

		if currentAmount.Float64 != update.RentPrice {
			// Amount changed: close old record and create new one
			if err := queries.UpdateRentPriceToInactive(ctx, sqlcgen.UpdateRentPriceToInactiveParams{
				PropertyID: propertyID,
				PeriodID:   update.PeriodID,
			}); err != nil {
				return fmt.Errorf("mark rent price inactive: %w", err)
			}

			if err := queries.CreateRentPriceHistoryRecord(ctx, sqlcgen.CreateRentPriceHistoryRecordParams{
				PropertyID:      propertyID,
				PeriodID:        update.PeriodID,
				RentPrice:       newAmount,
				Deposit:         newDeposit,
				Currency:        currentRow.Currency,
				IsNegotiable:    update.IsNegotiable,
				ChangedByUserID: ownerID,
			}); err != nil {
				return fmt.Errorf("create rent price history record: %w", err)
			}
		} else {
			// Amount didn't change: just update is_negotiable and deposit
			if err := queries.UpdateRentPriceIsNegotiableAndDeposit(ctx, sqlcgen.UpdateRentPriceIsNegotiableAndDepositParams{
				PropertyID:   propertyID,
				PeriodID:     update.PeriodID,
				IsNegotiable: update.IsNegotiable,
				Deposit:      newDeposit,
			}); err != nil {
				return fmt.Errorf("update rent price is_negotiable and deposit: %w", err)
			}
		}
	} else {
		return ValidationError{Message: fmt.Sprintf("no active rent_price found for period_id %d", update.PeriodID)}
	}

	return nil
}
