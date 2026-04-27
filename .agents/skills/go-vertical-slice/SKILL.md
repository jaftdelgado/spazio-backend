---
name: go-vertical-slice
description: >
  Best practices for structuring, writing, and refactoring Go backend modules
  using a vertical slice architecture with Gin and pgx/sqlc. Use this skill
  whenever the user is creating a new module, refactoring an existing one,
  reviewing Go backend code, or asking about handler/service/repository patterns
  in Go. Trigger even for partial tasks like "add a new endpoint", "split this
  use case", or "clean up this handler".
---

# Go Vertical Slice — Best Practices

Reference architecture based on a production Go backend using Gin + pgx + sqlc.
Each module lives at `internal/modules/<name>/` and contains exactly these files:

```
internal/modules/<name>/
├── handler.go     — HTTP layer, routing, input parsing, validation
├── service.go     — Business logic, orchestration
├── repository.go  — Database access via sqlc-generated queries
├── model.go       — Types, interfaces, input/output structs
└── module.go      — Dependency wiring (constructor only, no logic)
```

---

## model.go

Define all types, interfaces, and input/output structs here.

**Rules:**

- One input type per use case — do not reuse a generic input struct across multiple operations.
- Do NOT add `form`, `json`, or `binding` struct tags to input types unless they are actually used by a binding mechanism. Unused tags are noise.
- Repository and service interfaces live here, not in their implementation files.
- Output types include a `Meta` struct when the response is a list (total, shown, optional query).

```go
// One input type per use case
type ListPopularInput struct {
    Limit int32
}

type SearchInput struct {
    Query string
    Limit int32
}

// Meta for list responses
type ListServicesMeta struct {
    Total int64   `json:"total"`
    Shown int     `json:"shown"`
    Query *string `json:"query,omitempty"`
}

// Interfaces always in model.go
type ServicesRepository interface {
    ListPopularServices(ctx context.Context, limit int32) ([]Service, int64, error)
    SearchServices(ctx context.Context, query string, limit int32) ([]Service, int64, error)
}

type ServicesService interface {
    ListPopularServices(ctx context.Context, input ListPopularInput) (ListServicesResult, error)
    SearchServices(ctx context.Context, input SearchInput) (ListServicesResult, error)
}
```

---

## handler.go

The handler owns:

1. Parsing raw HTTP input
2. Sanitizing input (`strings.TrimSpace`)
3. Routing decisions (which use case to call)
4. Calling the correct service method
5. Writing the HTTP response

**Rules:**

- Sanitize query params once in the handler. Do NOT re-sanitize in the service.
- The handler decides which use case to invoke — not the service.
- Use package-level constants for default limits. Name them clearly. Do not expose them to `shared`.
- Helper functions (`resolveLimit`, `resolveDefaultLimit`, `validate*`) stay in `handler.go` as unexported functions.
- Do NOT use `strings.TrimSpace` inside helper functions if the input was already trimmed at the call site.

```go
const (
    // Limits applied when no explicit limit is provided by the caller.
    defaultPopularLimit = 12
    defaultSearchLimit  = 10
)

func (h *Handler) listServices(c *gin.Context) {
    query := strings.TrimSpace(c.Query("q"))
    limit, err := resolveLimit(c.Query("limit"), resolveDefaultLimit(query))
    if err != nil {
        shared.BadRequest(c, err)
        return
    }

    if err := validateListServicesRequest(limit); err != nil {
        shared.BadRequest(c, err)
        return
    }

    ctx := c.Request.Context()

    var result ListServicesResult
    if query == "" {
        result, err = h.service.ListPopularServices(ctx, ListPopularInput{Limit: int32(limit)})
    } else {
        result, err = h.service.SearchServices(ctx, SearchInput{Query: query, Limit: int32(limit)})
    }
    if err != nil {
        shared.InternalError(c, "could not list services")
        return
    }

    c.JSON(http.StatusOK, result)
}

// resolveDefaultLimit returns the fallback limit based on whether a search
// query is present. Input must already be trimmed.
func resolveDefaultLimit(query string) int {
    if query == "" {
        return defaultPopularLimit
    }
    return defaultSearchLimit
}
```

