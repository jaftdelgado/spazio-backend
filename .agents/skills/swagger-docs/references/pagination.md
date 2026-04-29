# Pagination Pattern — Spazio API

Use this pattern whenever an endpoint returns a list of resources.

## Structs in model.go

```go
// PaginatedResponse is the standard wrapper for paginated responses.
type PaginatedResponse[T any] struct {
    Data       []T `json:"data"`
    Total      int `json:"total"        example:"150"`
    Page       int `json:"page"         example:"1"`
    Limit      int `json:"limit"        example:"20"`
    TotalPages int `json:"total_pages"  example:"8"`
}
```

> If Go generics are unavailable due to the Go version, create a concrete struct per module:
> `PropertyListResponse`, `ServiceListResponse`, etc.

## Swagger Annotations for a Paginated Endpoint

```go
// ListProperties godoc
// @Summary      List properties
// @Description  Returns a paginated list of properties with optional filters
// @Tags         Properties
// @Produce      json
// @Param        page    query int false "Page number"           default(1)  minimum(1)
// @Param        limit   query int false "Results per page"      default(20) maximum(100)
// @Param        type_id query int false "Filter by property type"
// @Param        zone_id query int false "Filter by zone"
// @Success      200     {object} PaginatedPropertiesResponse  "Paginated list"
// @Failure      400     {object} shared.ErrorResponse         "Invalid parameters"
// @Failure      500     {object} shared.ErrorResponse         "Internal error"
// @Router       /properties [get]
func (h *Handler) ListProperties(c *gin.Context) {
```

## Standard Pagination Query Params

| Param   | Type | Default | Description         |
| ------- | ---- | ------- | ------------------- |
| `page`  | int  | 1       | Requested page (≥1) |
| `limit` | int  | 20      | Results per page    |

## Extracting Params in the Handler

```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
if page < 1 { page = 1 }
if limit < 1 || limit > 100 { limit = 20 }
```
