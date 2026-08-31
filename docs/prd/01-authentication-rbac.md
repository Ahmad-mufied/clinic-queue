# Feature Specification: Authentication & Role-Based Access Control (RBAC)
**File:** `docs/prd/01-authentication-rbac.md`  
**Status:** Approved  
**Target Roles:** `patient`, `doctor`, `admin`

---

## 1. Feature Definition & Scope

The Authentication & RBAC subsystem secures access to the platform, validates user identities, and enforces authorization rules across all REST endpoints and SSE streams using **JSON Web Tokens (JWT)** and **Casbin v2**. 

It also provides a **One-Click Demo Switcher** allowing technical evaluators and executives to switch between personas instantly without manually entering credentials.

---

## 2. User Stories

1. **As a Walk-in Patient**, I want to register and log in to the portal with my username/password so that I can securely manage my queue tickets and view my personal estimated waiting time.
2. **As a Doctor**, I want to log in to my specific doctor account so that the system associates consultations and shift logs directly with my profile.
3. **As a Clinic Administrator**, I want access to executive analytics and audit logs while restricting patients and doctors from viewing sensitive administrative data.
4. **As an Executive / Evaluator (CEO)**, I want one-click demo login buttons to test all roles immediately without memorizing passwords.

---

## 3. Case Scenarios

### 3.1 Positive Scenarios
- **[POS-AUTH-01] Valid User Login:**  
  User submits valid `username` and `password`. The system validates bcrypt hash, returns HTTP `200 OK` with a signed JWT token containing claims (`user_id`, `role`, `doctor_id`, `name`), and redirects user to their role-specific dashboard.
- **[POS-AUTH-02] Patient Self-Registration:**  
  A new patient registers with username, password, and full name. System creates a `users` record with role `patient` and returns a JWT token immediately.
- **[POS-AUTH-03] Quick Demo Login:**  
  Evaluator clicks `[Login as Doctor A]`. The frontend sends demo credentials to `/api/auth/login`, receives token, and mounts the Doctor A workspace instantly.
- **[POS-AUTH-04] Casbin Role Authorization Allowed:**  
  A user with role `doctor` calls `POST /api/doctors/call-next`. Casbin enforcer checks `p, doctor, /api/doctors/call-next, POST` $\rightarrow$ Match found $\rightarrow$ request succeeds.

### 3.2 Negative Scenarios
- **[NEG-AUTH-01] Invalid Password:**  
  User enters correct username but incorrect password. System returns HTTP `401 Unauthorized` with error message `"Invalid username or password"`.
- **[NEG-AUTH-02] Unauthorized Role Access (403 Forbidden):**  
  A logged-in patient attempts to call `GET /api/admin/stats` or `POST /api/doctors/call-next`. Casbin enforcer rejects the request $\rightarrow$ HTTP `403 Forbidden` (`"Access denied: insufficient role privileges"`).
- **[NEG-AUTH-03] Expired or Malformed Token:**  
  Client sends request with an expired, forged, or missing `Authorization: Bearer <token>` header to protected routes. System returns HTTP `401 Unauthorized`.
- **[NEG-AUTH-04] Duplicate Registration:**  
  Patient attempts to register with an existing username. System returns HTTP `409 Conflict` (`"Username already exists"`).
- **[NEG-AUTH-05] Rate Limit Exceeded (Brute-Force & Credential Stuffing Defense):**  
  An IP client exceeds the authentication rate limit (10 requests/minute, burst 5) on `POST /api/auth/login` or `POST /api/auth/register`. System enforces Token Bucket rate limiting and returns HTTP `429 Too Many Requests` (`{"error": "Too many requests. Please try again later."}`).

### 3.3 Edge Cases
- **[EDGE-AUTH-01] Doctor Account without Linked Doctor ID:**  
  A user account has role `doctor` but `doctor_id` is null. System rejects doctor action requests with HTTP `422 Unprocessable Entity` (`"Doctor profile not associated with user"`).
- **[EDGE-AUTH-02] Concurrent Logins from Multiple Devices:**  
  Same doctor logs in from two browser tabs. Both JWTs remain valid (stateless), and actions in one tab reflect in the other via SSE.
- **[EDGE-AUTH-03] Empty Authorization Header:**  
  Client sends `Authorization: Bearer ` (empty string). Middleware catches format error before processing and returns clean `401 Unauthorized`.

---

## 4. Acceptance Criteria & Identity Specification

### 4.1 Acceptance Criteria
- [ ] JWT tokens are signed using a secure secret key with a configurable expiration (default: 24h).
- [ ] Passwords stored in PostgreSQL are hashed with `bcrypt` (cost factor $\geq 10$).
- [ ] Casbin policies restrict routes according to the 3-role matrix (`patient`, `doctor`, `admin`).
- [ ] Failed login attempts and successful logins emit `AUTH_LOGIN` audit log entries.
- [ ] Pre-seeded demo credentials exist for Doctor A, Doctor B, Patient John, and Admin.
- [ ] Authentication endpoints (`/api/auth/login`, `/api/auth/register`) are protected by Token Bucket Rate Limiting (10 req/min, burst 5 per client IP) returning HTTP `429 Too Many Requests`.

### 4.2 Identity & Identifier Separation (Database UUIDv7 vs Display Username)
- **Database Identity (`id`):** 128-bit Native UUIDv7 string (e.g. `01919df4-8e3b-7412-a1f9-90b567c9e101`) for unguessable primary keys and JWT `user_id` subject claim.
- **Human Display Identity (`username` & `name`):** Memorable handles (`@doctor_a`, `@patient_john`, `@admin`) and full names (`Dr. Sarah Adams`) for UI logins, sidebar badges, and staff recognition.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Authentication & RBAC feature PRD. |
| **v1.1.0** | 2026-08-30 | Solution Architect | **Identity Design Standard** | Added Section 4.2 defining separation of internal UUIDv7 user IDs from human-facing login usernames and display names. |
| **v1.2.0** | 2026-08-31 | Backend Security Engineer | **Rate Limiting & Armor Spec** | Added [NEG-AUTH-05] and Section 4.1 acceptance criteria for Token Bucket Rate Limiting (10 req/min, burst 5 per IP) to mitigate brute-force and credential stuffing attacks. |
