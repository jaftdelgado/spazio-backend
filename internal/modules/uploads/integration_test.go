//go:build integration

package uploads

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

func TestIntegration_UploadsRepository_SavePropertyPhoto(t *testing.T) {
	pool := shared.SetupTestDB(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	var propertyID int32
	var propertyUUID string
	var originalCover *string
	if err := pool.QueryRow(ctx, `
		SELECT property_id, property_uuid::text, cover_photo_url
		FROM properties
		WHERE deleted_at IS NULL
		LIMIT 1
	`).Scan(&propertyID, &propertyUUID, &originalCover); err != nil {
		t.Fatalf("query integration property: %v", err)
	}

	key := fmt.Sprintf("properties/%s/photos/integration-%d.webp", propertyUUID, time.Now().UnixNano())
	label := "Integration Photo"
	altText := "Integration Alt"

	photoID, err := repo.SavePropertyPhoto(ctx, SavePhotoInput{
		PropertyUUID: propertyUUID,
		StorageKey:   key,
		MimeType:     "image/webp",
		Label:        &label,
		AltText:      &altText,
		SortOrder:    77,
		IsCover:      true,
	})
	if err != nil {
		t.Fatalf("SavePropertyPhoto() error = %v", err)
	}
	if photoID == 0 {
		t.Fatal("expected photo id")
	}

	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM property_photos WHERE photo_id = $1`, photoID)
		if originalCover == nil {
			_, _ = pool.Exec(ctx, `UPDATE properties SET cover_photo_url = NULL WHERE property_id = $1`, propertyID)
		} else {
			_, _ = pool.Exec(ctx, `UPDATE properties SET cover_photo_url = $1 WHERE property_id = $2`, *originalCover, propertyID)
		}
	}()

	var storedKey string
	var isCover bool
	if err := pool.QueryRow(ctx, `
		SELECT storage_key, is_cover
		FROM property_photos
		WHERE photo_id = $1
	`, photoID).Scan(&storedKey, &isCover); err != nil {
		t.Fatalf("query stored photo: %v", err)
	}
	if storedKey != key || !isCover {
		t.Fatalf("unexpected stored photo values: key=%q is_cover=%v", storedKey, isCover)
	}

	var coverPhotoURL *string
	if err := pool.QueryRow(ctx, `
		SELECT cover_photo_url
		FROM properties
		WHERE property_id = $1
	`, propertyID).Scan(&coverPhotoURL); err != nil {
		t.Fatalf("query updated cover photo: %v", err)
	}
	if coverPhotoURL == nil || *coverPhotoURL != key {
		t.Fatalf("unexpected cover_photo_url: %+v want %q", coverPhotoURL, key)
	}
}
