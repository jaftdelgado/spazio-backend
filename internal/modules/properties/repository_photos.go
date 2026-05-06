package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func (r *repository) GetPropertyPhotos(ctx context.Context, propertyUUID string) (GetPropertyPhotosResult, error) {
	parsedUUID, err := uuid.Parse(propertyUUID)
	if err != nil {
		return GetPropertyPhotosResult{}, fmt.Errorf("parse property uuid: %w", err)
	}

	propertyID, err := getPropertyIDByUUID(ctx, r.queries, parsedUUID)
	if err != nil {
		return GetPropertyPhotosResult{}, err
	}

	rows, err := r.queries.ListPropertyPhotos(ctx, propertyID)
	if err != nil {
		return GetPropertyPhotosResult{}, fmt.Errorf("list property photos: %w", err)
	}

	result := GetPropertyPhotosResult{Data: make([]PropertyPhotoData, 0, len(rows))}
	for _, row := range rows {
		result.Data = append(result.Data, propertyPhotoDataFromRow(row))
	}

	return result, nil
}

func (r *repository) UpdatePropertyPhotos(ctx context.Context, propertyUUID string, input UpdatePropertyPhotosInput) error {
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

	if len(input.Photos) == 0 {
		if err := queries.DeletePropertyPhotos(ctx, propertyID); err != nil {
			return fmt.Errorf("delete property photos: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}

		return nil
	}

	photoIDs := make([]int32, 0, len(input.Photos))
	for _, photo := range input.Photos {
		photoIDs = append(photoIDs, photo.PhotoID)
	}

	rows, err := queries.ListPropertyPhotosByIDs(ctx, sqlcgen.ListPropertyPhotosByIDsParams{
		PropertyID: propertyID,
		Column2:    photoIDs,
	})
	if err != nil {
		return fmt.Errorf("list property photos by ids: %w", err)
	}

	if len(rows) != len(photoIDs) {
		return ValidationError{Message: "all photo_id values must belong to the property"}
	}

	storageKeysByPhotoID := make(map[int32]string, len(rows))
	var coverStorageKey string
	for _, row := range rows {
		storageKeysByPhotoID[row.PhotoID] = row.StorageKey
	}

	if err := queries.ClearPropertyPhotoCover(ctx, propertyID); err != nil {
		return fmt.Errorf("clear property photo cover: %w", err)
	}

	if err := queries.DeletePropertyPhotosExceptIDs(ctx, sqlcgen.DeletePropertyPhotosExceptIDsParams{
		PropertyID: propertyID,
		Column2:    photoIDs,
	}); err != nil {
		return fmt.Errorf("delete property photos except ids: %w", err)
	}

	for _, photo := range input.Photos {
		if photo.IsCover {
			coverStorageKey = storageKeysByPhotoID[photo.PhotoID]
		}

		if err := queries.UpdatePropertyPhotoMetadata(ctx, sqlcgen.UpdatePropertyPhotoMetadataParams{
			PropertyID: propertyID,
			PhotoID:    photo.PhotoID,
			SortOrder:  photo.SortOrder,
			IsCover:    photo.IsCover,
			Label:      textFromPointer(photo.Label),
			AltText:    textFromPointer(photo.AltText),
		}); err != nil {
			return fmt.Errorf("update property photo %d: %w", photo.PhotoID, err)
		}
	}

	if err := queries.UpdatePropertyCoverPhoto(ctx, sqlcgen.UpdatePropertyCoverPhotoParams{
		PropertyID:    propertyID,
		CoverPhotoUrl: textFromPointer(stringPointer(coverStorageKey)),
	}); err != nil {
		return fmt.Errorf("update property cover photo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func propertyPhotoDataFromRow(row sqlcgen.ListPropertyPhotosRow) PropertyPhotoData {
	return PropertyPhotoData{
		PhotoID:    row.PhotoID,
		StorageKey: row.StorageKey,
		MimeType:   row.MimeType,
		SortOrder:  row.SortOrder,
		IsCover:    row.IsCover,
		Label:      stringPointerFromText(row.Label),
		AltText:    stringPointerFromText(row.AltText),
	}
}

func stringPointerFromText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	text := value.String
	return &text
}

func stringPointer(value string) *string {
	return &value
}
