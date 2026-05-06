---
name: golang-testing
description: >
  Testing strategy and implementation skill for the Spazio backend (Go + Gin + pgx/v5 + sqlc + Cloudflare R2).
  Use this skill whenever the user wants to: write tests for any Spazio module, analyze test coverage gaps,
  plan a testing strategy for a handler/service/repository, create table-driven tests, mock interfaces defined
  in model.go, or test Gin HTTP handlers with httptest. Triggers on: "write tests for", "add tests", "test coverage",
  "unit test", "mock this", "test the service", "test the handler", "gap analysis", or any request involving
  testing a vertical slice module (handler.go / service.go / repository.go / model.go).
---

# Spazio Go Testing Skill

## Project Context

**Stack:** Go 1.26+ · Gin 1.12 · pgx/v5 · sqlc (generated code in `internal/sqlcgen/`) · JWT (Supabase) · Cloudflare R2 (AWS SDK v2)  
**Architecture:** Vertical slice — each module lives in `internal/modules/<domain>/` with exactly:

```
handler.go     ← Gin routes, request parsing, response shaping
service.go     ← Business rules, orchestration, error wrapping
repository.go  ← pgx/sqlcgen queries
model.go       ← Interfaces (Repository + Service) + DTOs + domain structs
module.go      ← Dependency wiring (NewModule)
```

**No external test libraries** — use only Go stdlib `testing` + `net/http/httptest`.  
`go-playground/validator/v10` is available (Gin uses it internally).  
`jackc/pgerrcode` is available for PostgreSQL error codes.

---

## Step 1 — Scan Before You Plan

Before writing any test, read all files in the target module:

```
internal/modules/<domain>/model.go       ← interfaces = what to mock
internal/modules/<domain>/service.go     ← business logic = unit test targets
internal/modules/<domain>/handler.go     ← HTTP layer = httptest targets
internal/modules/<domain>/repository.go  ← data access = mock or skip
internal/shared/                         ← shared error helpers
```

Then check for existing tests:

```bash
find internal/modules/<domain> -name "*_test.go"
```

---

## Step 2 — Module Summary Output

Produce this before any implementation:

1. **What the module does** (1–2 sentences)
2. **Files with existing tests** (list or "none")
3. **Files missing tests** (list)
4. **Priority classification** per function:
   - `HIGH` — business logic, branching, error paths, validation
   - `MEDIUM` — response shaping, pagination/filtering defaults
   - `LOW` — trivial pass-through, single-query repositories
   - `SKIP` — generated sqlc code, module wiring
5. **Unit vs Integration split**
6. **Coverage verdict**: sufficient / partial / insufficient

---

## Step 3 — Testing Strategy by Layer

### Handler Tests (`handler_test.go`)

Use `httptest.NewRecorder` + `gin.New()`. Never spin up a real server.

**What to test:**

- HTTP method and route binding
- URL parameter extraction and validation (missing, malformed, wrong type)
- Request body binding (missing fields, wrong types, invalid JSON)
- Status codes: 200, 201, 400, 404, 500
- Response body shape (JSON keys present, correct types)
- `shared.BadRequest` / `shared.NotFound` / `shared.InternalError` paths

**Pattern:**

```go
func TestHandler_GetSomething(t *testing.T) {
    tests := []struct {
        name       string
        param      string
        mockResult *SomethingResult
        mockErr    error
        wantStatus int
    }{
        {"valid id", "123", &SomethingResult{...}, nil, http.StatusOK},
        {"service error", "999", nil, errors.New("db error"), http.StatusInternalServerError},
        {"not found", "0", nil, ErrNotFound, http.StatusNotFound},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockSvc := &MockSomethingService{
                GetFunc: func(ctx context.Context, id int32) (*SomethingResult, error) {
                    return tt.mockResult, tt.mockErr
                },
            }
            h := NewHandler(mockSvc)
            w := httptest.NewRecorder()
            c, _ := gin.CreateTestContext(w)
            c.Params = gin.Params{{Key: "id", Value: tt.param}}
            h.getSomething(c)
            if w.Code != tt.wantStatus {
                t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
            }
        })
    }
}
```

