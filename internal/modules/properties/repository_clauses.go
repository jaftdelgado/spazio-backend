package properties

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPropertyClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyClausesResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyClausesResult{}, err
	}

	rows, err := r.queries.ListPropertyClauses(ctx, propertyID)
	if err != nil {
		return GetPropertyClausesResult{}, fmt.Errorf("list property clauses: %w", err)
	}

	result := GetPropertyClausesResult{Data: make([]PropertyClauseData, 0, len(rows))}
	for _, row := range rows {
		data, err := propertyClauseDataFromRow(row)
		if err != nil {
			return GetPropertyClausesResult{}, err
		}
		result.Data = append(result.Data, data)
	}

	return result, nil
}

func (r *repository) UpdatePropertyClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error {
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

	if err := queries.DeletePropertyClauses(ctx, propertyID); err != nil {
		return fmt.Errorf("delete property clauses: %w", err)
	}

	for _, clause := range input.Clauses {
		if err := r.createPropertyClause(ctx, queries, propertyID, clause); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func getPropertyIDByUUID(ctx context.Context, queries *sqlcgen.Queries, propertyUUID uuid.UUID) (int32, error) {
	pgUUID := pgtype.UUID{Bytes: propertyUUID, Valid: true}

	propertyID, err := queries.GetPropertyIDByUUID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPropertyNotFound
		}
		return 0, fmt.Errorf("get property id: %w", err)
	}

	return propertyID, nil
}

func propertyClauseDataFromRow(row sqlcgen.ListPropertyClausesRow) (PropertyClauseData, error) {
	data := PropertyClauseData{ClauseID: row.ClauseID}

	if row.BooleanValue.Valid {
		value := row.BooleanValue.Bool
		data.BooleanValue = &value
	}

	if row.IntegerValue.Valid {
		value := row.IntegerValue.Int32
		data.IntegerValue = &value
	}

	if row.MinValue.Valid {
		floatValue, err := row.MinValue.Float64Value()
		if err != nil {
			return PropertyClauseData{}, fmt.Errorf("convert min value: %w", err)
		}
		if floatValue.Valid {
			value := floatValue.Float64
			data.MinValue = &value
		}
	}

	if row.MaxValue.Valid {
		floatValue, err := row.MaxValue.Float64Value()
		if err != nil {
			return PropertyClauseData{}, fmt.Errorf("convert max value: %w", err)
		}
		if floatValue.Valid {
			value := floatValue.Float64
			data.MaxValue = &value
		}
	}

	return data, nil
}
