# GitHub Copilot Instructions for Go Backend

You are an expert Go developer assisting with the `learn-go` backend project. This is a RESTful API and WebSocket server built with Go, Gin, and GORM.

## 🏗 Architecture & Project Structure

- **Standard Go Layout**:
  - `cmd/server/`: Application entry point (`main.go`).
  - `internal/`: Private application code.
  - `pkg/`: Public library code (e.g., `logger`, `middleware`).
- **Layered Architecture** (inside `internal/`):
  - **api/http**: Gin handlers, routing, request binding.
  - **service**: Business logic, orchestration, transaction boundaries.
  - **repository**: Data access layer using GORM.
  - **domain**: Core domain models and interfaces.

## 🛠 Tech Stack & Libraries

- **Language**: Go 1.22+
- **Web Framework**: `gin-gonic/gin`.
- **Database**: `gorm` (SQLite for dev, PostgreSQL for prod).
- **Realtime**: `gorilla/websocket`.
- **Auth**: JWT (JSON Web Tokens).

## 📝 Coding Conventions

- **Dependency Injection**:
  - Services and Repositories are explicitly injected.
  - See `internal/api/http/handlers.go` for how handlers aggregate services.
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
- **Router/Handlers**: `internal/api/http/handlers.go`
- **Domain Models**: `internal/domain/`
