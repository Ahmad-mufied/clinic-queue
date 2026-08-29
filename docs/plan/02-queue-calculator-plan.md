# Implementation Plan: 02 - Patient Queue & Live Wait Countdown (Queue Engine & NATS JetStream SSE)
**File:** `docs/plan/02-queue-calculator-plan.md`  
**Status:** ✅ **100% COMPLETED & OFFICIALLY SIGNED OFF**  
**Target Feature:** [`docs/prd/02-patient-queue.md`](../prd/02-patient-queue.md) & [`docs/tech/02-queue-calculator-tech.md`](../tech/02-queue-calculator-tech.md)  
**Architecture:** Hexagonal Architecture (Ports & Adapters) in Go 1.27 + NATS JetStream + PostgreSQL 18  
**Completion Date:** 2026-08-29  

---

## 1. Objective & Scope

Implement **Feature 02: Patient Queue & Live Wait Countdown** using:
1. **Deterministic Greedy Multi-Doctor Simulation Engine:** Computes exact minute-accurate estimated waiting times for any arbitrary queue position under varying doctor states (online/offline, in-consultation with elapsed time, idle).
2. **Hexagonal Core & Ports:** Pure business logic in `internal/core/usecase/queue_usecase.go` and `internal/core/domain/calculator.go` decoupled via `ports/inbound` and `ports/outbound`.
3. **NATS JetStream & SSE Hub:** Publish queue events (`QUEUE_JOINED`, `QUEUE_UPDATED`, `TICKET_CALLED`) to NATS JetStream and broadcast them real-time to connected web clients via Server-Sent Events (`GET /api/events`).
4. **100% Table-Driven Test Suite:** Complete unit tests for Greedy Calculator, Queue UseCase, and Queue HTTP Handlers.

---

## 2. Technical Deliverables & Completion Checklist

- [x] **Phase 2.1: Domain Entities & Pure Greedy Calculator Engine**
  - [x] `internal/core/domain/queue.go` (Entities `QueueTicket`, `TicketStatus`, `Doctor`, `DoctorAvailability`, `QueueStatus`, `QueueTicketSummary`).
  - [x] `internal/core/domain/calculator.go` (Pure Go 1.27 Greedy Earliest-Available-First multi-doctor simulation algorithm).
  - [x] `internal/core/domain/calculator_test.go` & `queue_test.go` (100% statement coverage table-driven unit tests).

- [x] **Phase 2.2: Inbound & Outbound Ports Definition**
  - [x] `internal/core/ports/inbound/queue_port.go` (`QueueUseCase` Inbound Port).
  - [x] `internal/core/ports/outbound/queue_repo_port.go` (`QueueRepositoryPort` Outbound SPI Port).
  - [x] `internal/core/ports/outbound/doctor_repo_port.go` (`DoctorRepositoryPort` Outbound SPI Port).
  - [x] `internal/core/ports/outbound/event_pub_port.go` (`EventPublisherPort` Outbound SPI Port).

- [x] **Phase 2.3: Core UseCase Implementation**
  - [x] `internal/core/usecase/queue_usecase.go` (Queue join validation, double check-in prevention, dynamic wait time recalculation, sequential ticket generation `A-01`, `A-02`).
  - [x] `internal/core/usecase/queue_usecase_test.go` (100% statement coverage table-driven tests with closure-based mock ports).

- [x] **Phase 2.4: Outbound Adapters (PostgreSQL 18 & NATS JetStream)**
  - [x] `internal/adapters/outbound/postgres/queue_repo.go` (PostgreSQL 18 implementation via `pgxpool.Pool`).
  - [x] `internal/adapters/outbound/postgres/doctor_repo.go` (PostgreSQL 18 implementation with active session left joins).
  - [x] `internal/adapters/outbound/nats/nats_client.go` (NATS JetStream connection & stream `CLINIC_EVENTS` provisioning).
  - [x] `internal/adapters/outbound/nats/nats_publisher.go` (Event publisher emitting to `clinic.events.queue`).

- [x] **Phase 2.5: Inbound Adapters (Echo Handlers & Real-Time SSE Hub)**
  - [x] `internal/adapters/inbound/http/queue_handler.go` (`POST /api/queue/join`, `GET /api/queue/my-ticket`, `GET /api/queue/status`).
  - [x] `internal/adapters/inbound/http/queue_handler_test.go` (100% statement coverage table-driven HTTP tests).
  - [x] `internal/adapters/inbound/http/sse_handler.go` (`GET /api/events` Server-Sent Events broadcaster with keep-alive).
  - [x] `internal/adapters/inbound/http/sse_handler_test.go` (100% statement coverage unit tests).

