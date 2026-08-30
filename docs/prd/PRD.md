# Product Requirement Document (PRD) Master Overview
## Smart Clinic Queue Management & Business Intelligence Platform

---

## 1. Product Vision & Goals

### 1.1 Vision
To eliminate unpredictable and frustrating clinic waiting room experiences by delivering a **real-time, transparent, and intelligent queue platform**. Walk-in patients receive dynamic, minute-accurate wait time countdowns, while doctors and administrators gain operational control, automated audit trails, and executive performance analytics.

### 1.2 Core Business Objectives
- **Reduce Perceived Wait Time:** Dynamic, real-time wait estimation minimizes patient anxiety and waiting room overcrowding.
- **Doctor Workflow Optimization:** Doctors manage patient admission and shift statuses seamlessly.
- **Executive Visibility & Operational Health:** Administrators track actual vs estimated wait times, doctor productivity rates, and clinic throughput.
- **Compliance & Transparency:** Full audit trail tracking every authentication, shift change, queue entry, and consultation.

---

## 2. User Personas

| Persona | Role | Core Goals & Needs |
| :--- | :--- | :--- |
| **Patient** | Walk-in Patient | Checks in online, monitors real-time estimated wait time, receives alerts when called into a doctor's room. |
| **Doctor** | Medical Practitioner | Toggles Online/Offline status, calls the next patient, tracks consultation duration, completes examinations. |
| **Administrator / CEO** | Executive / Clinic Manager | Monitors clinic throughput, doctor utilization rates, reviews audit logs, adjusts clinic parameters. |
| **Evaluator / Reviewer** | Technical Reviewer / CEO | Evaluates system capabilities rapidly using pre-seeded accounts and one-click demo switchers. |

---

## 3. Modular Feature Structure

The PRD is modularized into dedicated feature specification documents:

```
docs/prd/
├── PRD.md                       # (This Document) Master Overview & System Architecture
├── 01-authentication-rbac.md    # Feature 1: Authentication & Role-Based Access Control (Casbin)
├── 02-patient-queue.md          # Feature 2: Patient Online Walk-in Queue & Live Wait Estimation
├── 03-doctor-workspace.md       # Feature 3: Doctor Shift Management & Consultation Workflow
├── 04-admin-analytics.md        # Feature 4: Executive Business Analytics & Doctor Productivity
└── 05-audit-trail.md            # Feature 5: Comprehensive Activity Logging & Audit Trail
```

---

## 4. System Role & Permission Matrix

```
┌──────────────────────────────────────┬─────────┬─────────┬─────────┐
│ Feature / Capability                 │ Patient │ Doctor  │  Admin  │
├──────────────────────────────────────┼─────────┼─────────┼─────────┤
│ Login / Logout / Session Profile     │   Yes   │   Yes   │   Yes   │
│ Join Queue / Walk-in Registration    │   Yes   │   No    │   Yes   │
│ View Live Personal Ticket & Wait Time│   Yes   │   No    │   Yes   │
│ Toggle Doctor Shift (Online/Offline) │   No    │   Yes   │   Yes   │
│ Call Next Patient from Queue         │   No    │   Yes   │   No    │
│ Complete Consultation Session        │   No    │   Yes   │   No    │
│ View Executive Analytics & KPIs      │   No    │   No    │   Yes   │
│ View Doctor Productivity Table       │   No    │   No    │   Yes   │
│ View Full Audit Trail Logs           │   No    │   No    │   Yes   │
│ Live Public Queue Monitor (SSE)      │   Yes   │   Yes   │   Yes   │
└──────────────────────────────────────┴─────────┴─────────┴─────────┘
```

---

---

## 5. Identity & Human-Readable Reference Standards (Dual Identity Design)

To balance **database integrity and cryptographic security** with **intuitive clinical user experience (UX)**, the system enforces a strict separation between technical entity identifiers and human-facing display codes:

| Entity Type | Database Internal ID (Technical) | Human-Facing Display Reference (UI / Audio / Staff) | Use Case & Purpose |
| :--- | :--- | :--- | :--- |
| **Patient Queue Ticket** | `id UUIDv7` (`01919df4-...`) | `queue_number` (`A-01`, `A-11`, `B-03`) | Displayed on waiting room screens, printed on physical slips, called via audio speakers. |
| **User Account** | `id UUIDv7` (`01919df4-...`) | `username` (`@doctor_a`, `@patient_john`, `@admin`) | Used for login credentials, audit actor badges, and human-friendly account references. |
| **Doctor Profile** | `id UUIDv7` (`01919df4-...`) | `name` (`Dr. Sarah Adams`) & Role Badge | Displayed on consultation room boards, public clinic status monitors, and doctor workspace. |
| **Audit Activity Log** | `id UUIDv7` (`01919df4-...`) | `actor_name` (`Dr. Sarah Adams (@doctor_a)`) | Highlighted in UI table cards with underlying UUIDv7 preserved in the forensic JSON payload. |

---

## 6. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Master PRD document establishing product vision, user personas, modular documentation structure, and global system permission matrix. |
| **v1.1.0** | 2026-08-30 | Solution Architect | **Identity Design Standard** | Added Section 5 specifying Dual Identity Design separating internal PostgreSQL 18 UUIDv7 keys from human-readable clinical display codes (`queue_number`, `username`, `actor_name`). |
