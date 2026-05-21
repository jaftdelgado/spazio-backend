package properties

import (
	"context"
	"errors"
	"fmt"
)

func (s *service) GetPropertyHistory(ctx context.Context, propertyUUID string) (GetPropertyHistoryResult, error) {
	data, err := s.repository.ListPropertyStatusHistory(ctx, propertyUUID)
	if err != nil {
		return GetPropertyHistoryResult{}, err
	}

	return GetPropertyHistoryResult{Data: data}, nil
}

func (s *service) ListProperties(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
	if len(input.StatusIDs) == 0 && input.RoleID == RoleAgentID {
		input.StatusIDs = []int32{StatusAvailable}
	}

	var (
		items []PropertyCardData
		total int64
		err   error
	)

	if input.RoleID == RoleAgentID {
		items, total, err = s.repository.ListPropertiesForAgent(ctx, input)
	} else {
		items, total, err = s.repository.ListProperties(ctx, input)
	}
	if err != nil {
		return ListPropertiesResult{}, fmt.Errorf("list properties: %w", err)
	}

	totalPages := resolvePropertiesTotalPages(total, input.PageSize)

	return ListPropertiesResult{
		Data: items,
		Meta: ListPropertiesMeta{
			TotalCount:  total,
			TotalPages:  totalPages,
			CurrentPage: input.Page,
			PageSize:    input.PageSize,
			HasNext:     input.Page < totalPages,
			HasPrev:     input.Page > 1,
		},
	}, nil
}

func (s *service) GetPropertyForRole(ctx context.Context, propertyUUID string, userID int32, roleID int32) (GetPropertyResult, error) {
	if roleID != RoleAdminID && roleID != RoleAgentID {
		return GetPropertyResult{}, errors.New("forbidden: unsupported role")
	}

	result, err := s.repository.GetPropertyByUUID(ctx, propertyUUID)
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("get property: %w", err)
	}

	if roleID == RoleAgentID {
		assigned, err := s.repository.IsPropertyAssignedToAgent(ctx, result.Data.PropertyID, userID)
		if err != nil {
			return GetPropertyResult{}, fmt.Errorf("check agent assignment: %w", err)
		}
		if !assigned {
			return GetPropertyResult{}, errors.New("forbidden: property not assigned to agent")
		}
		result.Data.RegisteredBy = ""
	}

	return result, nil
}

func (s *service) GetPricesHistory(ctx context.Context, propertyUUID string) (GetPropertyPricesHistoryResult, error) {
	result, err := s.repository.GetPropertyPricesHistory(ctx, propertyUUID)
	if err != nil {
		return GetPropertyPricesHistoryResult{}, fmt.Errorf("get prices history: %w", err)
	}

	return result, nil
}

func resolvePropertiesTotalPages(total int64, pageSize int32) int32 {
	if total == 0 {
		return 0
	}

	return int32((total + int64(pageSize) - 1) / int64(pageSize))
}
