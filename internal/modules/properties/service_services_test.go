package properties

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestService_GetServices tests the GetServices service method in table-driven format.
func TestService_GetServices(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		repoResult GetPropertyServicesResult
		repoErr    error
		wantErr    bool
		wantData   []int32
	}{
		{
			name: "returns service ids when property services are found",
			repoResult: GetPropertyServicesResult{
				Data: GetPropertyServicesData{
					ServiceIDs: []int32{1, 3, 7},
				},
			},
			repoErr:  nil,
			wantErr:  false,
			wantData: []int32{1, 3, 7},
		},
		{
			name: "returns empty service ids when property has no services",
			repoResult: GetPropertyServicesResult{
				Data: GetPropertyServicesData{
					ServiceIDs: []int32{},
				},
			},
			repoErr:  nil,
			wantErr:  false,
			wantData: []int32{},
		},
		{
			name:       "returns error when property is not found",
			repoResult: GetPropertyServicesResult{},
			repoErr:    ErrPropertyNotFound,
			wantErr:    true,
		},
		{
			name:       "returns error when repository fails",
			repoResult: GetPropertyServicesResult{},
			repoErr:    errors.New("db"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyServicesFunc: func(ctx context.Context, propertyUUID string) (GetPropertyServicesResult, error) {
					return tt.repoResult, tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetServices(context.Background(), validUUID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.repoErr == ErrPropertyNotFound && !errors.Is(err, ErrPropertyNotFound) {
					t.Fatalf("error type: got %v, want ErrPropertyNotFound", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result.Data.ServiceIDs, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", result.Data.ServiceIDs, tt.wantData)
			}
		})
	}
}

// TestService_UpdateServices tests the UpdateServices service method in table-driven format.
func TestService_UpdateServices(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name           string
		input          UpdatePropertyServicesInput
		repoErr        error
		wantErr        bool
		wantRepoCalled bool
		wantInput      UpdatePropertyServicesInput
	}{
		{
			name: "returns error when service id is zero",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{0},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "returns error when service id is negative",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{-1},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
		},
		{
			name: "updates property services successfully with empty service ids",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyServicesInput{
				ServiceIDs: []int32{},
			},
		},
		{
			name: "updates property services successfully",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1, 3, 7},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1, 3, 7},
			},
		},
		{
			name: "returns error when property is not found",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1},
			},
			repoErr:        ErrPropertyNotFound,
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1},
			},
		},
		{
			name: "returns error when repository fails",
			input: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1},
			},
			repoErr:        errors.New("db"),
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyServicesInput{
				ServiceIDs: []int32{1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUpdateCalled := false
			var calledInput UpdatePropertyServicesInput

			repo := &mockPropertyRepository{
				updatePropertyServicesFunc: func(ctx context.Context, propertyUUID string, input UpdatePropertyServicesInput) error {
					repoUpdateCalled = true
					calledInput = input
					return tt.repoErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			err := svc.UpdateServices(context.Background(), validUUID, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				// Optional: Check if validation error is correctly bubbled up
				if tt.name == "validateServiceIDs fails (service_id = 0)" || tt.name == "validateServiceIDs fails (negative service_id)" {
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
