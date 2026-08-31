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

### Running Locally

1. **Start Infrastructure (PostgreSQL & NATS)**
   ```bash
   docker-compose up -d
   ```

2. **Run Backend (Go)**
   ```bash
   go mod download
   go run cmd/api/main.go
   ```

3. **Run Frontend (Next.js)**
   ```bash
   cd web
   npm install
   npm run dev
   ```

## Documentation
For deeper technical insights and feature specifications, refer to the `docs/` folder:
- [Architecture Overview](docs/tech/ARCH.md)
- [Product Requirements Master](docs/prd/PRD.md)
