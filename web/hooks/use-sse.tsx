"use client";

import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { toast } from "sonner";
import type { SSEEventPayload } from "@/lib/types";

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
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
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

              case "TICKET_CALLED":
                queryClient.invalidateQueries({ queryKey: ["queue-status"] });
                queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
                queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
                queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
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
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });

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
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
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
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
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

              case "AUDIT_LOG_CREATED":
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs-infinite"] });
                queryClient.invalidateQueries({ queryKey: ["admin-audit-logs"] });
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
