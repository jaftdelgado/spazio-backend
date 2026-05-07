package properties

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestService_UpdateProperty tests the UpdateProperty service method in table-driven format.
func TestService_UpdateProperty(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name           string
		repoResult     UpdatePropertyResult
		repoErr        error
		wantErr        bool
		wantRepoCalled bool
		wantResult     UpdatePropertyResult
	}{
		{
			name:           "returns updated message when property is updated successfully",
			repoResult:     UpdatePropertyResult{Message: "property updated successfully"},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantResult:     UpdatePropertyResult{Message: "property updated successfully"},
		},
		{
			name:           "returns no changes message when no updates are applied",
			repoResult:     UpdatePropertyResult{Message: "no changes detected"},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantResult:     UpdatePropertyResult{Message: "no changes detected"},
		},
		{
			name:           "returns error when property is not found",
			repoResult:     UpdatePropertyResult{},
			repoErr:        ErrPropertyNotFound,
			wantErr:        true,
			wantRepoCalled: true,
		},
		{
			name:           "returns error when repository fails",
			repoResult:     UpdatePropertyResult{},
			repoErr:        errors.New("db"),
			wantErr:        true,
			wantRepoCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUpdateCalled := false
			var calledUUID string
			var calledInput UpdatePropertyInput

			repo := &mockPropertyRepository{
				updatePropertyFunc: func(ctx context.Context, propertyUUID string, input UpdatePropertyInput) (UpdatePropertyResult, error) {
					repoUpdateCalled = true
					calledUUID = propertyUUID
					calledInput = input
					return tt.repoResult, tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})

			dummyInput := UpdatePropertyInput{
				Title: ptrString("New Title"),
			}

			result, err := svc.UpdateProperty(context.Background(), validUUID, dummyInput)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.repoErr == ErrPropertyNotFound && !errors.Is(err, ErrPropertyNotFound) {
					t.Fatalf("error type: got %v, want ErrPropertyNotFound", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.wantResult) {
					t.Fatalf("result mismatch: got %#v want %#v", result, tt.wantResult)
				}
			}

			if repoUpdateCalled != tt.wantRepoCalled {
				t.Fatalf("repo called: got %v, want %v", repoUpdateCalled, tt.wantRepoCalled)
			}

			if tt.wantRepoCalled {
				if calledUUID != validUUID {
					t.Fatalf("uuid mismatch: got %v want %v", calledUUID, validUUID)
				}
				if !reflect.DeepEqual(calledInput, dummyInput) {
					t.Fatalf("input mismatch: got %#v want %#v", calledInput, dummyInput)
				}
			}
		})
	}
}
