package services

import "context"

// Service is a catalog item exposed by the services endpoint.
type Service struct {
	ServiceID    int32  `json:"service_id" example:"1"`
	Code         string `json:"code" example:"wifi"`
	Icon         string `json:"icon" example:"wifi"`
	CategoryCode string `json:"category_code" example:"basic"`
}

// ListPopularInput defines filters for listing popular services.
type ListPopularInput struct {
	CategoryID int32
	Page       int32
	PageSize   int32
}

// SearchInput defines filters for searching services.
type SearchInput struct {
	Query      string
	CategoryID int32
	Page       int32
	PageSize   int32
}

// ListServicesMeta defines metadata returned with service catalog results.
type ListServicesMeta struct {
	Total      int64   `json:"total"`
	Shown      int     `json:"shown"`
	Page       int32   `json:"page"`
	PageSize   int32   `json:"page_size"`
	TotalPages int32   `json:"total_pages"`
	Query      *string `json:"query,omitempty"`
}

// ListServicesResult is the response payload returned by the services use case.
type ListServicesResult struct {
	Data []Service        `json:"data"`
	Meta ListServicesMeta `json:"meta"`
}

// ServicesRepository defines persistence operations for the services catalog.
type ServicesRepository interface {
	ListPopularServices(ctx context.Context, input ListPopularInput) ([]Service, int64, error)
	SearchServices(ctx context.Context, input SearchInput) ([]Service, int64, error)
}

// ServicesService defines business operations for the services catalog.
type ServicesService interface {
	ListPopularServices(ctx context.Context, input ListPopularInput) (ListServicesResult, error)
	SearchServices(ctx context.Context, input SearchInput) (ListServicesResult, error)
}
