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
  - **domain/**: Entities and value objects. MUST NOT import other `internal/` packages.
  - **usecase/**: Business logic, orchestration, transaction boundaries. MAY import only `domain/` (and shared `pkg/` as needed).
  - **interface/**: Interface adapters.
    - `handler/`: HTTP request handlers, binding, error mapping.
    - `route/`: HTTP route registration.
    - `grpcserver/`, `grpcpb/`, `grpcmapper/`, `imstream/`: gRPC / realtime adapters.
  - **infra/**: Frameworks & drivers (`config/`, `repository/`, `realtime/`).
  - **app/**: Composition root — wires all layers together.

### Dependency Rule

- `domain/` MUST NOT import any other `internal/` package.
- `usecase/` MAY import `domain/` (and repository ports).
- `interface/` MAY import `usecase/` and `domain/`.
- `infra/` MAY import any package (provides concrete implementations).
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
  - Config is loaded from `.env` or environment variables via `internal/infra/config`.

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
- **Domain Models**: `internal/domain/`