- [x] **Phase 2.6: Server Bootstrap / Composition Root**
  - [x] `cmd/api/main.go` (Wires NATS JetStream, PostgreSQL repos, QueueUseCase, SSE Broadcaster, and Echo route handlers).

---

## 3. Comprehensive Quality & Testing Report

### 3.1 Unit Test Statement Coverage Report (`go test -v -race`)

```text
ok      clinic-queue/config                                 coverage: 100.0% of statements
ok      clinic-queue/internal/core/domain                   coverage: 100.0% of statements
ok      clinic-queue/internal/core/usecase                  coverage: 100.0% of statements
ok      clinic-queue/internal/adapters/inbound/http         coverage: 100.0% of statements
ok      clinic-queue/internal/adapters/inbound/middleware   coverage: 100.0% of statements
```

- **Deterministic Greedy Calculator:** **100.0%** coverage.
- **Queue UseCase:** **100.0%** coverage across all positive, validation, and error branches.
- **HTTP Handlers:** **100.0%** coverage (200, 201, 400, 401, 404, 409, 500 status codes).
- **SSE Stream Broadcaster:** **100.0%** coverage.
- **Race Condition Detection:** **0 race conditions** with `-race` flag enabled.

---

### 3.2 Live Manual Integration & Real-Time SSE Test Report

The live server was booted alongside active PostgreSQL 18 and NATS JetStream containers. Full end-to-end integration scenarios were executed and verified:

| Test ID | Scenario | HTTP Method & Path | Payload / Conditions | Result | Verification Status |
| :---: | :--- | :--- | :--- | :---: | :---: |
| **E2E-QUE-01** | Public Queue Status | `GET /api/queue/status` | *None* | `200 OK` | ✅ **PASS** (Returned Doctor A [3m] & Doctor B [4m]) |
| **E2E-QUE-02** | Patient 1 Join (John) | `POST /api/queue/join` | `{"patient_name": "John Doe"}` | `201 Created` | ✅ **PASS** (Ticket `A-01`, Pos: 1, Wait: 0m) |
| **E2E-QUE-03** | Patient 2 Join (Lucas) | `POST /api/queue/join` | `{"patient_name": "Lucas Smith"}` | `201 Created` | ✅ **PASS** (Ticket `A-02`, Pos: 2, Wait: 0m) |
| **E2E-QUE-04** | My Ticket Status | `GET /api/queue/my-ticket` | `Bearer <John_Token>` | `200 OK` | ✅ **PASS** (Active Ticket `A-01` returned) |
| **E2E-QUE-05** | Duplicate Join Prevention | `POST /api/queue/join` | `{"patient_name": "John Doe"}` | `409 Conflict` | ✅ **PASS** (`Active queue ticket already exists`) |
| **E2E-QUE-06** | Real-Time SSE Stream | `GET /api/events` | *EventSource Connection* | `200 OK` (stream) | ✅ **PASS** (Instant `QUEUE_JOINED` events received) |
| **E2E-QUE-07** | 11th Patient Wait Math | `POST /api/queue/join` (11th) | 10 patients in queue ahead | `201 Created` | ✅ **PASS** (Ticket `A-11`, Pos: 11, Wait: **16m**) |

---

## 4. Sign-Off & Readiness for Feature 03

| Milestone | Target | Actual Result | Sign-Off Status |
| :--- | :---: | :---: | :---: |
| **Greedy Queue Algorithm Accuracy** | 100% Case Study Match | Verified Pos 1-11 = 0, 0, 3, 4, 6, 8, 9, 12, 12, 15, 16 mins | ✅ **APPROVED** |
| **Code Statement Coverage** | 100% | **100.0%** across core usecases and handlers | ✅ **APPROVED** |
| **Real-Time NATS/SSE Streaming** | Live Stream | Real-time SSE delivery $< 100\text{ms}$ | ✅ **APPROVED** |
| **Database Persistence & Locks** | PostgreSQL 18 | Sequential ticket numbering & active tracking | ✅ **APPROVED** |

**Conclusion:** Feature 02 (Patient Queue & Live Wait Countdown) is **officially completed, tested, and locked**. Ready to proceed to **Feature 03: Doctor Workspace & Shift Management**.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Lead Architect | **Initial Baseline** | Master implementation plan for Feature 02 (Queue Engine & NATS SSE). |
| **v1.1.0** | 2026-08-29 | Backend Lead | **Final Sign-Off Report** | Full execution report, 100% table-driven unit tests, real-time SSE stream validation, and mathematical wait time verification. |
