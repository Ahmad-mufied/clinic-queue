import type {
  AdminDashboardStats,
  AuditLogParams,
  AuthResponse,
  ConsultationSession,
  DoctorWorkspace,
  PaginatedAuditLogs,
  QueueStatusResponse,
  QueueTicket,
  User,
} from "./types";

const TOKEN_KEY = "clinic_queue_jwt";

export function getStoredToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(TOKEN_KEY, token);
  }
}

export function clearStoredToken(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export function isTokenValid(token: string | null): boolean {
  if (!token) return false;
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return false;
    let jsonStr: string;
    if (typeof window !== "undefined") {
      const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
      jsonStr = atob(base64);
    } else {
      jsonStr = Buffer.from(parts[1], "base64").toString("utf-8");
    }
    const payload = JSON.parse(jsonStr);
    if (!payload.exp) return true;
    // Buffer by 10 seconds for clock skew
    return payload.exp * 1000 > Date.now() + 10000;
  } catch {
    return false;
  }
}

class APIClient {
  private async request<T>(
    method: string,
    endpoint: string,
    body?: any,
    customToken?: string
  ): Promise<T> {
    const token = customToken || getStoredToken();
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const baseUrl = process.env.NEXT_PUBLIC_API_URL || "";
    const url = endpoint.startsWith("http") ? endpoint : `${baseUrl}${endpoint}`;

    let response: Response;
    try {
      response = await fetch(url, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
    } catch (err: any) {
      // Retry once after 400ms on transient browser/extension fetch aborts
      await new Promise((r) => setTimeout(r, 400));
      try {
        response = await fetch(url, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
        });
      } catch {
        throw new Error(err?.message || "Failed to fetch");
      }
    }

    if (!response.ok) {
      if (response.status === 401) {
        clearStoredToken();
        if (typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("clinic:unauthorized"));
        }
      }

      let errorMsg = `HTTP ${response.status} ${response.statusText}`;
      try {
        const errorData = await response.json();
        if (errorData.error) {
          errorMsg = errorData.error;
        } else if (errorData.message) {
          errorMsg = errorData.message;
        }
      } catch {
        // Fallback to generic status text
      }
      throw new Error(errorMsg);
    }

    return response.json();
  }

  // Auth endpoints
  async login(credentials: { username: string; password: string }): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>("POST", "/api/auth/login", credentials);
    if (res.token) {
      setStoredToken(res.token);
    }
    return res;
  }

  async register(data: { username: string; password: string; name: string }): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>("POST", "/api/auth/register", data);
    if (res.token) {
      setStoredToken(res.token);
    }
    return res;
  }

  async getMe(): Promise<User> {
    return this.request<User>("GET", "/api/auth/me");
  }

  // Patient Queue endpoints
  async getQueueStatus(): Promise<QueueStatusResponse> {
    return this.request<QueueStatusResponse>("GET", "/api/queue/status");
  }

  async joinQueue(patientName: string): Promise<{ message: string; ticket: QueueTicket }> {
    return this.request<{ message: string; ticket: QueueTicket }>("POST", "/api/queue/join", {
      patient_name: patientName,
    });
  }

  async getMyTicket(): Promise<{ ticket: QueueTicket | null }> {
    try {
      return await this.request<{ ticket: QueueTicket }>("GET", "/api/queue/my-ticket");
    } catch (err: any) {
      const msg = err?.message?.toLowerCase() || "";
      if (
        msg.includes("404") ||
        msg.includes("not found") ||
        msg.includes("no active ticket") ||
        msg.includes("ticket not found") ||
        msg.includes("401") ||
        msg.includes("unauthorized") ||
        msg.includes("invalid or expired jwt")
      ) {
        return { ticket: null };
      }
      return { ticket: null };
    }
  }

  // Doctor endpoints
  async getDoctorWorkspace(): Promise<DoctorWorkspace> {
    return this.request<DoctorWorkspace>("GET", "/api/doctors/workspace");
  }

  async toggleDoctorStatus(isOnline: boolean): Promise<{ message: string; is_online: boolean }> {
    return this.request<{ message: string; is_online: boolean }>("POST", "/api/doctors/status", {
      is_online: isOnline,
    });
  }

  async callNextPatient(): Promise<ConsultationSession | { message: string }> {
    return this.request<ConsultationSession | { message: string }>("POST", "/api/doctors/call-next");
  }

  async finishConsultation(): Promise<{
    message: string;
    session_id: string;
    actual_duration_minutes: number;
    doctor_status: string;
  }> {
    return this.request<{
      message: string;
      session_id: string;
      actual_duration_minutes: number;
      doctor_status: string;
    }>("POST", "/api/doctors/finish");
  }

  // Admin endpoints
  async getAdminStats(): Promise<AdminDashboardStats> {
    return this.request<AdminDashboardStats>("GET", "/api/admin/stats");
  }

  async updateDoctorConfig(doctorId: string, avgMinutes: number): Promise<{ message: string; avg_consultation_time: number }> {
    return this.request<{ message: string; avg_consultation_time: number }>("POST", "/api/admin/doctors", {
      doctor_id: doctorId,
      avg_consultation_time_min: avgMinutes,
    });
  }

  async getAuditLogs(params?: AuditLogParams): Promise<PaginatedAuditLogs> {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", params.page.toString());
    if (params?.limit) query.set("limit", params.limit.toString());
    if (params?.cursor && typeof params.cursor === "string" && params.cursor.trim() !== "") {
      query.set("cursor", params.cursor.trim());
    }
    if (params?.search && params.search.trim() !== "") query.set("search", params.search.trim());
    if (params?.action && params.action !== "ALL") query.set("action", params.action);
    if (params?.role && params.role !== "ALL") query.set("role", params.role);
    if (params?.user_id && params.user_id.trim() !== "") query.set("user_id", params.user_id.trim());
    if (params?.start_date && params.start_date.trim() !== "") query.set("start_date", params.start_date.trim());
    if (params?.end_date && params.end_date.trim() !== "") query.set("end_date", params.end_date.trim());
    if (params?.sort_order && params.sort_order.trim() !== "") query.set("sort_order", params.sort_order.trim());

    const qs = query.toString() ? `?${query.toString()}` : "";
    return this.request<PaginatedAuditLogs>("GET", `/api/admin/audit-logs${qs}`);
  }
}

export const api = new APIClient();
