package properties

import (
	"context"
	"errors"
	"fmt"
)

const (
	ModalitySale  int32 = 1
	ModalityRent  int32 = 2
	ModalityMixed int32 = 3
)

type modalityRequirements struct {
	RequiresSale bool
	RequiresRent bool
}

type service struct {
	repository PropertyRepository
}

// NewService builds a property service implementation.
func NewService(repository PropertyRepository) PropertyService {
	return &service{repository: repository}
}

func (s *service) CreateProperty(ctx context.Context, input CreatePropertyInput) (CreatePropertyResult, error) {
	subtype, err := s.repository.GetPropertySubtype(ctx, input.PropertyTypeID)
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("get property subtype: %w", err)
	}
	input.Subtype = subtype

	if err := validateSubtypePayload(input); err != nil {
		return CreatePropertyResult{}, ValidationError{Message: err.Error()}
	}

	requirements, err := resolveModalityRequirements(input.ModalityID)
	if err != nil {
		return CreatePropertyResult{}, ValidationError{Message: err.Error()}
	}

	if err := validateModalityPricing(input, requirements); err != nil {
		return CreatePropertyResult{}, ValidationError{Message: err.Error()}
	}

	if len(input.RentPrices) > 0 {
		if err := s.validateAllowedPeriods(ctx, input); err != nil {
			return CreatePropertyResult{}, err
		}
	}

	if len(input.Clauses) > 0 {
		if err := s.validateClauseValues(ctx, input.Clauses); err != nil {
			return CreatePropertyResult{}, err
		}
	}

	result, err := s.repository.CreateProperty(ctx, input)
	if err != nil {
		return CreatePropertyResult{}, fmt.Errorf("create property: %w", err)
	}

	return result, nil
}

func validateModalityPricing(input CreatePropertyInput, requirements modalityRequirements) error {
	if requirements.RequiresSale {
		if input.SalePrice == nil {
			return errors.New("sale_price is required for the selected modality")
		}
	} else if input.SalePrice != nil {
		return errors.New("sale_price is not allowed for the selected modality")
	}

	if requirements.RequiresRent {
		if len(input.RentPrices) == 0 {
			return errors.New("rent_prices must include at least one item for the selected modality")
		}
	} else if len(input.RentPrices) > 0 {
		return errors.New("rent_prices are not allowed for the selected modality")
	}

	return nil
}

func (s *service) validateAllowedPeriods(ctx context.Context, input CreatePropertyInput) error {
	allowedPeriods, err := s.repository.GetAllowedPeriods(ctx, input.PropertyTypeID)
	if err != nil {
		return fmt.Errorf("get allowed periods: %w", err)
	}

	for _, rentPrice := range input.RentPrices {
		if _, ok := allowedPeriods[rentPrice.PeriodID]; !ok {
			return ValidationError{
				Message: fmt.Sprintf(
					"period_id %d is not allowed for property_type_id %d",
					rentPrice.PeriodID,
					input.PropertyTypeID,
				),
			}
		}
	}

	return nil
}

func (s *service) validateClauseValues(ctx context.Context, clauses []CreatePropertyClauseInput) error {
	clauseIDs := uniqueClauseIDs(clauses)

	valueTypes, err := s.repository.GetClauseValueTypes(ctx, clauseIDs)
	if err != nil {
		return fmt.Errorf("get clause value types: %w", err)
	}

	for _, clauseID := range clauseIDs {
		if _, ok := valueTypes[clauseID]; !ok {
			return ValidationError{Message: fmt.Sprintf("clause_id %d is invalid", clauseID)}
		}
	}

	for _, clause := range clauses {
		if err := validateClauseValuePayload(clause, valueTypes[clause.ClauseID]); err != nil {
			return err
		}
	}

	return nil
}

func uniqueClauseIDs(clauses []CreatePropertyClauseInput) []int32 {
	seen := make(map[int32]struct{}, len(clauses))
	ids := make([]int32, 0, len(clauses))
	for _, clause := range clauses {
		if _, ok := seen[clause.ClauseID]; ok {
			continue
		}

		seen[clause.ClauseID] = struct{}{}
		ids = append(ids, clause.ClauseID)
	}

	return ids
}

func validateClauseValuePayload(clause CreatePropertyClauseInput, valueTypeID int32) error {
	hasBool := clause.BooleanValue != nil
	hasInt := clause.IntegerValue != nil
	hasMin := clause.MinValue != nil
	hasMax := clause.MaxValue != nil

	switch valueTypeID {
	case ClauseValueTypeBoolean:
		if hasBool && !hasInt && !hasMin && !hasMax {
			return nil
		}
		return ValidationError{Message: fmt.Sprintf("clause_id %d requires only boolean_value", clause.ClauseID)}
	case ClauseValueTypeRange:
		if !hasBool && !hasInt && hasMin && hasMax {
			if *clause.MinValue > *clause.MaxValue {
				return ValidationError{Message: fmt.Sprintf("clause_id %d requires min_value to be less than or equal to max_value", clause.ClauseID)}
			}
			return nil
		}
		return ValidationError{Message: fmt.Sprintf("clause_id %d requires min_value and max_value only", clause.ClauseID)}
	case ClauseValueTypeInteger:
		if !hasBool && hasInt && !hasMin && !hasMax {
			return nil
		}
		return ValidationError{Message: fmt.Sprintf("clause_id %d requires only integer_value", clause.ClauseID)}
	default:
		return ValidationError{Message: fmt.Sprintf("clause_id %d has an unsupported value type", clause.ClauseID)}
	}
}

func resolveModalityRequirements(modalityID int32) (modalityRequirements, error) {
	switch modalityID {
	case ModalitySale:
		return modalityRequirements{
			RequiresSale: true,
			RequiresRent: false,
		}, nil
	case ModalityRent:
		return modalityRequirements{
			RequiresSale: false,
			RequiresRent: true,
		}, nil
	case ModalityMixed:
		return modalityRequirements{
			RequiresSale: true,
			RequiresRent: true,
		}, nil
	default:
		return modalityRequirements{}, errors.New("modality_id is invalid")
	}
}
