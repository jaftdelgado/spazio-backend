package services

import "context"

// Service is a catalog item exposed by the services endpoint.
type Service struct {
	ServiceID    int32  `json:"service_id"`
	Code         string `json:"code"`
	Icon         string `json:"icon"`
	CategoryCode string `json:"category_code"`
}

// ListServicesInput defines filters for the service catalog list operation.
type ListServicesInput struct {
	Query string `form:"q"`
	Limit int32  `form:"limit"`
}

// ListServicesMeta defines metadata returned with service catalog results.
type ListServicesMeta struct {
	Total int64   `json:"total"`
	Shown int     `json:"shown"`
	Query *string `json:"query,omitempty"`
}

// ListServicesResult is the response payload returned by the services use case.
type ListServicesResult struct {
	Data []Service        `json:"data"`
	Meta ListServicesMeta `json:"meta"`
}

// ServicesRepository defines persistence operations for the services catalog.
type ServicesRepository interface {
	ListPopularServices(ctx context.Context, limit int32) ([]Service, int64, error)
	SearchServices(ctx context.Context, query string, limit int32) ([]Service, int64, error)
}

// ServicesService defines business operations for the services catalog.
type ServicesService interface {
	ListServices(ctx context.Context, input ListServicesInput) (ListServicesResult, error)
}
