package properties

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestService_ListProperties_CU12(t *testing.T) {
	tests := []struct {
		name      string
		repoItems []PropertyCardData
		repoTotal int64
		input     ListPropertiesInput
		wantStatusForced bool
	}{
		{
			name:      "CU-12: forces StatusAvailable when no status provided",
			repoItems: []PropertyCardData{{PropertyUUID: "a"}},
			repoTotal: 1,
			input:     ListPropertiesInput{Page: 1, PageSize: 20},
			wantStatusForced: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
					if tt.wantStatusForced {
						found := false
						for _, s := range input.StatusIDs {
							if s == StatusAvailable {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected StatusAvailable (2) in filters, got %v", input.StatusIDs)
						}
					}
					return tt.repoItems, tt.repoTotal, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			_, _ = svc.ListProperties(context.Background(), tt.input)
		})
	}
}

func TestService_ListProperties(t *testing.T) {
	tests := []struct {
		name            string
		repoItems       []PropertyCardData
		repoTotal       int64
		repoErr         error
		input           ListPropertiesInput
		wantErr         bool
		wantData        []PropertyCardData
		wantMeta        ListPropertiesMeta
		wantMetaSet     bool
		wantErrContains string
	}{
		{
			name:      "returns properties with pagination metadata",
			repoItems: []PropertyCardData{{PropertyUUID: "a"}, {PropertyUUID: "b"}},
			repoTotal: 45,
			input:     ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:   false,
			wantData:  []PropertyCardData{{PropertyUUID: "a"}, {PropertyUUID: "b"}},
			wantMeta: ListPropertiesMeta{
				TotalCount:  45,
				TotalPages:  3,
				CurrentPage: 1,
				PageSize:    20,
				HasNext:     true,
				HasPrev:     false,
			},
			wantMetaSet: true,
		},
		{
			name:        "returns empty properties when no results are found",
			repoItems:   []PropertyCardData{},
			repoTotal:   0,
			input:       ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{},
			wantMeta:    ListPropertiesMeta{TotalCount: 0, TotalPages: 0, CurrentPage: 1, PageSize: 20, HasNext: false, HasPrev: false},
			wantMetaSet: true,
		},
		{
			name:        "returns properties with previous and next pages available",
			repoItems:   []PropertyCardData{{PropertyUUID: "c"}},
			repoTotal:   50,
			input:       ListPropertiesInput{Page: 2, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{{PropertyUUID: "c"}},
			wantMeta:    ListPropertiesMeta{TotalCount: 50, TotalPages: 3, CurrentPage: 2, PageSize: 20, HasNext: true, HasPrev: true},
			wantMetaSet: true,
		},
		{
			name:        "returns properties with no next page on last page",
			repoItems:   []PropertyCardData{{PropertyUUID: "d"}},
			repoTotal:   20,
			input:       ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:     false,
			wantData:    []PropertyCardData{{PropertyUUID: "d"}},
			wantMeta:    ListPropertiesMeta{TotalCount: 20, TotalPages: 1, CurrentPage: 1, PageSize: 20, HasNext: false, HasPrev: false},
			wantMetaSet: true,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			input:           ListPropertiesInput{Page: 1, PageSize: 20},
			wantErr:         true,
			wantErrContains: "list properties:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
					if tt.repoErr != nil {
						return nil, 0, tt.repoErr
					}
					return tt.repoItems, tt.repoTotal, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.ListProperties(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Data == nil {
				t.Fatal("expected non-nil data slice")
			}

			if !reflect.DeepEqual(result.Data, tt.wantData) {
				t.Fatalf("data mismatch: got %#v want %#v", result.Data, tt.wantData)
			}

			if tt.wantMetaSet {
				if !reflect.DeepEqual(result.Meta, tt.wantMeta) {
					t.Fatalf("meta mismatch: got %#v want %#v", result.Meta, tt.wantMeta)
				}
			}
		})
	}
}

func TestService_GetProperty(t *testing.T) {
	tests := []struct {
		name            string
		repoResult      GetPropertyResult
		repoErr         error
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:       "returns property when repository returns result",
			repoResult: GetPropertyResult{Data: GetPropertyData{PropertyUUID: "id", OwnerID: 100}},
			wantErr:    false,
		},
		{
			name:      "returns error when property is not found",
			repoErr:   ErrPropertyNotFound,
			wantErr:   true,
			wantErrIs: ErrPropertyNotFound,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "get property:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyFunc: func(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
					if tt.repoErr != nil {
						return GetPropertyResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetProperty(context.Background(), "uuid")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Data.OwnerID != 0 {
				t.Errorf("OwnerID: got %d, want 0 (CU-12 Privacy)", result.Data.OwnerID)
			}
		})
	}
}

