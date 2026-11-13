# GitHub Copilot Instructions for Go Backend

You are an expert Go developer assisting with the `learn-go` backend project. This is a RESTful API and WebSocket server built with Go, Gin, and GORM.

## 🏗 Architecture & Project Structure

This project follows **Clean Architecture** (aligned with `cloudpvp-matcher`):

```
infrastructure → interface → usecase → domain
  (外部框架)       (适配层)      (用例层)      (实体层)
```

- **Standard Go Layout**:
  - `cmd/server/`: Application entry point (`main.go`).
  - `internal/`: Private application code.
  - `pkg/`: Public library code (e.g., `logger`, `middleware`).
- **Directory-to-Layer Mapping** (inside `internal/`):
  - **domain/**: Split by bounded context. Each subpackage owns its entities, value objects and repository interfaces (`repository.go`, the ports implemented by `infra/repository/gorm`). Shared errors (e.g. `domain.ErrNotFound`) live in the root `domain` package. Domain packages MUST NOT import other `internal/` packages except other `domain/` packages (no cycles).
    - `identity/`: accounts, roles, password reset tokens.
    - `organization/`: schools, departments, classes, classrooms.
    - `roster/`: teacher / student profiles, teacher-student links, student reminders.
    - `curriculum/`: courses, enrollments, course teachers, chapters.
    - `scheduling/`: schedule rules, sessions, time slots.
    - `coursework/`: assignments, questions, submissions, comments.
    - `messaging/`: conversations, messages, read receipts.
    - `notebook/`: notes and note comments.
    - `notification/`: user notifications.
    - `aiagent/`: AI agent settings, audits, usage logs.
    - `system/`: switches, parameters, broadcasts, audit logs.
    - `library/`: teaching resources, resource files, favorites.
    - `storage/`: uploaded files and OSS credentials / policies / audits.
  - **usecase/**: Business logic, orchestration, transaction boundaries. MAY import only `domain/` (plus the standard library and `github.com/google/uuid`).
  - **interface/**: Interface adapters.
    - `handler/`: HTTP request handlers, binding, error mapping.
    - `route/`: HTTP route registration.
    - `grpcserver/`, `grpcpb/`, `grpcmapper/`, `imstream/`: gRPC / realtime adapters.
  - **infra/**: Implementations of domain ports only (`repository/gorm/`, `security/`, `objectstorage/`, `realtime/`). Every exported type MUST carry a `var _ <domain port> = ...` assertion and expose no exported methods beyond that port.
  - **config/**: Application configuration loading (not part of infra).
  - **app/**: Composition root — wires all layers together.

### Dependency Rule

- `domain/` MUST NOT import any other `internal/` package outside `domain/`.
- `usecase/` MAY import only `domain/` (repository, security, object storage and publisher ports are defined there); it MUST NOT import `infra/`, `interface/`, `pkg/`, gorm or jwt. Use `domain.ErrNotFound` for missing records, never `gorm.ErrRecordNotFound`.
- `interface/` MAY import `usecase/` and `domain/`.
- `infra/` MAY import `domain/`, `pkg/` and third-party drivers; it MUST NOT import `usecase/`, `interface/`, `handler/` or `config/`. Transport-specific code (e.g. protobuf conversion, gRPC stream fan-out) belongs in `interface/`.
- `cmd/` / `app/` is the composition root — the place where all layers are wired together.

## 🛠 Tech Stack & Libraries

- **Language**: Go 1.22+
- **Web Framework**: `gin-gonic/gin`.
- **Database**: `gorm` (SQLite for dev, PostgreSQL for prod).
- **Realtime**: `gorilla/websocket`.
- **Auth**: JWT (JSON Web Tokens).

## 📝 Coding Conventions

- **Dependency Injection**:
  - Usecases and Repositories are explicitly injected.
  - See `internal/handler/handlers.go` for how handlers aggregate usecases.
  - See `internal/route/` for route registration.
  - See `internal/app/app.go` for the wiring of the application.
- **Database**:
  - Use `gorm.Model` or custom structs with GORM tags.
  - Repositories should accept `*gorm.DB` or interfaces.
- **Error Handling**:
  - Return errors up the stack.
  - Handlers are responsible for mapping errors to HTTP status codes and JSON responses.
- **Configuration**:
  - Config is loaded from `.env` or environment variables via `internal/config`.

## 🚀 Development Workflow

- **Run Server**:
  ```bash
  go run ./cmd/server
  ```
- **Run Tests**:
  ```bash
  go test ./...
  ```
- **Format Code**:
  ```bash
  gofmt -w .
  ```

## 🔍 Key Files

- **Entry Point**: `cmd/server/main.go`
- **App Wiring**: `internal/app/app.go`
- **Handlers**: `internal/handler/`
- **Routes**: `internal/route/`
- **Domain Models & Repository Ports**: `internal/domain/<context>/`
- **Repository Implementations**: `internal/infra/repository/gorm/`