### Service Tests (`service_test.go`)

Mock the `Repository` interface defined in `model.go`. Test **behavior**, not SQL.

**What to test:**

- Happy path: correct result returned
- Repository error → correct error wrapping
- Input validation or transformation before repository call
- Business rules: conditions, guards, defaults
- Result shaping: `ListXxxResult{Data: ...}` wrapping

**Mock pattern (no external lib):**

```go
type mockCatalogsRepository struct {
    listModalitiesFunc func(ctx context.Context) ([]Modality, error)
}

func (m *mockCatalogsRepository) ListModalities(ctx context.Context) ([]Modality, error) {
    return m.listModalitiesFunc(ctx)
}
// implement all interface methods — unused ones return zero values
```

**Table-driven service test:**

```go
func TestService_ListModalities(t *testing.T) {
    tests := []struct {
        name     string
        repoData []Modality
        repoErr  error
        wantLen  int
        wantErr  bool
    }{
        {"returns all items", []Modality{{1, "Rent"}, {2, "Sale"}}, nil, 2, false},
        {"empty catalog", []Modality{}, nil, 0, false},
        {"repository error", nil, errors.New("db down"), 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockCatalogsRepository{
                listModalitiesFunc: func(ctx context.Context) ([]Modality, error) {
                    return tt.repoData, tt.repoErr
                },
            }
            svc := NewService(repo)
            result, err := svc.ListModalities(context.Background())
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if len(result.Data) != tt.wantLen {
                t.Errorf("got %d items, want %d", len(result.Data), tt.wantLen)
            }
        })
    }
}
```

### Repository Tests

**Default: skip unit tests** — repositories are thin wrappers over sqlcgen.

**Test only if** the repository contains:

- Non-trivial filtering or pagination logic
- Error translation (e.g., `pgerrcode` → domain error)
- Data transformation before returning

If needed, use an in-memory or test DB, never mock pgx directly.

---

## Step 4 — Spazio-Specific Edge Cases

Always consider these for every module:

### IDs and Parameters

- `id = 0` or negative
- Non-integer where integer expected
- UUID format invalid or empty string
- Entity not found (→ `shared.NotFound`)

### Pagination (when present)

- `page = 0` → should default to 1
- `page_size = 0` → should use default
- `page_size` > max allowed
- Negative values

### Soft Deletes

- Query for deleted entity → 404, not 500
- Restore of non-deleted entity
- List endpoint must exclude `deleted_at IS NOT NULL`

### Optional Fields

- `*string` icon field: nil vs empty string vs valid value
- Nullable foreign keys: missing vs zero value

### Catalog-specific (e.g., `ListRentPeriods`)

- `propertyTypeID = 0` → invalid
- `propertyTypeID` with no associated periods → empty `Data`, not error
- Valid ID → correct filtered result

### External Dependencies (R2 / Storage)

- Upload success → URL returned correctly
- Upload failure → `shared.InternalError` (500), not panic
- Empty file / zero bytes
- Invalid MIME type
- File too large

### Auth/JWT (when middleware is involved)

- Missing `Authorization` header → 401
- Malformed token → 401
- Expired token → 401
- Valid token, insufficient role → 403

### Concurrent / Transaction Safety

- Only test if the module explicitly uses transactions
- Rollback on partial failure

---

## Step 5 — Test File Organization

Follow the module structure exactly:

```
internal/modules/<domain>/
├── handler.go
├── handler_test.go       ← httptest, mocks Service interface
├── service.go
├── service_test.go       ← unit tests, mocks Repository interface
├── repository.go
├── repository_test.go    ← only if repo has non-trivial logic
├── model.go
└── module.go
```

**Naming conventions:**

```
TestHandler_<MethodName>_<Scenario>
TestService_<MethodName>_<Scenario>
TestRepository_<MethodName>_<Scenario>  // only if needed
```

