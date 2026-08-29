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
| **🧑‍🦱 Patient** | Walk-in Patient | Checks in online, monitors real-time estimated wait time, receives alerts when called into a doctor's room. |
| **👨‍⚕️ Doctor** | Medical Practitioner | Toggles Online/Offline status, calls the next patient, tracks consultation duration, completes examinations. |
| **👔 Administrator / CEO** | Executive / Clinic Manager | Monitors clinic throughput, doctor utilization rates, reviews audit logs, adjusts clinic parameters. |
| **⚡ Evaluator / Reviewer** | Technical Reviewer / CEO | Evaluates system capabilities rapidly using pre-seeded accounts and one-click demo switchers. |

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
│ Login / Logout / Session Profile     │    ✅   │    ✅   │    ✅   │
│ Join Queue / Walk-in Registration    │    ✅   │    ❌   │    ✅   │
│ View Live Personal Ticket & Wait Time│    ✅   │    ❌   │    ✅   │
│ Toggle Doctor Shift (Online/Offline) │    ❌   │    ✅   │    ✅   │
│ Call Next Patient from Queue         │    ❌   │    ✅   │    ❌   │
│ Complete Consultation Session        │    ❌   │    ✅   │    ❌   │
│ View Executive Analytics & KPIs      │    ❌   │    ❌   │    ✅   │
│ View Doctor Productivity Table       │    ❌   │    ❌   │    ✅   │
│ View Full Audit Trail Logs           │    ❌   │    ❌   │    ✅   │
│ Live Public Queue Monitor (SSE)      │    ✅   │    ✅   │    ✅   │
└──────────────────────────────────────┴─────────┴─────────┴─────────┘
```

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Master PRD document establishing product vision, user personas, modular documentation structure, and global system permission matrix. |
