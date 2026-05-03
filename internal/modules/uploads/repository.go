package uploads

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) UploadsRepository {
	return &repository{db: db}
}

func (r *repository) SavePropertyPhoto(ctx context.Context, input SavePhotoInput) (int32, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := sqlcgen.New(tx)

	parsedUUID := uuid.MustParse(input.PropertyUUID)
	pgUUID := pgtype.UUID{Bytes: parsedUUID, Valid: true}

	propertyID, err := q.GetPropertyIDByUUID(ctx, pgUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ErrPropertyNotFound
		}
		return 0, fmt.Errorf("get property id: %w", err)
	}

	if input.IsCover {
		if err := q.ClearPropertyPhotoCover(ctx, propertyID); err != nil {
			return 0, fmt.Errorf("clear property photo cover: %w", err)
		}
	}

	var label pgtype.Text
	if input.Label != nil {
		label = pgtype.Text{String: *input.Label, Valid: true}
	}

	var altText pgtype.Text
	if input.AltText != nil {
		altText = pgtype.Text{String: *input.AltText, Valid: true}
	}

	photoID, err := q.InsertPropertyPhoto(ctx, sqlcgen.InsertPropertyPhotoParams{
		PropertyID: propertyID,
		StorageKey: input.StorageKey,
		MimeType:   input.MimeType,
		SortOrder:  int16(input.SortOrder),
		IsCover:    input.IsCover,
		Label:      label,
		AltText:    altText,
	})
	if err != nil {
		return 0, fmt.Errorf("insert property photo: %w", err)
	}

	if input.IsCover {
		err = q.UpdatePropertyCoverPhoto(ctx, sqlcgen.UpdatePropertyCoverPhotoParams{
			PropertyID:    propertyID,
			CoverPhotoUrl: pgtype.Text{String: input.StorageKey, Valid: true},
		})
		if err != nil {
			return 0, fmt.Errorf("update cover photo: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return photoID, nil
}
