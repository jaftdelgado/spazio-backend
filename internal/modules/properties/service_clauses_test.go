package properties

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestService_GetClauses tests the GetClauses service method in table-driven format.
func TestService_GetClauses(t *testing.T) {
	tests := []struct {
		name         string
		propertyUUID string
		repoResult   GetPropertyClausesResult
		repoErr      error
		wantData     []PropertyClauseData
		wantErr      bool
		wantErrType  error
	}{
		{
			name:         "repository returns data",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			repoResult: GetPropertyClausesResult{
				Data: []PropertyClauseData{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
					{ClauseID: 2, IntegerValue: ptrInt32(5)},
				},
			},
			wantData: []PropertyClauseData{
				{ClauseID: 1, BooleanValue: ptrBool(true)},
				{ClauseID: 2, IntegerValue: ptrInt32(5)},
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:         "repository returns empty data",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			repoResult: GetPropertyClausesResult{
				Data: []PropertyClauseData{},
			},
			wantData: []PropertyClauseData{},
			repoErr:  nil,
			wantErr:  false,
		},
		{
			name:         "repository returns property not found",
			propertyUUID: "invalid-uuid",
			repoResult:   GetPropertyClausesResult{},
			repoErr:      ErrPropertyNotFound,
			wantErr:      true,
			wantErrType:  ErrPropertyNotFound,
		},
		{
			name:         "repository returns generic error",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			repoResult:   GetPropertyClausesResult{},
			repoErr:      errors.New("database connection failed"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getPropertyClausesFunc: func(ctx context.Context, uuid string) (GetPropertyClausesResult, error) {
					return tt.repoResult, tt.repoErr
				},
			}

			storage := &mockPropertyPhotoStorage{}
			svc := NewService(repo, storage)

			result, err := svc.GetClauses(context.Background(), tt.propertyUUID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
					t.Fatalf("error type: got %v, want %v", err, tt.wantErrType)
				}
				return
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if result.Data == nil {
					t.Fatal("expected non-nil data slice")
				}

				if !reflect.DeepEqual(result.Data, tt.wantData) {
					t.Fatalf("data mismatch: got %#v want %#v", result.Data, tt.wantData)
				}
			}
		})
	}
}

// TestService_UpdateClauses tests the UpdateClauses service method.
func TestService_UpdateClauses(t *testing.T) {
	tests := []struct {
		name           string
		propertyUUID   string
		input          UpdatePropertyClausesInput
		repoErr        error
		wantErr        bool
		wantRepoCalled bool
		wantInput      UpdatePropertyClausesInput
		validateFn     func(t *testing.T, err error)
	}{
		{
			name:         "valid clauses, repository returns nil",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
		},
		{
			name:         "empty clauses, repository called",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{},
			},
			repoErr:        nil,
			wantErr:        false,
			wantRepoCalled: true,
			wantInput: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{},
			},
		},
		{
			name:         "returns validation error when clause id is invalid",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 0},
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
			validateFn: func(t *testing.T, err error) {
				var validationErr ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if validationErr.Message != "clauses[0].clause_id must be greater than 0" {
					t.Fatalf("error message: got %q, want %q", validationErr.Message, "clauses[0].clause_id must be greater than 0")
				}
			},
		},
		{
			name:         "returns validation error when clause id does not exist",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 999},
				},
			},
			repoErr:        nil,
			wantErr:        true,
			wantRepoCalled: false,
			validateFn: func(t *testing.T, err error) {
				var validationErr ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
			},
		},
		{
			name:         "returns error when property is not found",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
			repoErr:        ErrPropertyNotFound,
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
			validateFn: func(t *testing.T, err error) {
				if !errors.Is(err, ErrPropertyNotFound) {
					t.Fatalf("expected ErrPropertyNotFound, got %v", err)
				}
			},
		},
		{
			name:         "returns error when repository fails",
			propertyUUID: "123e4567-e89b-12d3-a456-426614174000",
			input: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
			repoErr:        errors.New("transaction failed"),
			wantErr:        true,
			wantRepoCalled: true,
			wantInput: UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{ClauseID: 1, BooleanValue: ptrBool(true)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoUpdateCalled := false
			var calledInput UpdatePropertyClausesInput

			repo := &mockPropertyRepository{
				getClauseValueTypesFunc: func(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
					// Return valid mappings for test clauses
					result := make(map[int32]int32)
					for _, id := range clauseIDs {
						switch id {
						case 1:
							result[id] = ClauseValueTypeBoolean
						case 2:
							result[id] = ClauseValueTypeRange
						}
						// ID 999 will not be in the map to trigger validation error
					}
					return result, nil
				},
				updatePropertyClausesFunc: func(ctx context.Context, uuid string, input UpdatePropertyClausesInput) error {
					repoUpdateCalled = true
					calledInput = input
					return tt.repoErr
				},
			}

			storage := &mockPropertyPhotoStorage{}
			svc := NewService(repo, storage)

			err := svc.UpdateClauses(context.Background(), tt.propertyUUID, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.validateFn != nil {
					tt.validateFn(t, err)
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

// TestService_UpdateClauses_ValidateClauseValues_RangeValidation tests range value constraints.
func TestService_UpdateClauses_ValidateClauseValues_RangeValidation(t *testing.T) {
	tests := []struct {
		name     string
		minValue float64
		maxValue float64
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid range: min < max",
			minValue: 1.0,
			maxValue: 3.0,
			wantErr:  false,
		},
		{
			name:     "valid range: min == max",
			minValue: 2.0,
			maxValue: 2.0,
			wantErr:  false,
		},
		{
			name:     "invalid range: min > max",
			minValue: 5.0,
			maxValue: 1.0,
			wantErr:  true,
			errMsg:   "min_value to be less than or equal to max_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPropertyRepository{
				getClauseValueTypesFunc: func(ctx context.Context, clauseIDs []int32) (map[int32]int32, error) {
					return map[int32]int32{2: ClauseValueTypeRange}, nil
				},
				updatePropertyClausesFunc: func(ctx context.Context, uuid string, input UpdatePropertyClausesInput) error {
					return nil
				},
			}

			storage := &mockPropertyPhotoStorage{}
			svc := NewService(repo, storage)

			input := UpdatePropertyClausesInput{Actor: ActorContext{UserID: 1, RoleID: RoleAdminID},
				Clauses: []CreatePropertyClauseInput{
					{
						ClauseID: 2,
						MinValue: &tt.minValue,
						MaxValue: &tt.maxValue,
					},
				},
			}

			err := svc.UpdateClauses(context.Background(), "123e4567-e89b-12d3-a456-426614174000", input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var validationErr ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
