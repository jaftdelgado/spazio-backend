package properties

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestService_GetPhotos tests the GetPhotos service method in table-driven format.
func TestService_GetPhotos(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		repoResult GetPropertyPhotosResult
		repoErr    error
		wantErr    bool
		wantData   []PropertyPhotoData
	}{
		{
			name: "returns property photos when photos are found",
			repoResult: GetPropertyPhotosResult{
				Data: []PropertyPhotoData{
					{PhotoID: 1, StorageKey: "properties/uuid/photos/1.webp"},
					{PhotoID: 2, StorageKey: "properties/uuid/photos/2.webp"},
				},
			},
			repoErr: nil,
			wantErr: false,
			wantData: []PropertyPhotoData{
				{
					PhotoID:    1,
					StorageKey: "properties/uuid/photos/1.webp",
					URL:        "https://cdn.example.com/properties/uuid/photos/1.webp",
				},
				{
					PhotoID:    2,
					StorageKey: "properties/uuid/photos/2.webp",
					URL:        "https://cdn.example.com/properties/uuid/photos/2.webp",
				},
			},
		},
		{
			name: "returns empty property photos when no photos are found",
			repoResult: GetPropertyPhotosResult{
				Data: []PropertyPhotoData{},
			},
			repoErr:  nil,
			wantErr:  false,
			wantData: []PropertyPhotoData{},
		},
		{
			name:       "returns error when property is not found",
			repoResult: GetPropertyPhotosResult{},
			repoErr:    ErrPropertyNotFound,
			wantErr:    true,
		},
		{
			name:       "returns error when repository fails",
			repoResult: GetPropertyPhotosResult{},
			repoErr:    errors.New("db"),
			wantErr:    true,
		},
		{
			name: "returns error when public url generation fails",
			repoResult: GetPropertyPhotosResult{
				Data: []PropertyPhotoData{{PhotoID: 1, StorageKey: "properties/uuid/photos/1.webp"}},
			},
			repoErr:  nil,
			wantErr:  true,
			wantData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyPhotosFunc: func(ctx context.Context, uuid string) (GetPropertyPhotosResult, error) {
					return tt.repoResult, tt.repoErr
				},
			}

			storage := &mockPropertyPhotoStorage{
				publicURLFunc: func(ctx context.Context, storageKey string) (string, error) {
					if tt.name == "returns error when public url generation fails" {
						return "", errors.New("url failed")
					}
					return "https://cdn.example.com/" + storageKey, nil
				},
			}

			svc := NewService(repo, storage)
			result, err := svc.GetPhotos(context.Background(), validUUID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.repoErr == ErrPropertyNotFound && !errors.Is(err, ErrPropertyNotFound) {
					t.Fatalf("error type: got %v, want ErrPropertyNotFound", err)
				}
				if tt.name == "returns error when public url generation fails" && !strings.Contains(err.Error(), "build property photo public url") {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result.Data, tt.wantData) {
					t.Fatalf("data mismatch: got %#v want %#v", result.Data, tt.wantData)
				}
			}
		})
	}
}

// TestService_UpdatePhotos tests the UpdatePhotos service method in table-driven format.
func TestService_UpdatePhotos(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name           string
		input          UpdatePropertyPhotosInput
		repoErr        error
		wantErr        bool
		wantRepoCalled bool
		wantInput      UpdatePropertyPhotosInput
	}{
		{
			name: "returns error when photo id is zero",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 0}},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when photo ids are duplicated",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1}, {PhotoID: 1}},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when no cover photo is provided",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: false}},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when multiple cover photos are provided",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}, {PhotoID: 2, IsCover: true}},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "updates property photos successfully with empty photos",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{},
			},
		},
		{
			name: "updates property photos successfully",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
		},
		{
			name: "returns validation error when repository validation fails",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
			repoErr:        ValidationError{Message: "invalid photo_id"},
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
		},
		{
			name: "returns error when property is not found",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
			repoErr:        ErrPropertyNotFound,
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
		},
		{
			name: "returns error when repository fails",
			input: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
			repoErr:        errors.New("db"),
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyPhotosInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Photos: []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUpdateCalled := false
			var calledInput UpdatePropertyPhotosInput

			repo := &mockPropertyRepository{
				updatePropertyPhotosFunc: func(ctx context.Context, uuid string, input UpdatePropertyPhotosInput) error {
					repoUpdateCalled = true
					calledInput = input
					return tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			err := svc.UpdatePhotos(context.Background(), validUUID, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				// Validations that aren't repo errors must be ValidationError
				if tt.repoErr == nil {
					var ve ValidationError
					if !errors.As(err, &ve) {
						t.Fatalf("expected ValidationError, got %T: %v", err, err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if repoUpdateCalled != tt.wantRepoCalled {
				t.Fatalf("repo called: got %v, want %v", repoUpdateCalled, tt.wantRepoCalled)
			}

			if tt.wantRepoCalled {
				if !reflect.DeepEqual(calledInput, tt.wantInput) {
					t.Fatalf("input mismatch: got %#v want %#v", calledInput, tt.wantInput)
				}
			}
		})
	}
}

// TestValidatePhotoMetadataInputs tests the validatePhotoMetadataInputs function directly.
func TestValidatePhotoMetadataInputs(t *testing.T) {
	tests := []struct {
		name    string
		input   []UpdatePhotoMetadataInput
		wantErr bool
	}{
		{
			name:    "returns nil when photos list is empty",
			input:   []UpdatePhotoMetadataInput{},
			wantErr: false,
		},
		{
			name:    "returns nil when photo metadata is valid",
			input:   []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}},
			wantErr: false,
		},
		{
			name:    "returns error when photo id is zero",
			input:   []UpdatePhotoMetadataInput{{PhotoID: 0, IsCover: true}},
			wantErr: true,
		},
		{
			name:    "returns error when photo ids are duplicated",
			input:   []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}, {PhotoID: 1, IsCover: false}},
			wantErr: true,
		},
		{
			name:    "returns error when no cover photo is provided",
			input:   []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: false}},
			wantErr: true,
		},
		{
			name:    "returns error when multiple cover photos are provided",
			input:   []UpdatePhotoMetadataInput{{PhotoID: 1, IsCover: true}, {PhotoID: 2, IsCover: true}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhotoMetadataInputs(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