func TestService_GetFullProperty(t *testing.T) {
	tests := []struct {
		name            string
		repoResult      GetPropertyFullResult
		repoErr         error
		wantErr         bool
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:       "returns full property when repository returns result",
			repoResult: GetPropertyFullResult{Data: GetPropertyFullData{PropertyUUID: "id", OwnerID: 100}},
			wantErr:    false,
		},
		{
			name:      "returns error when property is not found",
			repoErr:   ErrPropertyNotFound,
			wantErr:   true,
			wantErrIs: ErrPropertyNotFound,
		},
		{
			name:            "returns error when repository fails",
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "get full property:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getFullPropertyFunc: func(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
					if tt.repoErr != nil {
						return GetPropertyFullResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetFullProperty(context.Background(), "uuid")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrIs)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Data.OwnerID != 0 {
				t.Errorf("OwnerID: got %d, want 0 (CU-12 Privacy)", result.Data.OwnerID)
			}
		})
	}
}

func TestService_GetPropertyHistory_CU18(t *testing.T) {
	tests := []struct {
		name            string
		propertyUUID    string
		userID          int32
		roleID          int32
		mockOwnerID     int32
		mockOwnerErr    error
		mockHistory     []PropertyStatusHistoryData
		mockHistoryErr  error
		wantErr         bool
		wantErrContains string
		wantDataLen     int
	}{
		{
			name:         "Admin can see history of any property",
			propertyUUID: "prop-1",
			userID:       99, // Not the owner
			roleID:       RoleAdminID,
			mockHistory: []PropertyStatusHistoryData{
				{HistoryID: 1, PreviousStatusName: "Draft", NewStatusName: "Available"},
			},
			wantDataLen: 1,
		},
		{
			name:         "Owner can see history of their own property",
			propertyUUID: "prop-1",
			userID:       10,
			roleID:       RoleClientID,
			mockOwnerID:  10,
			mockHistory: []PropertyStatusHistoryData{
				{HistoryID: 1}, {HistoryID: 2},
			},
			wantDataLen: 2,
		},
		{
			name:            "Non-owner is forbidden from seeing history",
			propertyUUID:    "prop-1",
			userID:          20,
			roleID:          RoleClientID,
			mockOwnerID:     10,
			wantErr:         true,
			wantErrContains: "forbidden",
		},
		{
			name:            "Returns error if property not found during ownership check",
			propertyUUID:    "missing",
			userID:          10,
			roleID:          RoleClientID,
			mockOwnerErr:    ErrPropertyNotFound,
			wantErr:         true,
			wantErrContains: ErrPropertyNotFound.Error(),
		},
		{
			name:            "Returns error if repository fails to list history",
			propertyUUID:    "prop-1",
			userID:          10,
			roleID:          RoleAdminID,
			mockHistoryErr:  errors.New("db error"),
			wantErr:         true,
			wantErrContains: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyOwnerByUUIDFunc: func(ctx context.Context, propertyUUID string) (int32, error) {
					return tt.mockOwnerID, tt.mockOwnerErr
				},
				listPropertyStatusHistoryFunc: func(ctx context.Context, propertyUUID string) ([]PropertyStatusHistoryData, error) {
					return tt.mockHistory, tt.mockHistoryErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetPropertyHistory(context.Background(), tt.propertyUUID, tt.userID, tt.roleID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error message: got %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Data) != tt.wantDataLen {
				t.Fatalf("data length: got %d, want %d", len(result.Data), tt.wantDataLen)
			}
		})
	}
}

func TestResolvePropertiesTotalPages(t *testing.T) {
	if got := resolvePropertiesTotalPages(0, 20); got != 0 {
		t.Fatalf("resolvePropertiesTotalPages(0, 20) = %d, want 0", got)
	}
	if got := resolvePropertiesTotalPages(20, 20); got != 1 {
		t.Fatalf("resolvePropertiesTotalPages(20, 20) = %d, want 1", got)
	}
	if got := resolvePropertiesTotalPages(21, 20); got != 2 {
		t.Fatalf("resolvePropertiesTotalPages(21, 20) = %d, want 2", got)
	}
	if got := resolvePropertiesTotalPages(40, 20); got != 2 {
		t.Fatalf("resolvePropertiesTotalPages(40, 20) = %d, want 2", got)
	}
	if got := resolvePropertiesTotalPages(41, 20); got != 3 {
		t.Fatalf("resolvePropertiesTotalPages(41, 20) = %d, want 3", got)
	}
	if got := resolvePropertiesTotalPages(1, 20); got != 1 {
		t.Fatalf("resolvePropertiesTotalPages(1, 20) = %d, want 1", got)
	}
}