Examples:

```
TestHandler_GetModalities_ReturnsOK
TestHandler_GetModalities_ServiceError_Returns500
TestService_ListRentPeriods_InvalidPropertyTypeID
TestService_ListRentPeriods_EmptyResult_ReturnsEmptyData
TestService_ListModalities_RepositoryError_PropagatesError
```

---

## Step 6 — Mock Generation Rules

**Always derive mocks from the interfaces in `model.go`.**

Every method in the interface needs a corresponding `Func` field:

```go
// From model.go:
// type CatalogsRepository interface {
//     ListModalities(ctx context.Context) ([]Modality, error)
//     ListPropertyTypes(ctx context.Context) ([]PropertyType, error)
//     ListRentPeriodsByPropertyType(ctx context.Context, propertyTypeID int32) ([]RentPeriod, error)
//     ListOrientations(ctx context.Context) ([]Orientation, error)
// }

type mockCatalogsRepository struct {
    listModalitiesFunc              func(ctx context.Context) ([]Modality, error)
    listPropertyTypesFunc           func(ctx context.Context) ([]PropertyType, error)
    listRentPeriodsByPropertyTypeFunc func(ctx context.Context, id int32) ([]RentPeriod, error)
    listOrientationsFunc            func(ctx context.Context) ([]Orientation, error)
}

func (m *mockCatalogsRepository) ListModalities(ctx context.Context) ([]Modality, error) {
    if m.listModalitiesFunc != nil {
        return m.listModalitiesFunc(ctx)
    }
    return nil, nil
}
// ... rest of methods
```

Keep mocks in the `_test.go` file where they are used. Do not create a shared `mocks/` package unless three or more test files need the same mock.

---

## Step 7 — Shared Helpers Reference

`internal/shared/` contains response helpers. Your tests should verify these are triggered:

```go
// These map to HTTP status codes in tests:
shared.BadRequest(c, err)      → 400
shared.NotFound(c, "message") → 404
shared.InternalError(c, "msg") → 500
```

Response envelope to verify in handler tests:

```json
{ "data": [...], "meta": { "total": 0, "page": 1, "page_size": 20, "total_pages": 0 } }
```

For simple catalog endpoints, only `{ "data": [...] }` is expected.

---

## Step 8 — Implementation Order

When writing tests for a new module, implement in this order:

1. `service_test.go` — mock the Repository, test all business paths
2. `handler_test.go` — mock the Service, test HTTP layer
3. `repository_test.go` — only if repo has logic beyond query forwarding

For each file:

1. Write the mock struct
2. Write the happy path table-driven test
3. Add error path cases to the same table
4. Add edge cases specific to the domain

---

## Step 9 — Standards Checklist

Before finalizing any test file, verify:

- [ ] All tests are deterministic (no `time.Now()`, no random, no global state)
- [ ] Each test is independent (no shared mutable state between cases)
- [ ] Table-driven when ≥ 3 similar cases exist
- [ ] Test names follow `Test<Layer>_<Method>_<Scenario>` pattern
- [ ] Error paths verified (not just happy path)
- [ ] `wantErr bool` pattern used — avoid `err != nil` checks without a table column
- [ ] No assertions on internal implementation details (SQL queries, field assignment order)
- [ ] Mocks derive from interfaces in `model.go` only
- [ ] `t.Helper()` used in any shared assertion helper
- [ ] `t.Cleanup()` used if any resource is allocated in test setup

---

## Quick Reference — Common Gin Test Setup

```go
// Set Gin to test mode in TestMain or init
func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)
    os.Exit(m.Run())
}

// Create context with URL params
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Params = gin.Params{{Key: "uuid", Value: "some-uuid"}}

// Create context with JSON body
body := strings.NewReader(`{"name": "test"}`)
req := httptest.NewRequest(http.MethodPost, "/", body)
req.Header.Set("Content-Type", "application/json")
c.Request = req

// Check response
var result SomeStruct
json.NewDecoder(w.Body).Decode(&result)
```
