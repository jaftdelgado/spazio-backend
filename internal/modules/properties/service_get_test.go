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
		name            string
		input           ListPropertiesInput
		repoItems       []PropertyCardData
		repoTotal       int64
		repoErr         error
		useAgentRepo    bool
		wantErr         bool
		wantErrContains string
		wantMeta        ListPropertiesMeta
	}{
		{
			name:  "uses admin repository and forces available status when no status is provided",
			input: ListPropertiesInput{Page: 1, PageSize: 20, RoleID: RoleAdminID},
			repoItems: []PropertyCardData{{
				PropertyUUID:  "a",
				CoverPhotoURL: ptrString("properties/a/photos/cover.webp"),
			}},
			repoTotal: 1,
			wantMeta:  ListPropertiesMeta{TotalCount: 1, TotalPages: 1, CurrentPage: 1, PageSize: 20, HasNext: false, HasPrev: false},
		},
		{
			name:         "uses agent repository for agent role",
			input:        ListPropertiesInput{Page: 2, PageSize: 20, RoleID: RoleAgentID, UserID: 9},
			repoItems:    []PropertyCardData{{PropertyUUID: "b", CoverPhotoURL: ptrString("properties/b/photos/cover.webp")}},
			repoTotal:    45,
			useAgentRepo: true,
			wantMeta:     ListPropertiesMeta{TotalCount: 45, TotalPages: 3, CurrentPage: 2, PageSize: 20, HasNext: true, HasPrev: true},
		},
		{
			name:            "wraps repository error",
			input:           ListPropertiesInput{Page: 1, PageSize: 20, RoleID: RoleAdminID},
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "list properties:",
		},
		{
			name:            "wraps public url generation error",
			input:           ListPropertiesInput{Page: 1, PageSize: 20, RoleID: RoleAdminID},
			repoItems:       []PropertyCardData{{PropertyUUID: "c", CoverPhotoURL: ptrString("properties/c/photos/cover.webp")}},
			repoTotal:       1,
			wantErr:         true,
			wantErrContains: "attach public cover photo urls:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminCalled := false
			agentCalled := false

			repo := &mockPropertyRepository{
				listPropertiesFunc: func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
					adminCalled = true
					if len(input.StatusIDs) != 0 {
						t.Fatalf("StatusIDs = %v, want []", input.StatusIDs)
					}
					return tt.repoItems, tt.repoTotal, tt.repoErr
				},
				listPropertiesForAgentFunc: func(ctx context.Context, input ListPropertiesInput) ([]PropertyCardData, int64, error) {
					agentCalled = true
					if input.UserID != 9 {
						t.Fatalf("UserID = %d, want 9", input.UserID)
					}
					if len(input.StatusIDs) != 1 || input.StatusIDs[0] != StatusAvailable {
						t.Fatalf("StatusIDs = %v, want [%d]", input.StatusIDs, StatusAvailable)
					}
					return tt.repoItems, tt.repoTotal, tt.repoErr
				},
			}

			storage := &mockPropertyPhotoStorage{
				publicURLFunc: func(ctx context.Context, storageKey string) (string, error) {
					if tt.name == "wraps public url generation error" {
						return "", errors.New("url failed")
					}
					return "https://cdn.example.com/" + storageKey, nil
				},
			}

			svc := NewService(repo, storage)
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

			if tt.useAgentRepo {
				if !agentCalled || adminCalled {
					t.Fatalf("expected only agent repository call, got admin=%v agent=%v", adminCalled, agentCalled)
				}
			} else if !adminCalled || agentCalled {
				t.Fatalf("expected only admin repository call, got admin=%v agent=%v", adminCalled, agentCalled)
			}

			if !reflect.DeepEqual(result.Data, tt.repoItems) {
				expected := tt.repoItems
				for i := range expected {
					if expected[i].CoverPhotoURL != nil {
						url := "https://cdn.example.com/" + *expected[i].CoverPhotoURL
						expected[i].CoverPhotoURL = &url
					}
				}
				if !reflect.DeepEqual(result.Data, expected) {
					t.Fatalf("data mismatch: got %#v want %#v", result.Data, expected)
				}
			}
			if !reflect.DeepEqual(result.Meta, tt.wantMeta) {
				t.Fatalf("meta mismatch: got %#v want %#v", result.Meta, tt.wantMeta)
			}
		})
	}
}

