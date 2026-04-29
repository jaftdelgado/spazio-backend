---
name: swagger-docs
description: >
  Documents Spazio Backend REST API endpoints with Swagger/OpenAPI annotations for Go + Gin.
  Use this skill whenever the user asks to document a handler, add Swagger annotations,
  write @Summary/@Param/@Success comments for Gin, generate module documentation,
  or update the OpenAPI contract for any backend endpoint.
  Also applies when swaggo, gin-swagger, swag generate, or any task related to keeping
  documentation in sync with code changes is mentioned.
---

# Spazio Swagger Docs Skill

Generates and maintains industry-standard Swagger/OpenAPI annotations for the Spazio backend (Go + Gin + pgx + swaggo).

> **All documentation must be written in English** — summaries, descriptions, param descriptions, struct comments, and field examples.

## Project Context

- **Stack**: Go, Gin, pgx v5, swaggo/swag
- **Architecture**: Vertical slices per domain module
- **Active modules**: `properties`, `services`, `catalogs`, `clauses`, `locations`
- **Each module contains**: `handler.go`, `service.go`, `repository.go`, `model.go`, `module.go`
- **Responses**: always JSON
- **Base path**: `/`
- **Local host**: `localhost:8080`

---

## Before Writing Anything

Read the module's `model.go` and `handler.go` first. Check `model.go` for request/response structs, and `handler.go` for registered routes, validations, and HTTP status codes used. Only look at `repository.go` if the contract is unclear.

---

## Annotation Format

Place the godoc block **immediately above** the handler function:

```go
// FunctionName godoc
// @Summary      Short imperative title (≤10 words)
// @Description  Full description of behavior, including relevant business rules
// @Tags         ModuleName
// @Accept       json
// @Produce      json
// @Param        id       path     int                 true  "Resource ID"
// @Param        request  body     CreateXxxRequest    true  "Creation payload"
// @Success      200      {object} XxxResponse               "Resource found"
// @Success      201      {object} XxxResponse               "Resource created"
// @Failure      400      {object} shared.ErrorResponse      "Invalid input"
// @Failure      404      {object} shared.ErrorResponse      "Not found"
// @Failure      500      {object} shared.ErrorResponse      "Internal error"
// @Router       /route/{id} [get]
func (h *Handler) FunctionName(c *gin.Context) {
```

### Module Tags

| Module     | Swagger Tag | Route prefix  |
| ---------- | ----------- | ------------- |
| properties | Properties  | `/properties` |
| services   | Services    | `/services`   |
| catalogs   | Catalogs    | `/catalogs`   |
| clauses    | Clauses     | `/clauses`    |
| locations  | Locations   | `/locations`  |

---

## Model Structs

Define request/response structs in `model.go` with `json` tags and `example` annotations. Swaggo uses these to generate sample values in Swagger UI.

```go
// CreatePropertyRequest holds the data needed to register a new property.
type CreatePropertyRequest struct {
    Title       string  `json:"title"        example:"Apartment in Xalapa"`
    Description string  `json:"description"  example:"Spacious apartment with park view"`
    Price       float64 `json:"price"        example:"3500.00"`
    TypeID      int     `json:"type_id"      example:"1"`
}

// PropertyResponse represents a serialized property for the client.
type PropertyResponse struct {
    ID        int     `json:"id"         example:"42"`
    Title     string  `json:"title"      example:"Apartment in Xalapa"`
    Price     float64 `json:"price"      example:"3500.00"`
    CreatedAt string  `json:"created_at" example:"2025-01-15T10:00:00Z"`
}
```

Keep Request and Response structs clearly separated. Mark optional fields with `omitempty` and set their `@Param` as `false`.

---

## Shared Error Response

`internal/shared/` must have this struct (create it if missing):

```go
// ErrorResponse is the standard error structure for API responses.
type ErrorResponse struct {
    Code    int    `json:"code"    example:"404"`
    Message string `json:"message" example:"Resource not found"`
}
```

Reference it as `shared.ErrorResponse` in all `@Failure` annotations.

---

## Parameter Reference

```go
// Path param
// @Param id path int true "Property ID"

// Optional query params
// @Param page  query int false "Page number"           default(1)
// @Param limit query int false "Results per page"      default(20)

// Request body
// @Param request body CreateXxxRequest true "Resource payload"

// Auth header (if applicable)
// @Param Authorization header string true "Bearer {token}"
```

---

## Global Annotations (main.go)

```go
// @title           Spazio API
// @version         1.0
// @description     REST API for real estate management — properties, clients, operations, and payments.
// @contact.name    Spazio Team
// @license.name    Proprietary
// @host            localhost:8080
// @BasePath        /
// @schemes         http https
```

---

## Regenerating Docs

After adding or modifying annotations, remind the user to regenerate:

```bash
swag init -g cmd/api/main.go --output docs
# or, if the entry point is at the root:
swag init -g main.go --output docs
```

---

## Endpoint Quality Checklist

- [ ] `@Summary` is ≤10 words, imperative ("Get property by ID", not "This endpoint gets...")
- [ ] `@Description` covers behavior and any relevant business rules
- [ ] Each `@Param` has: name, location, type, required/optional, description
- [ ] `@Success` uses the correct HTTP code (200 for GET/PUT, 201 for POST, 204 for DELETE with no body)
- [ ] `@Failure` covers at least 400, 404 (when applicable), and 500
- [ ] `@Router` matches the registered route exactly (including leading `/` and `{id}` params)
- [ ] Referenced structs exist in `model.go` with `example` tags
- [ ] Module tag is consistent with the naming table above

---

## Go ↔ Swagger Type Reference

| Go type     | Swagger type |
| ----------- | ------------ |
| `int`       | `integer`    |
| `float64`   | `number`     |
| `string`    | `string`     |
| `bool`      | `boolean`    |
| `time.Time` | `string`     |
| `[]XxxType` | `array`      |
| struct      | `object`     |

For arrays: `{array} []PropertyResponse`

---

## Reference Files

- `references/pagination.md` — Standard pattern for paginated endpoints
- `references/domain-entities.md` — Domain entity summary for business context

Read these when the endpoint involves pagination or when domain context is needed.
