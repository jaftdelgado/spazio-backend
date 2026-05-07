package properties

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type deleteArgs struct {
	propertyID      int32
	changedByUserID int32
}

func TestService_DeleteProperty(t *testing.T) {
	propertyUUID := "123e4567-e89b-12d3-a456-426614174000"
	changedByUserID := int32(7)

	tests := []struct {
		name                   string
		propertyResult         GetPropertyResult
		propertyErr            error
		storageKeys            []string
		storageKeysErr         error
		storageDeleteErrKey    string
		deleteRepoErr          error
		input                  DeletePropertyInput
		wantErr                bool
		wantErrIs              error
		wantErrContains        string
		wantErrExact           string
		wantValidation         bool
		wantRepoCalled         bool
		wantStorageDeleteCalls int
		wantDeleteArgs         deleteArgs
	}{
		{
			name: "deletes property successfully when no photos exist",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 10, StatusID: StatusAvailable},
			},
			storageKeys:            []string{},
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                false,
			wantRepoCalled:         true,
			wantStorageDeleteCalls: 0,
			wantDeleteArgs:         deleteArgs{propertyID: 10, changedByUserID: changedByUserID},
		},
		{
			name: "returns validation error when property status is not available",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 11, StatusID: StatusDeleted},
			},
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantValidation:         true,
			wantRepoCalled:         false,
			wantStorageDeleteCalls: 0,
		},
		{
			name:                   "returns error when property is not found",
			propertyErr:            ErrPropertyNotFound,
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantErrIs:              ErrPropertyNotFound,
			wantRepoCalled:         false,
			wantStorageDeleteCalls: 0,
		},
		{
			name:                   "returns error when get property fails",
			propertyErr:            errors.New("db error"),
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantErrContains:        "get property:",
			wantRepoCalled:         false,
			wantStorageDeleteCalls: 0,
		},
		{
			name: "returns error when getting storage keys fails",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 12, StatusID: StatusAvailable},
			},
			storageKeysErr:         errors.New("storage keys error"),
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantErrContains:        "get property storage keys:",
			wantRepoCalled:         false,
			wantStorageDeleteCalls: 0,
		},
		{
			name: "returns error when repository delete fails",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 13, StatusID: StatusAvailable},
			},
			storageKeys:            []string{},
			deleteRepoErr:          errors.New("transaction failed"),
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantErrExact:           "could not complete deletion: database transaction failed",
			wantRepoCalled:         true,
			wantStorageDeleteCalls: 0,
			wantDeleteArgs:         deleteArgs{propertyID: 13, changedByUserID: changedByUserID},
		},
		{
			name: "deletes property photos from storage successfully with one photo",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 14, StatusID: StatusAvailable},
			},
			storageKeys:            []string{"key-1"},
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                false,
			wantRepoCalled:         true,
			wantStorageDeleteCalls: 1,
			wantDeleteArgs:         deleteArgs{propertyID: 14, changedByUserID: changedByUserID},
		},
		{
			name: "deletes property photos from storage successfully with multiple photos",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 15, StatusID: StatusAvailable},
			},
			storageKeys:            []string{"key-1", "key-2", "key-3"},
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                false,
			wantRepoCalled:         true,
			wantStorageDeleteCalls: 3,
			wantDeleteArgs:         deleteArgs{propertyID: 15, changedByUserID: changedByUserID},
		},
		{
			name: "returns error when deleting property photos from storage fails",
			propertyResult: GetPropertyResult{
				Data: GetPropertyData{PropertyID: 16, StatusID: StatusAvailable},
			},
			storageKeys:            []string{"key-1", "key-2"},
			storageDeleteErrKey:    "key-1",
			input:                  DeletePropertyInput{Confirm: true, ChangedByUserID: changedByUserID},
			wantErr:                true,
			wantErrExact:           "could not delete property photos from storage",
			wantRepoCalled:         false,
			wantStorageDeleteCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoCalled := false
			var calledArgs deleteArgs

			repo := &mockPropertyRepository{
				getPropertyFunc: func(ctx context.Context, uuid string) (GetPropertyResult, error) {
					if tt.propertyErr != nil {
						return GetPropertyResult{}, tt.propertyErr
					}
					return tt.propertyResult, nil
				},
				getPropertyStorageKeysFunc: func(ctx context.Context, propertyID int32) ([]string, error) {
					if tt.storageKeysErr != nil {
						return nil, tt.storageKeysErr
					}
					return tt.storageKeys, nil
				},
				deletePropertyFunc: func(ctx context.Context, propertyID int32, changedBy int32) error {
					repoCalled = true
					calledArgs = deleteArgs{propertyID: propertyID, changedByUserID: changedBy}
					return tt.deleteRepoErr
				},
			}

			storageCalls := 0
			storage := &mockPropertyPhotoStorage{
				deleteFunc: func(ctx context.Context, storageKey string) error {
					storageCalls++
					if tt.storageDeleteErrKey != "" && storageKey == tt.storageDeleteErrKey {
						return errors.New("storage error")
					}
					return nil
				},
			}

			svc := NewService(repo, storage)
			err := svc.DeleteProperty(context.Background(), propertyUUID, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				if tt.wantValidation {
					var validationErr ValidationError
					if !errors.As(err, &validationErr) {
						t.Fatalf("expected ValidationError, got %T", err)
					}
				}
				if tt.wantErrExact != "" && err.Error() != tt.wantErrExact {
					t.Fatalf("error message: got %q, want %q", err.Error(), tt.wantErrExact)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if repoCalled != tt.wantRepoCalled {
				t.Fatalf("repo called: got %v, want %v", repoCalled, tt.wantRepoCalled)
			}
			if tt.wantRepoCalled {
				if !reflect.DeepEqual(calledArgs, tt.wantDeleteArgs) {
					t.Fatalf("delete args mismatch: got %#v want %#v", calledArgs, tt.wantDeleteArgs)
				}
			}
			if storageCalls != tt.wantStorageDeleteCalls {
				t.Fatalf("storage delete calls: got %d, want %d", storageCalls, tt.wantStorageDeleteCalls)
			}
		})
	}
}
