# Feature Specification: Comprehensive Activity Logging & Audit Trail
**File:** `docs/prd/05-audit-trail.md`  
**Status:** Approved  
**Target Roles:** `admin`

---

## 1. Feature Definition & Scope

The Audit Trail subsystem captures an immutable, chronological stream of all security, operational, and lifecycle events across the clinic platform. 

It satisfies healthcare compliance standards, enables forensic accountability, and serves as the data foundation for business analytics.

---

## 2. User Stories

1. **As a Security & Compliance Officer**, I want every login, logout, and access attempt recorded with timestamps, user ID, and IP address so that unauthorized access can be investigated.
2. **As an Operations Auditor**, I want to see the full historical lifecycle of any queue ticket (when joined, who called it, when completed) to resolve patient complaints or disputes.
3. **As a Clinic Administrator**, I want a paginated and filterable Audit Log Feed in the UI so that I can inspect recent clinic activities in real time.

---

## 3. Audit Event Taxonomy

| Event Category | Action Identifier | Actor | Details Payload Schema (JSONB) |
| :--- | :--- | :--- | :--- |
| **Authentication** | `AUTH_LOGIN` | Any | `{ "username": "dr_smith", "role": "doctor", "success": true }` |
| | `AUTH_LOGOUT` | Any | `{ "session_duration_min": 120 }` |
| | `AUTH_REGISTER` | Patient | `{ "username": "john_doe", "name": "John Doe" }` |
| **Doctor Shift** | `DOCTOR_SHIFT_STARTED` | Doctor | `{ "doctor_id": 1, "doctor_name": "Doctor A" }` |
| | `DOCTOR_SHIFT_ENDED` | Doctor | `{ "doctor_id": 1, "shift_duration_min": 240 }` |
| **Queue Lifecycle** | `QUEUE_JOINED` | Patient | `{ "ticket_id": 101, "queue_number": "A-06", "ahead_count": 5 }` |
| | `CONSULTATION_STARTED` | Doctor | `{ "ticket_id": 101, "doctor_id": 1, "patient": "John" }` |
| | `CONSULTATION_FINISHED`| Doctor | `{ "ticket_id": 101, "actual_duration_min": 3.2, "target_min": 3 }` |
| | `QUEUE_CANCELLED` | Patient / Admin| `{ "ticket_id": 101, "reason": "patient_left" }` |
| **Configuration** | `DOCTOR_CONFIG_UPDATED`| Admin | `{ "doctor_id": 1, "old_avg_min": 3, "new_avg_min": 4 }` |

---

## 4. Case Scenarios

### 4.1 Positive Scenarios
- **[POS-AUDIT-01] Automated Queue Event Logging:**  
  When John clicks "Join Queue", system creates ticket in DB and atomically inserts an `audit_logs` record (`action = 'QUEUE_JOINED'`, actor John, IP `127.0.0.1`).
- **[POS-AUDIT-02] Doctor Consultation Lifecycle Logging:**  
  When Doctor A calls John $\rightarrow$ `CONSULTATION_STARTED` logged. When Doctor A completes examination after $3.5\text{m}$ $\rightarrow$ `CONSULTATION_FINISHED` logged with `details = {"duration_min": 3.5, "delta": +0.5}`.
- **[POS-AUDIT-03] Filter Audit Logs by Actor and Action:**  
  Admin searches for all events related to `"Doctor A"` or filters by action `"CONSULTATION_FINISHED"`. System returns filtered results paginated (20 per page) in $< 30\text{ms}$.
- **[POS-AUDIT-04] Live Audit Stream (SSE):**  
  Admin stays on the Audit Log screen. As new clinic actions occur in other rooms, new log rows append to the top of the table in real time with a subtle highlight animation.

### 4.2 Negative Scenarios
- **[NEG-AUDIT-01] Non-Admin Audit Log Access:**  
  Patient or Doctor attempts to query `GET /api/admin/audit-logs`. System returns HTTP `403 Forbidden`.
- **[NEG-AUDIT-02] Audit Log Mutation Attempt:**  
  No API endpoint exists to update (`PUT/PATCH`) or delete (`DELETE`) audit records. The `audit_logs` table is strictly append-only.

### 4.3 Edge Cases
- **[EDGE-AUDIT-01] High-Concurrency Batch Events:**  
  100 patients join during morning rush. System handles asynchronous log batching via PostgreSQL multi-row inserts or buffered channels without blocking HTTP request latency.
- **[EDGE-AUDIT-02] Unauthenticated / Public Events:**  
  A public user accesses the live board or fails a login attempt. System records `user_id = NULL`, `actor_name = 'Anonymous / System'`, and stores client IP.

---

## 5. Acceptance Criteria

- [ ] Every state-changing API request produces an audit log entry in the same transaction or guaranteed post-commit hook.
- [ ] Audit logs table includes `created_at`, `user_id`, `actor_name`, `role`, `action`, `details` (JSONB), and `ip_address`.
- [ ] Log entries are strictly immutable (append-only).
- [ ] Admin UI includes search/filter controls by Action, Role, Date Range, and Keyword.
- [ ] Live audit stream broadcasts new events to admin subscribers via SSE.

---

## 6. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Comprehensive Activity Logging PRD, detailing event taxonomy, JSONB payload schemas, immutability constraints, and live SSE log streaming. |
