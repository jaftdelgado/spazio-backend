package properties

import (
	"context"
	"fmt"
)

func (s *service) GetClauses(ctx context.Context, propertyUUID string) (GetPropertyClausesResult, error) {
	result, err := s.repository.GetPropertyClauses(ctx, propertyUUID)
	if err != nil {
		return GetPropertyClausesResult{}, fmt.Errorf("get property clauses: %w", err)
	}

	return result, nil
}

func (s *service) UpdateClauses(ctx context.Context, propertyUUID string, input UpdatePropertyClausesInput) error {
	if err := validateClauseInputs(input.Clauses); err != nil {
		return ValidationError{Message: err.Error()}
	}

	if len(input.Clauses) > 0 {
		if err := s.validateClauseValues(ctx, input.Clauses); err != nil {
			return err
		}
	}

	if err := s.repository.UpdatePropertyClauses(ctx, propertyUUID, input); err != nil {
		return fmt.Errorf("update property clauses: %w", err)
	}

	return nil
}
