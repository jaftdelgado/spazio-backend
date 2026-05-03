# Contributing to spazio-backend

## Table of contents

- [Branch strategy](#branch-strategy)
- [Commit conventions](#commit-conventions)
- [Pull requests](#pull-requests)
- [Tests](#tests)
- [Code style](#code-style)

---

## Branch strategy

This project follows **Gitflow**. The long-lived branches are:

| Branch | Purpose |
|---|---|
| `main` | Production-ready code. Only receives merges from `hotfix/*` and `develop` via release. |
| `develop` | Integration branch. All feature work targets this branch. |

Short-lived branches must be created from and merged back into the correct base:

| Prefix | Base | Merges into | When to use |
|---|---|---|---|
| `feature/*` | `develop` | `develop` | New functionality or use cases |
| `hotfix/*` | `main` | `main` + `develop` | Critical fixes on production |
| `docs/*` | `develop` | `develop` | Documentation only — no logic changes |

### Naming convention

Branch names must be lowercase, hyphen-separated and scoped to the module they affect:

```
feature/properties-delete-property
feature/uploads-bulk-photo-upload
hotfix/properties-fix-price-hierarchy
docs/properties-api-reference
```

Never commit directly to `main` or `develop`.

---

## Commit conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org). Every commit message must follow this format:

```
<type>(<scope>): <short description>
```

The short description must be in **English**, imperative mood, lowercase, no trailing period.

### Allowed types

| Type | When to use |
|---|---|
| `feat` | New feature or endpoint |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `chore` | Tooling, dependencies, config — no production code |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `perf` | Performance improvement |

### Scope

Use the module name as scope: `properties`, `uploads`, `auth`, `shared`, `sqlc`, `migrations`, etc.

### Examples

```
feat(properties): add paginated property listing endpoint
fix(properties): correct rent price hierarchy in card display
refactor(properties): remove redundant db roundtrip in delete flow
chore(deps): upgrade pgx to v5.6.0
docs(properties): add swagger annotations to delete endpoint
test(properties): add handler tests for soft delete use case
```

---

## Pull requests

### Rules

- Every PR must target `develop` (or `main` for hotfixes).
- **At least one approval is required** before merging. The author cannot approve their own PR.
- PRs must pass all CI checks before they can be merged.
- Squash merge is preferred to keep `develop` history clean.

### PR title

Follow the same Conventional Commits format used for commits:

```
feat(properties): complete delete property use case
```

### PR description

Include the following sections:

```
## What this PR does
Brief description of the change and why it was made.

## Endpoints affected (if any)
List new or modified endpoints with method and path.

## How to test
Steps to manually verify the change if applicable.

## Checklist
- [ ] All existing tests pass (`go test ./...`)
- [ ] New tests added for new behavior
- [ ] Swagger annotations updated
- [ ] No direct writes in read-only endpoints
- [ ] `sqlc generate` was run if SQL queries were modified
```

---

## Tests

All commits and PRs are validated against the test suite. **A PR will not be merged if any test fails.**

### Running tests locally

```bash
go test ./...
```

### Requirements

- Every new handler must have tests in `handler_test.go` covering at minimum: happy path, not found, and validation errors.
- Read-only endpoints must not produce any database writes — this is enforced by design and must be verified in tests.
- If you modify or add sqlc queries, run `sqlc generate` before committing and include the regenerated file in your PR.

---

## Code style

- Follow the patterns established in each module. Read existing files before writing new ones.
- Use the vertical slice structure: `handler_*.go`, `service_*.go`, `repository_*.go`, `model.go`.
- All write operations that touch multiple tables must run inside a single transaction with full rollback on error.
- Error messages must be lowercase and in English.
- Do not use string concatenation to build SQL queries — use `sqlc` with typed parameters.
- Swagger/godoc annotations are required for every public endpoint.
