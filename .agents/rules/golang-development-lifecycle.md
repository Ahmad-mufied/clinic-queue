# Rule: Golang Development Lifecycle & Hexagonal Code Patterns
**Scope:** Backend Go Services  
**Target Environment:** Go 1.27+ | Echo v4 | PostgreSQL 18 | NATS JetStream | Casbin v2  
**Status:** Active Rule

---

## 1. Development Lifecycle Workflow (TDD & Port-First)

Every backend feature must progress through the following 5-stage lifecycle:

```
[1. Port Definition]
   └── Define Inbound Port (UseCase interface) & Outbound Port (SPI interface)
   ▼
[2. Core Domain & UseCase]
   └── Implement pure business logic in `internal/core/usecase` (Zero external framework imports)
   ▼
[3. Table-Driven Core Unit Tests]
   └── Implement 100% table-driven unit tests with mock ports in `internal/core/usecase/*_test.go`
   ▼
[4. Adapters Implementation]
   ├── Inbound Adapter: Echo HTTP Handlers & Middlewares (`internal/adapters/inbound/`)
   └── Outbound Adapter: PostgreSQL (pgx) & NATS JetStream (`internal/adapters/outbound/`)
   ▼
[5. Table-Driven Adapter Tests & Quality Verification]
   └── Handlers unit tests (100% coverage), `go test -v -race -coverprofile=...`, `go vet ./...`
```

---

## 2. Hexagonal Architecture Code Patterns

### 2.1 Boundary & Dependency Rules
- **Core Domain & UseCases (`internal/core/`):** Must NEVER import external HTTP frameworks (Echo), database drivers (pgx), or messaging drivers (NATS).
- **Inbound Ports (`internal/core/ports/inbound/`):** Define what external actors can request from the core application.
- **Outbound Ports (`internal/core/ports/outbound/`):** Define what the core application requires from secondary infrastructure (Repositories, Publishers, Loggers).

### 2.2 Standard Ports & Mock Patterns
Use closure-based mock structs for testing ports without external mock generator binaries:

```go
// Outbound Port Interface
type UserRepositoryPort interface {
    FindByUsername(ctx context.Context, username string) (*domain.User, error)
    CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
}

// Testing Mock Struct (Located in *_test.go)
type mockUserRepositoryPort struct {
    findByUsernameFunc func(ctx context.Context, username string) (*domain.User, error)
    createUserFunc     func(ctx context.Context, user *domain.User) (*domain.User, error)
}

func (m *mockUserRepositoryPort) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
    if m.findByUsernameFunc != nil {
        return m.findByUsernameFunc(ctx, username)
    }
    return nil, nil
}
```

---

## 3. Go 1.27 Idiomatic Code Conventions

### 3.1 Modern Control Flow & Slices
- **Count Iterations:** Use `for range count` when index variable is unused.
- **Sorting & Comparisons:** Use standard `slices.SortFunc` and `cmp.Compare`:
  ```go
  slices.SortFunc(items, func(a, b *Item) int {
      return cmp.Compare(a.Priority, b.Priority)
  })
  ```

### 3.2 Error Handling Standard
- Define package sentinel errors with `var ErrX = errors.New(...)`.
- Wrap errors with descriptive context: `fmt.Errorf("execute query for user %s: %w", username, err)`.
- Use `errors.Is(err, TargetErr)` and `errors.As(err, &customErr)` for checking.

### 3.3 Concurrency & Database Atomicity
- Pass `ctx context.Context` as the first argument in all UseCase and Repository methods.
- For concurrent queue operations, use atomic PostgreSQL row locking: `SELECT ... FOR UPDATE SKIP LOCKED`.

---

## 4. Verification & Quality Commands

```bash
# Run unit tests with race detection and statement coverage
go test -v -race -coverprofile=coverage.out ./...

# View function-level coverage breakdown
go tool cover -func=coverage.out

# Static analysis & vetting
go vet ./...
```
