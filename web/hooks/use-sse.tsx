"use client";

import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import type { InfiniteData } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { toast } from "sonner";
import type { AuditLog, PaginatedAuditLogs, SSEEventPayload } from "@/lib/types";
import { formatAuditLocation } from "@/lib/geo";

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  description: string;
  timeFormatted: string;
  category: "queue" | "doctor" | "consultation" | "admin";
  read: boolean;
}

interface SSEContextType {
  isConnected: boolean;
  lastEvent: SSEEventPayload | null;
  notifications: AppNotification[];
  unreadCount: number;
  markAllAsRead: () => void;
  clearNotifications: () => void;
}

const SSEContext = createContext<SSEContextType>({
  isConnected: false,
  lastEvent: null,
  notifications: [],
  unreadCount: 0,
  markAllAsRead: () => {},
  clearNotifications: () => {},
});

export function SSEProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<SSEEventPayload | null>(null);
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const eventSourceRef = useRef<EventSource | null>(null);
  const userRef = useRef(user);

  useEffect(() => {
    userRef.current = user;
  }, [user]);

  const markAllAsRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  };

  const clearNotifications = () => {
    setNotifications([]);
  };

  const unreadCount = notifications.filter((n) => !n.read).length;

  useEffect(() => {
    let reconnectTimeout: NodeJS.Timeout | null = null;
    let isUnmounted = false;
    let retryCount = 0;

    function connect() {
      if (isUnmounted) return;

      try {
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
        }

        const baseApiUrl = process.env.NEXT_PUBLIC_API_URL || "";
        const sseUrl = baseApiUrl
          ? `${baseApiUrl}/api/events`
          : typeof window !== "undefined" && window.location.hostname === "localhost"
          ? "http://localhost:8080/api/events"
          : "/api/events";
        const es = new EventSource(sseUrl);
        eventSourceRef.current = es;

        es.onopen = () => {
          if (!isUnmounted) {
            setIsConnected(true);
            retryCount = 0; // Reset retry count on successful connection
          }
        };

        es.onmessage = (e) => {
          if (isUnmounted) return;
          try {
            const raw = JSON.parse(e.data);
            const eventType = (raw.type || raw.event || "").toUpperCase();
            if (!eventType || eventType === "CONNECTED") return;

            const payload: SSEEventPayload = {
              type: eventType as any,
              data: raw.data || raw,
              timestamp: raw.timestamp || new Date().toISOString(),
            };
            setLastEvent(payload);

            const now = new Date();
            const timeFormatted = now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
            const notifId = `${eventType}-${Date.now()}-${Math.random().toString(36).substring(2, 6)}`;

            let newNotif: AppNotification | null = null;
            const currentUser = userRef.current;
            const isAdmin = currentUser?.role === "admin";

            // Invalidate relevant query keys and construct notification based on event taxonomy
            switch (eventType) {
              case "QUEUE_JOINED":
              case "QUEUE_UPDATED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
                newNotif = {
                  id: notifId,
                  type: "QUEUE_UPDATED",
                  title: "Queue Synchronized",
                  description: "Clinic lobby patient queue updated in real-time.",
                  timeFormatted,
                  category: "queue",
                  read: false,
                };
                break;

              case "QUEUE_CANCELLED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });

                const isCancelledPatient =
                  currentUser?.role === "patient" &&
                  (currentUser?.name?.trim().toLowerCase() === payload.data?.patient_name?.trim().toLowerCase() ||
                   currentUser?.id === payload.data?.user_id);

                if (isCancelledPatient) {
                  // Multi-tab reliability: clean up cached ticket in localStorage if matches cancelled ticket
                  if (typeof window !== "undefined") {
                    try {
                      const saved = localStorage.getItem("clinic_queue_ticket");
                      if (saved) {
                        const parsed = JSON.parse(saved);
                        if (parsed?.id === payload.data?.ticket_id || parsed?.queue_number === payload.data?.queue_number) {
                          localStorage.removeItem("clinic_queue_ticket");
                        }
                      }
                    } catch {
                      // ignore parse error
                    }
                  }

                  const wasCancelledByAdmin = payload.data?.role === "admin" || (payload.data?.cancelled_by && payload.data?.cancelled_by !== currentUser?.id);
                  toast.info("Queue Ticket Cancelled", {
                    id: `queue-cancel-${payload.data?.ticket_id || payload.data?.queue_number || "patient"}`,
                    description: wasCancelledByAdmin
                      ? `Your ticket ${payload.data?.queue_number || ""} was cancelled by clinic administration.`
                      : `Your ticket ${payload.data?.queue_number || ""} has been cancelled.`,
                  });
                } else if (isAdmin) {
                  toast.info(`Ticket Cancelled: ${payload.data?.queue_number || ""}`, {
                    id: `queue-cancel-${payload.data?.ticket_id || payload.data?.queue_number || "admin"}`,
                    description: `Patient ${payload.data?.patient_name || "Patient"} cancelled their queue ticket.`,
                  });
                }

                newNotif = {
                  id: notifId,
                  type: "QUEUE_CANCELLED",
                  title: `Ticket Cancelled: ${payload.data?.queue_number || ""}`,
                  description: `Patient ${payload.data?.patient_name || "Patient"} cancelled ticket.`,
                  timeFormatted,
                  category: "queue",
                  read: false,
                };
                break;

              case "TICKET_CALLED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
                const docCalledName =
                  payload.data?.doctor_name ||
                  (payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e101"
                    ? "Dr. Sarah Adams"
                    : payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e102"
                    ? "Dr. Michael Chen"
                    : "Doctor Room");

                const isTargetPatient =
                  currentUser?.role === "patient" &&
                  (currentUser?.name?.trim().toLowerCase() === payload.data?.patient_name?.trim().toLowerCase() ||
                   currentUser?.id === payload.data?.user_id);

                // Role-aware toast filtering to guarantee zero duplicate notifications:
                if (payload.data?.patient_name) {
                  if (isTargetPatient) {
                    toast.info(`YOUR TICKET IS CALLED: ${payload.data.patient_name}`, {
                      description: `Please proceed immediately to ${docCalledName}!`,
                      duration: 8000,
                    });
                  } else if (isAdmin) {
                    toast.info(`Ticket Called: ${payload.data.patient_name}`, {
                      description: `Room: ${docCalledName}`,
                    });
                  }
                }

                newNotif = {
                  id: notifId,
                  type: eventType,
                  title: `Patient Called: ${payload.data?.patient_name || "Patient"}`,
                  description: `Proceed to Examination Room (${docCalledName}).`,
                  timeFormatted,
                  category: "consultation",
                  read: false,
                };
                break;

              case "TICKET_FINISHED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });

                const isFinishingPatient =
                  currentUser?.role === "patient" &&
                  (currentUser?.name?.trim().toLowerCase() === payload.data?.patient_name?.trim().toLowerCase() ||
                   currentUser?.id === payload.data?.user_id);

                if (payload.data?.patient_name) {
                  if (isFinishingPatient) {
                    toast.success("Consultation Completed", {
                      description: "Your examination is complete. Thank you for visiting SmartClinic!",
                      duration: 6000,
                    });
                  } else if (isAdmin) {
                    toast.success("Consultation Finished", {
                      description: `Patient ${payload.data.patient_name} examination completed.`,
                    });
                  }
                }
                newNotif = {
                  id: notifId,
                  type: eventType,
                  title: `Consultation Finished: ${payload.data?.patient_name || "Patient"}`,
                  description: `Examination session finalized (${payload.data?.actual_duration_minutes ?? payload.data?.duration_minutes ?? 0}m).`,
                  timeFormatted,
                  category: "consultation",
                  read: false,
                };
                break;

              case "DOCTOR_STATUS_CHANGED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
                const doctorName =
                  payload.data?.name ||
                  payload.data?.doctor_name ||
                  (payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e101"
                    ? "Dr. Sarah Adams"
                    : payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e102"
                    ? "Dr. Brian Miller"
                    : "Practitioner");
                const statusText = payload.data?.is_online ? "ONLINE" : "OFFLINE";
                newNotif = {
                  id: notifId,
                  type: eventType,
                  title: `Doctor Shift: ${doctorName}`,
                  description: `${doctorName} is now ${statusText}.`,
                  timeFormatted,
                  category: "doctor",
                  read: false,
                };
                break;

              case "DOCTOR_CONFIG_UPDATED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
                const configDocName =
                  payload.data?.name ||
                  payload.data?.doctor_name ||
                  (payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e101"
                    ? "Dr. Sarah Adams"
                    : payload.data?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e102"
                    ? "Dr. Brian Miller"
                    : "Practitioner");
                newNotif = {
                  id: notifId,
                  type: eventType,
                  title: "Target Speed Configured",
                  description: `${configDocName} target speed set to ${payload.data?.avg_consultation_time_min} min.`,
                  timeFormatted,
                  category: "admin",
                  read: false,
                };
                break;

              case "AUDIT_LOG_CREATED": {
                const rawLog = payload.data as Partial<AuditLog> | undefined;
                if (rawLog && rawLog.id) {
                  const formattedLog: AuditLog = {
                    id: rawLog.id,
                    user_id: rawLog.user_id,
                    actor_name: rawLog.actor_name || "System",
                    role: rawLog.role || "public",
                    action: rawLog.action || "UNKNOWN",
                    details: rawLog.details || {},
                    ip_address: rawLog.ip_address || "127.0.0.1",
                    location: formatAuditLocation({
                      location: rawLog.location,
                      ip_address: rawLog.ip_address,
                      details: rawLog.details,
                    }),
                    created_at: rawLog.created_at || new Date().toISOString(),
                  };

                  // Delta cache update: incrementally prepend only the new record to all matching active infinite queries without triggering network refetches
                  queryClient.getQueryCache().findAll({ queryKey: ["admin-audit-logs-infinite"] }).forEach((query) => {
                    const qKey = query.queryKey as [string, number?, string?, string?, string?, string?, string?, string?];
                    const [, , searchFilter, , , sortOrderFilter, actionFilter, roleFilter] = qKey;

                    // Only prepend to newest-first (desc) views
                    if (sortOrderFilter === "asc") {
                      return;
                    }

                    // Check action filter
                    if (actionFilter && actionFilter !== "ALL" && formattedLog.action !== actionFilter) {
                      return;
                    }

                    // Check role filter
                    if (roleFilter && roleFilter !== "ALL" && formattedLog.role !== roleFilter) {
                      return;
                    }

                    // Check search query keyword
                    if (searchFilter && searchFilter.trim() !== "") {
                      const term = searchFilter.trim().toLowerCase();
                      const matchesActor = formattedLog.actor_name.toLowerCase().includes(term);
                      const matchesAction = formattedLog.action.toLowerCase().includes(term);
                      const matchesIP = (formattedLog.ip_address || "").toLowerCase().includes(term);
                      const matchesLocation = (formattedLog.location || "").toLowerCase().includes(term);
                      if (!matchesActor && !matchesAction && !matchesIP && !matchesLocation) {
                        return;
                      }
                    }

                    queryClient.setQueryData<InfiniteData<PaginatedAuditLogs>>(query.queryKey, (old) => {
                      if (!old || !old.pages || old.pages.length === 0) return old;

                      // Prevent duplicate entry insertion
                      const alreadyExists = old.pages.some((p) => p.logs.some((l) => l.id === formattedLog.id));
                      if (alreadyExists) return old;

                      const firstPage = old.pages[0];
                      const updatedFirstPage: PaginatedAuditLogs = {
                        ...firstPage,
                        total_records: (firstPage.total_records || 0) + 1,
                        logs: [formattedLog, ...firstPage.logs],
                      };

                      return {
                        ...old,
                        pages: [updatedFirstPage, ...old.pages.slice(1)],
                      };
                    });
                  });
                }

                newNotif = {
                  id: notifId,
                  type: eventType,
                  title: "Audit Trail Entry Logged",
                  description: `Action ${payload.data?.action || "activity"} recorded in system trail.`,
                  timeFormatted,
                  category: "admin",
                  read: false,
                };
                break;
              }
            }

            if (newNotif) {
              setNotifications((prev) => [newNotif!, ...prev.slice(0, 19)]);
            }
          } catch {
            // Ignore parse errors on ping/handshake frames
          }
        };

        es.onerror = () => {
          if (isUnmounted) return;
          setIsConnected(false);
          if (es.readyState === EventSource.CLOSED || es.readyState === EventSource.CONNECTING) {
            es.close();
            if (reconnectTimeout) clearTimeout(reconnectTimeout);
            
            // Exponential backoff: 500ms, 1s, 2s, up to max 5s
            const delay = Math.min(500 * Math.pow(2, retryCount), 5000);
            retryCount++;
            
            reconnectTimeout = setTimeout(connect, delay);
          }
        };
      } catch {
        // Prevent unhandled rejection on network failure
      }
    }

    connect();

    return () => {
      isUnmounted = true;
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [queryClient]);

  return (
    <SSEContext.Provider
      value={{
        isConnected,
        lastEvent,
        notifications,
        unreadCount,
        markAllAsRead,
        clearNotifications,
      }}
    >
      {children}
    </SSEContext.Provider>
  );
}

export function useSSE() {
  return useContext(SSEContext);
}
