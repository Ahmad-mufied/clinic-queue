# AGENTS.md: Backend Workflow & Golang Development Guidelines
**Project:** Smart Clinic Queue & Analytics Platform  
**Target Environment:** Go 1.27 | PostgreSQL 18 | NATS JetStream | Echo v4 | Casbin v2

---

## 1. Development Lifecycle Rules & Coding Standards

All Go backend development and Git operations strictly comply with the rules in:  
👉 **[`.agents/rules/golang-development-lifecycle.md`](.agents/rules/golang-development-lifecycle.md)** (Backend Go Lifecycle & Patterns)  
👉 **[`.agents/rules/git-workflow.md`](.agents/rules/git-workflow.md)** (Git Manual Commit Policy)  

---

## 2. Feature Execution Roadmap & Documentation References

Backend development strictly follows the phased roadmap detailed in the documentation:

1. **System Architecture & Hexagonal Design:**  
   👉 Reference: [`docs/tech/ARCH.md`](docs/tech/ARCH.md)
2. **Feature 01: Authentication & Casbin RBAC (In Progress):**  
   👉 PRD: [`docs/prd/01-authentication-rbac.md`](docs/prd/01-authentication-rbac.md)  
   👉 Tech Spec: [`docs/tech/01-auth-rbac-tech.md`](docs/tech/01-auth-rbac-tech.md)  
   👉 Plan: [`docs/plan/01-auth-rbac-plan.md`](docs/plan/01-auth-rbac-plan.md)
3. **Feature 02: Queue Engine & Real-Time SSE:**  
   👉 PRD: [`docs/prd/02-patient-queue.md`](docs/prd/02-patient-queue.md) | Tech: [`docs/tech/02-queue-calculator-tech.md`](docs/tech/02-queue-calculator-tech.md)
4. **Feature 03: Doctor Workspace & Shift Management:**  
   👉 PRD: [`docs/prd/03-doctor-workspace.md`](docs/prd/03-doctor-workspace.md) | Tech: [`docs/tech/03-doctor-workspace-tech.md`](docs/tech/03-doctor-workspace-tech.md)
5. **Feature 04 & 05: Admin Analytics & Audit Trail:**  
   👉 PRD: [`docs/prd/04-admin-analytics.md`](docs/prd/04-admin-analytics.md) & [`docs/prd/05-audit-trail.md`](docs/prd/05-audit-trail.md)  
   👉 Tech: [`docs/tech/04-admin-analytics-tech.md`](docs/tech/04-admin-analytics-tech.md) & [`docs/tech/05-audit-trail-tech.md`](docs/tech/05-audit-trail-tech.md)

## 3. Local Skills Catalog (`.agents/skills/`)

The following specialized skills are installed locally for this workspace:
- **Golang Engineering:** `use-modern-go`, `golang-code-style`, `golang-testing`, `golang-error-handling`, `golang-concurrency`, `golang-design-patterns`, `golang-database`, `golang-structs-interfaces`, `golang-project-layout`, `golang-dependency-injection`, `golang-lint`, `golang-security`
- **Backend & System Design:** `api-designer`, `system-design`, `postgres-pro`, `sql-pro`
- **Frontend & UI:** `vue-expert`, `vue-expert-js`
- **QA & Security:** `test-master`, `security-reviewer`

---

## 4. Quick Verification Commands

```bash
# Run all unit tests with race detection and 100% target coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -func=coverage.out

# Code vetting
go vet ./...
```
