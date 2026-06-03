# Spazio Backend

> A modern, scalable real estate management platform built with Go, PostgreSQL, and a vertical slice architecture.

## 📋 Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Database Setup](#database-setup)
- [Working with SQLC](#working-with-sqlc)
- [API Documentation](#api-documentation)
- [Development Workflow](#development-workflow)

---

## 🎯 Overview

**Spazio** is a comprehensive backend API for real estate management covering:

- **Property Management** — Publication and administration of residential and commercial properties
- **User Management** — Role-based access control (RBAC) and user authentication with JWT
- **CRM & Logistics** — Inquiry tracking, visit scheduling, and agent assignment
- **Financial Operations** — Multi-currency pricing, rent periods, real payment gateway integration (MercadoPago), and transaction management
- **Multimedia** — Property photos stored on Cloudflare R2 with optimized URLs
- **Security** — Soft deletes, audit trails, and audit events for compliance

### Key Features

| Feature               | Implementation                |
| --------------------- | ----------------------------- |
| **Authentication**    | Local JWT access tokens and refresh tokens |
| **Authorization**     | Role-based permissions (RBAC) |
| **Database**          | PostgreSQL 15+ with pgx v5    |
| **Data Access**       | sqlc for type-safe SQL        |
| **Migrations**        | golang-migrate with SQL files |
| **Object Storage**    | Cloudflare R2 (S3-compatible) |
| **API Documentation** | Swagger/OpenAPI with swaggo   |

---

## 🛠️ Tech Stack

### Core

```
Go              1.26.1    Language
Gin             1.12      HTTP Framework
PostgreSQL      15+       Database
pgx/v5          5.9.2     Database Driver
```

### Data & ORM

```
sqlc            1.13      Type-safe SQL compiler
golang-migrate  4.19.1    Database migrations
```

### Authentication & Storage

```
JWT v5          5.3.1     Token handling
AWS SDK v2      1.41.7    S3/R2 client
UUID            1.6.0     ID generation
```

### API & Documentation

```
swaggo/swag     1.16.6    Swagger code generator
gin-swagger     1.6.1     Swagger UI
```

### Development

```
godotenv        1.5.1     Environment variables
go-webpbin      0.0.0     WebP image encoding
```

---

## 🏗️ Architecture

### Vertical Slice Pattern

Spazio uses **vertical slicing** to organize code by feature domain rather than technical layers. Each module is self-contained with its own request handler → business logic → data access pipeline.

```
Module (Domain)
├── handler.go       ← HTTP layer (routes, validation, response)
├── service.go       ← Business logic (rules, orchestration)
├── repository.go    ← Data access (queries, persistence)
├── model.go         ← Interfaces, DTOs, domain types
└── module.go        ← Composition root (dependency injection)
```

### Request Flow

```
1. HTTP Request
    ↓
2. Handler (validation, parameter extraction)
    ↓
3. Service (business rules, orchestration)
    ↓
4. Repository (database queries)
    ↓
5. JSON Response
```

### Dependency Injection Pattern

Modules use **manual wiring** (no framework). Each module constructs its own dependencies:

```go
func NewModule(db *pgxpool.Pool, r2Client *storage.R2Client) *Module {
	repository := NewRepository(db)        // Layer 1: Data access
	service := NewService(repository, r2Client) // Layer 2: Logic
	handler := NewHandler(service)         // Layer 3: HTTP
	return &Module{handler: handler}
}
```

---

## 🚀 Getting Started

### Prerequisites

- **Go** 1.26.1+
- **PostgreSQL** 15+
- **Git**
- **SQLC** v1.13+ (for code generation)
- **golang-migrate** (for database migrations)

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/jaftdelgado/spazio-backend.git
   cd spazio-backend
   ```

2. **Install Go dependencies**

   ```bash
   go mod download
   go mod tidy
   ```

3. **Set up environment variables**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your configuration:

   ```env
   APP_PORT=8080
   DATABASE_URL=postgres://user:password@localhost:5432/spazio
   ```

---

## Database Setup

### Run Migrations

The project uses **golang-migrate** for versioned schema changes.

```bash
# Run all pending migrations
go run ./cmd/migrate

# Migrations are applied in order from migration/sql/
# Each migration has up (.up.sql) and down (.down.sql) files
```

### Migration Files

| File                             | Purpose                           |
| -------------------------------- | --------------------------------- |
| `000001_init.up.sql`             | Initial schema setup              |
| `000002_schema.up.sql`           | Additional tables and constraints |
| `000003_clauses.up.sql`          | Property clauses entities         |
| `000004_update.up.sql`           | Schema refinements                |
| `000005_prices.up.sql`           | Pricing entities                  |
| `000006_typeperiods.up.sql`      | Rent period types                 |
| `000007_property_subtype.up.sql` | Property subtypes                 |

### Connection Pool

The application uses **pgx v5** with connection pooling for performance:

```go
database, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
if err != nil {
    log.Fatal(err)
}
defer database.Close()
```

---

## 📝 Working with SQLC

SQLC generates type-safe Go code from the SQL files under `sqlc/`. The setup lives in `sqlc/sqlc.yaml`, and generated code is written to `internal/sqlcgen/`.

### Regenerate

```bash
cd sqlc
sqlc generate
```

### Configuration

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema.sql"
    gen:
      go:
        package: "sqlcgen"
        out: "../internal/sqlcgen"
        sql_package: "pgx/v5"
```

---

## 📚 API Documentation

Swagger is generated from handler comments in `internal/modules/*/handler.go`.

```go
// getProperty godoc
// @Summary      Get property by UUID
// @Description  Returns property data including prices and photos
// @Tags         Properties
// @Produce      json
// @Param        uuid  path     string  true  "Property UUID"
// @Success      200   {object} PropertyResponse
// @Failure      404   {object} shared.ErrorResponse
// @Router       /api/v1/properties/{uuid} [get]
func (h *Handler) getProperty(c *gin.Context) { ... }
```

### Regenerate docs

```bash
swag init -g cmd/api/main.go --output docs
```

### View docs

```
http://localhost:8080/swagger/index.html
```

The documented modules are properties, services, catalogs, clauses, locations, users, uploads, and visits.

---

## 💻 Development Workflow

### 1. Making Database Changes

```bash
# Step 1: Create a new migration
# Create migration/sql/000008_your_feature.up.sql
# Create migration/sql/000008_your_feature.down.sql

# Step 2: Update the database schema
sqlc/schema.sql

# Step 3: Update your SQL queries
# Edit sqlc/queries/*.sql

# Step 4: Regenerate sqlc code
cd sqlc && sqlc generate

# Step 5: Run the migration
go run ./cmd/migrate

# Step 6: Regenerate Swagger docs
swag init -g cmd/api/main.go --output docs
```

### 2. Creating a New Module

```bash
# Create module directory
mkdir -p internal/modules/your_feature

# Create files
touch internal/modules/your_feature/{handler,service,repository,model,module}.go
```

**Module Template:**

```go
// model.go - Define interfaces and DTOs
type YourFeatureService interface {
    GetByID(ctx context.Context, id int32) (*YourFeature, error)
}

// repository.go - Implement data access
type Repository struct { db *pgxpool.Pool }
func (r *Repository) GetByID(ctx context.Context, id int32) (*YourFeature, error) { ... }

// service.go - Implement business logic
type Service struct { repo YourFeatureRepository }
func (s *Service) GetByID(ctx context.Context, id int32) (*YourFeature, error) { ... }

// handler.go - HTTP handlers with Swagger annotations
func (h *Handler) getYourFeature(c *gin.Context) { ... }

// module.go - Compose dependencies
func NewModule(db *pgxpool.Pool) *Module {
    repo := NewRepository(db)
    svc := NewService(repo)
    handler := NewHandler(svc)
    return &Module{handler: handler}
}
```

### 3. Running the Application

```bash
# Development mode
go run cmd/api/main.go

# With custom environment
APP_PORT=3000 DATABASE_URL=... go run ./cmd/api

# Production build
go build -o spazio-api ./cmd/api
./spazio-api
```

### 4. Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific module
go test ./internal/modules/properties/...
```

---

## 🔑 Key Conventions

### Error Handling

All endpoints return JSON error responses using `shared.ErrorResponse`:

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

Helper functions in `internal/shared/`:

```go
shared.BadRequest(c, err)        // 400
shared.NotFound(c, "message")    // 404
shared.InternalError(c, "msg")   // 500
```

### Soft Deletes

Most entities use soft deletes (set `deleted_at` timestamp):

```sql
SELECT * FROM properties WHERE deleted_at IS NULL
```

### Response Format

All responses are JSON. Standard envelope:

```json
{
  "data": [...],
  "meta": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### Authentication

JWT tokens passed in `Authorization` header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

Signed locally with `JWT_SECRET` and validated in `middleware/auth.go`.

---

## 📖 Additional Resources

- [Database Schema](./database.md)
- [SQL Queries](./sqlc/queries/)
- [Migration Files](./migration/sql/)
- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/)
- [SQLC Documentation](https://sqlc.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

---

## 📝 License

Proprietary. All rights reserved.
resql.org/docs/)

---

## 📝 License

Proprietary. All rights reserved.