func TestService_GetPropertyForRole(t *testing.T) {
	tests := []struct {
		name             string
		userID           int32
		roleID           int32
		repoResult       GetPropertyResult
		repoErr          error
		assigned         bool
		assignedErr      error
		wantErr          bool
		wantErrIs        error
		wantErrContains  string
		wantRegisteredBy string
	}{
		{
			name:             "admin can view property with registered by",
			userID:           10,
			roleID:           RoleAdminID,
			repoResult:       GetPropertyResult{Data: GetPropertyData{PropertyID: 5, PropertyUUID: "uuid", RegisteredBy: "Admin User"}},
			wantRegisteredBy: "Admin User",
		},
		{
			name:             "assigned agent can view property without registered by",
			userID:           20,
			roleID:           RoleAgentID,
			repoResult:       GetPropertyResult{Data: GetPropertyData{PropertyID: 5, PropertyUUID: "uuid", RegisteredBy: "Admin User"}},
			assigned:         true,
			wantRegisteredBy: "",
		},
		{
			name:            "unassigned agent is forbidden",
			userID:          20,
			roleID:          RoleAgentID,
			repoResult:      GetPropertyResult{Data: GetPropertyData{PropertyID: 5, PropertyUUID: "uuid", RegisteredBy: "Admin User"}},
			assigned:        false,
			wantErr:         true,
			wantErrContains: "forbidden",
		},
		{
			name:            "wraps property repository error",
			userID:          10,
			roleID:          RoleAdminID,
			repoErr:         ErrPropertyNotFound,
			wantErr:         true,
			wantErrIs:       ErrPropertyNotFound,
			wantErrContains: "get property:",
		},
		{
			name:            "wraps assignment check error",
			userID:          20,
			roleID:          RoleAgentID,
			repoResult:      GetPropertyResult{Data: GetPropertyData{PropertyID: 5, PropertyUUID: "uuid", RegisteredBy: "Admin User"}},
			assignedErr:     errors.New("db"),
			wantErr:         true,
			wantErrContains: "check agent assignment:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyByUUIDFunc: func(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
					if tt.repoErr != nil {
						return GetPropertyResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
				isPropertyAssignedToAgentFunc: func(ctx context.Context, propertyID int32, agentID int32) (bool, error) {
					if propertyID != 5 || agentID != tt.userID {
						t.Fatalf("assignment check args: got propertyID=%d agentID=%d", propertyID, agentID)
					}
					return tt.assigned, tt.assignedErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetPropertyForRole(context.Background(), "uuid", tt.userID, tt.roleID)

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

			if result.Data.RegisteredBy != tt.wantRegisteredBy {
				t.Fatalf("RegisteredBy = %q, want %q", result.Data.RegisteredBy, tt.wantRegisteredBy)
			}
		})
	}
}

func TestService_GetPricesHistory(t *testing.T) {
	tests := []struct {
		name            string
		repoResult      GetPropertyPricesHistoryResult
		repoErr         error
		wantErr         bool
		wantErrContains string
	}{
		{
			name:       "returns prices history",
			repoResult: GetPropertyPricesHistoryResult{Data: []PropertyPriceHistoryData{{PriceType: "sale", Amount: 1000}}},
		},
		{
			name:            "wraps repository error",
			repoErr:         errors.New("db"),
			wantErr:         true,
			wantErrContains: "get prices history:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyPricesHistoryFunc: func(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error) {
					if tt.repoErr != nil {
						return GetPropertyPricesHistoryResult{}, tt.repoErr
					}
					return tt.repoResult, nil
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetPricesHistory(context.Background(), "uuid")

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

			if !reflect.DeepEqual(result, tt.repoResult) {
				t.Fatalf("result mismatch: got %#v want %#v", result, tt.repoResult)
			}
		})
	}
}

func TestService_GetPropertyHistory_CU18(t *testing.T) {
	tests := []struct {
		name            string
		propertyUUID    string
		mockHistory     []PropertyStatusHistoryData
		mockHistoryErr  error
		wantErr         bool
		wantErrContains string
		wantDataLen     int
	}{
		{
			name:         "returns history for property",
			propertyUUID: "prop-1",
			mockHistory: []PropertyStatusHistoryData{
				{HistoryID: 1, PreviousStatusName: "Draft", NewStatusName: "Available"},
			},
			wantDataLen: 1,
		},
		{
			name:            "returns error if repository fails to list history",
			propertyUUID:    "prop-1",
			mockHistoryErr:  errors.New("db error"),
			wantErr:         true,
			wantErrContains: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				listPropertyStatusHistoryFunc: func(ctx context.Context, propertyUUID string) ([]PropertyStatusHistoryData, error) {
					return tt.mockHistory, tt.mockHistoryErr
				},
			}

			svc := NewService(repo, &mockPropertyPhotoStorage{})
			result, err := svc.GetPropertyHistory(context.Background(), tt.propertyUUID)

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