---

## service.go

The service owns business logic and orchestration. It trusts its input — it does not re-sanitize or re-validate what the handler already processed.

**Rules:**

- One method per use case. Do NOT merge two distinct operations into one method with an internal `if`.
- Trust the input. Do not repeat `TrimSpace` or validation already done in the handler.
- Wrap repository errors with `fmt.Errorf("operation name: %w", err)`.
- Return zero-value structs on error, not nil.

```go
// WRONG — merges two use cases into one method
func (s *service) ListServices(ctx context.Context, input ListServicesInput) (ListServicesResult, error) {
    if input.Query == "" { ... } else { ... }
}

// CORRECT — one method per use case
func (s *service) ListPopularServices(ctx context.Context, input ListPopularInput) (ListServicesResult, error) {
    items, total, err := s.repository.ListPopularServices(ctx, input.Limit)
    if err != nil {
        return ListServicesResult{}, fmt.Errorf("list popular services: %w", err)
    }
    return ListServicesResult{
        Data: items,
        Meta: ListServicesMeta{Total: total, Shown: len(items)},
    }, nil
}
```

---

## repository.go

The repository owns all database access. It maps sqlc-generated rows to domain types.

**Rules:**

- Never issue two queries (COUNT + SELECT) when a single query with `COUNT(*) OVER()` can return both. Two separate queries introduce a race condition between the count and the fetch.
- Use `rows[0].TotalCount` with a `len(rows) == 0` guard instead of a separate count query.
- Always pre-allocate slices with `make([]T, 0, len(rows))`.
- Wrap all errors with `fmt.Errorf("operation: %w", err)`.

```go
// WRONG — two queries, race condition risk
total, err := r.queries.CountActiveServices(ctx)
rows, err  := r.queries.ListPopularServices(ctx, limit)

// CORRECT — single query with window function
func (r *repository) ListPopularServices(ctx context.Context, limit int32) ([]Service, int64, error) {
    rows, err := r.queries.ListPopularServices(ctx, limit)
    if err != nil {
        return nil, 0, fmt.Errorf("list popular services: %w", err)
    }

    services := make([]Service, 0, len(rows))
    for _, row := range rows {
        services = append(services, Service{
            ServiceID:    row.ServiceID,
            Code:         row.Code,
            Icon:         row.Icon,
            CategoryCode: row.CategoryCode,
        })
    }

    if len(rows) == 0 {
        return services, 0, nil
    }

    return services, rows[0].TotalCount, nil
}
```

---

## SQL (sqlc)

Use `COUNT(*) OVER()` as a window function to return the total count in the same query as the rows.

```sql
-- name: ListPopularServices :many
SELECT
    s.service_id,
    s.code,
    s.icon,
    sc.code AS category_code,
    COUNT(*) OVER() AS total_count
FROM services AS s
JOIN service_categories AS sc ON sc.category_id = s.category_id
LEFT JOIN property_services AS ps ON ps.service_id = s.service_id
WHERE s.is_active = true
  AND s.is_deprecated = false
GROUP BY s.service_id, s.code, s.icon, sc.code, s.sort_order
ORDER BY COUNT(ps.property_id) DESC, s.sort_order ASC
LIMIT $1;
```

**Rules:**

- `COUNT(*) OVER()` without `PARTITION BY` counts the full result set before `LIMIT` is applied — this is the correct behavior for pagination metadata.
- Always filter `is_active = true` and `is_deprecated = false` on catalog tables.

---

## module.go

Only wires dependencies. No logic.

```go
func NewModule(db *pgxpool.Pool) *Module {
    repository := NewRepository(db)
    service    := NewService(repository)
    handler    := NewHandler(service)
    return &Module{handler: handler}
}
```

---

## shared/

Use `internal/shared` only for truly cross-cutting utilities:

- HTTP response helpers (`BadRequest`, `InternalError`)
- Generic validation runner (`Validate`, `ValidationRule`)

**Do NOT put in shared:**

- Module-specific constants (default limits, page sizes)
- Business rules from a specific module
- Types that only one module uses
