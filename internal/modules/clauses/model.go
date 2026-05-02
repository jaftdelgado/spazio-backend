package clauses

import "context"

// ClauseValueType defines the value type exposed to clients.
type ClauseValueType struct {
	Code string `json:"code" example:"boolean"`
}

// Clause is a catalog item exposed by the clauses endpoint.
type Clause struct {
	ClauseID  int32           `json:"clause_id" example:"1"`
	Code      string          `json:"code" example:"pets_allowed"`
	ValueType ClauseValueType `json:"value_type"`
	SortOrder int32           `json:"sort_order" example:"10"`
}

// ListClausesInput defines filters for listing clauses.
type ListClausesInput struct {
	ModalityID int32
	Page       int32
	PageSize   int32
}

// SearchClausesInput defines filters for searching clauses.
type SearchClausesInput struct {
	ModalityID int32
	Query      string
	Page       int32
	PageSize   int32
}

// ListClausesMeta defines metadata returned with clauses catalog results.
type ListClausesMeta struct {
	Total      int64   `json:"total"`
	Page       int32   `json:"page"`
	PageSize   int32   `json:"page_size"`
	TotalPages int32   `json:"total_pages"`
	Query      *string `json:"query,omitempty"`
}

// ListClausesResult is the response payload returned by clauses use cases.
type ListClausesResult struct {
	Data []Clause        `json:"data"`
	Meta ListClausesMeta `json:"meta"`
}

// ClausesRepository defines persistence operations for the clauses catalog.
type ClausesRepository interface {
	ListClauses(ctx context.Context, modalityID, pageSize, pageOffset int32) ([]Clause, int64, error)
	SearchClauses(ctx context.Context, modalityID int32, query string, pageSize, pageOffset int32) ([]Clause, int64, error)
}

// ClausesService defines business operations for the clauses catalog.
type ClausesService interface {
	ListClauses(ctx context.Context, input ListClausesInput) (ListClausesResult, error)
	SearchClauses(ctx context.Context, input SearchClausesInput) (ListClausesResult, error)
}
