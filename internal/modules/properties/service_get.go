package properties

import (
	"context"
	"fmt"
)

func (s *service) ListProperties(ctx context.Context, input ListPropertiesInput) (ListPropertiesResult, error) {
	items, total, err := s.repository.ListProperties(ctx, input)
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

func (s *service) GetProperty(ctx context.Context, propertyUUID string) (GetPropertyResult, error) {
	result, err := s.repository.GetProperty(ctx, propertyUUID)
	if err != nil {
		return GetPropertyResult{}, fmt.Errorf("get property: %w", err)
	}

	return result, nil
}

func (s *service) GetFullProperty(ctx context.Context, propertyUUID string) (GetPropertyFullResult, error) {
	result, err := s.repository.GetFullProperty(ctx, propertyUUID)
	if err != nil {
		return GetPropertyFullResult{}, fmt.Errorf("get full property: %w", err)
	}

	return result, nil
}

func resolvePropertiesTotalPages(total int64, pageSize int32) int32 {
	if total == 0 {
		return 0
	}

	return int32((total + int64(pageSize) - 1) / int64(pageSize))
}
