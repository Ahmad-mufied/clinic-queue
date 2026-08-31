# Clinic Queue

A real-time and transparent queue management platform for clinics. It provides dynamic wait-time estimation for walk-in patients, operational control for doctors, automated audit trails, and executive performance analytics.

## Features

- **Authentication & RBAC**: Secure authentication and role-based access control (Casbin) for Patients, Doctors, and Admins.
- **Patient Queue Engine**: Online walk-in registration with real-time SSE queue monitoring and dynamic wait estimation.
- **Doctor Workspace**: Shift management (Online/Offline) and consultation workflow (call next patient, complete session).
- **Admin Analytics**: Executive dashboard for tracking clinic throughput, doctor utilization rates, and system metrics.
- **Audit Trail**: Comprehensive, immutable activity logging for compliance and transparency.

## Technology Stack

**Backend**
- Go 1.27 (Hexagonal Architecture)
- PostgreSQL 18 (UUIDv7 identifiers)
- NATS JetStream (Event Streaming)
- Echo v4 (Web Framework)
- Casbin v2 (RBAC Authorization)

**Frontend**
- Next.js (App Router)
- React
- Tailwind CSS
- Server-Sent Events (SSE) for Real-Time Updates

## Project Structure

- `cmd/`: Application entrypoints.
- `internal/`: Core business logic, domain models, and adapters (Hexagonal Architecture).
- `web/`: Next.js frontend application.
- `docs/`: Comprehensive Product Requirements Documents (PRD) and Technical Specifications.
- `migrations/`: Database migrations and schema definitions.

## Getting Started

### Prerequisites
- Go 1.27+
- Node.js 20+
- Docker & Docker Compose
- GNU Make

### Quick Start with Makefile (Recommended)

1. **Initial Setup (Environment & Dependencies)**
   ```bash
   make setup
   ```
   *Creates `.env` from `.env.example` and installs frontend dependencies.*

2. **Start Infrastructure (PostgreSQL 18 & NATS JetStream)**
   ```bash
   make infra-up
   ```
   *Spins up PostgreSQL on port `5433` and NATS on ports `4222`/`8222` in the background.*

3. **Run Development Servers**
   - **Terminal 1 (Backend Go with Air hot-reload on `:8080`):**
     ```bash
     make dev-api
     ```
   - **Terminal 2 (Frontend Next.js on `:3000`):**
     ```bash
     make dev-web
     ```

*(Run `make dev` anytime for quick multi-terminal startup instructions)*

---

### Makefile Commands Reference

| Target | Description |
| :--- | :--- |
| **`make help`** | Display all available make targets and descriptions |
| **`make setup`** | Prepare `.env` file and install frontend npm dependencies |
| **`make infra-up`** | Start PostgreSQL 18 (`:5433`) & NATS (`:4222`) containers |
| **`make infra-down`** | Stop infrastructure containers |
| **`make infra-logs`** | Follow real-time logs from DB and message broker |
| **`make dev-api`** | Run Go API server with Air hot-reload (`:8080`) |
| **`make dev-web`** | Run Next.js frontend dev server (`:3000`) |
| **`make build`** | Build both backend binary (`bin/api`) and frontend production bundle |
| **`make test`** | Run all unit tests with race detection & coverage profile |
| **`make test-cover`** | View code coverage percentage per function |
| **`make test-html`** | Generate interactive HTML code coverage visualization |
| **`make test-e2e`** | Execute automated E2E regression suite (70 live scenarios) |
| **`make test-stress`** | Run high-concurrency stress benchmark (500 joins, 50 calls) |
| **`make vet`** | Run Go static analysis (`go vet`) |
| **`make clean`** | Remove build binaries, logs, and coverage reports |

---

### Manual Setup (Without Make)

1. **Start Infrastructure**
   ```bash
   docker compose up -d
   ```

2. **Run Backend**
   ```bash
   go run cmd/api/main.go
   ```

3. **Run Frontend**
   ```bash
   cd web
   npm install
   npm run dev
   ```

## Documentation
For deeper technical insights and feature specifications, refer to the `docs/` folder:
- [Architecture Overview](docs/tech/ARCH.md)
- [Product Requirements Master](docs/prd/PRD.md)
- [NATS JetStream Setup Guide](docs/deployment/NATS-SETUP.md)
- [Identity & UUIDv7 Design](docs/tech/IDENTITY-DESIGN.md)

