"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import { useSSE } from "@/hooks/use-sse";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Users,
  Clock,
  Activity,
  ArrowUpRight,
  TrendingUp,
  Ticket,
  Stethoscope,
  BarChart3,
  MoreHorizontal,
  ArrowRight,
  SlidersHorizontal,
  Layers,
  Calendar,
  CheckCircle2,
  Loader2,
} from "lucide-react";
import type { AdminDashboardStats, HourlyPatientFlow } from "@/lib/types";

export default function AdminDashboardPage() {
  const { user, token, isLoading, isMounted, switchPersona } = useAuth();
  const { isConnected } = useSSE();
  const router = useRouter();

  // If authenticated as doctor or patient, redirect to their dedicated workspace
  useEffect(() => {
    if (!isLoading && user) {
      if (user.role === "doctor") {
        router.replace("/doctor");
      } else if (user.role === "patient") {
        router.replace("/patient");
      }
    }
  }, [user, isLoading, router]);

  // Query: Live Queue Status (SSE-aware adaptive polling)
  const { data: queueStatus } = useQuery({
    queryKey: ["queue-status"],
    queryFn: () => api.getQueueStatus(),
    enabled: !!token && user?.role === "admin",
    refetchInterval: isConnected ? 30000 : 3000,
  });

  // Query: Executive Stats (SSE-aware adaptive polling)
  const { data: adminStats } = useQuery<AdminDashboardStats>({
    queryKey: ["admin-stats"],
    queryFn: () => api.getAdminStats(),
    enabled: !!token && user?.role === "admin",
    refetchInterval: isConnected ? 30000 : 3000,
    placeholderData: () => {
      if (typeof window === "undefined") return undefined;
      try {
        const saved = localStorage.getItem("clinic_admin_stats");
        return saved ? JSON.parse(saved) : undefined;
      } catch {
        return undefined;
      }
    },
  });

  // Persist latest stats to localStorage for snappy next-mount placeholder
  useEffect(() => {
    if (adminStats) {
      localStorage.setItem("clinic_admin_stats", JSON.stringify(adminStats));
    }
  }, [adminStats]);

  if (!isMounted) {
    return null;
  }

  if (!token || user?.role !== "admin") {
    return (
      <div className="mx-auto max-w-md px-4 py-16 text-center">
        <Card className="p-8 space-y-4 rounded-2xl shadow-sm border-slate-200 dark:border-slate-800">
          <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
            <BarChart3 className="h-6 w-6" />
          </div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">Admin Dashboard Access</h2>
          <p className="text-xs text-slate-500">
            You must be authenticated with an Administrator account to view the clinic dashboard.
          </p>
          <div className="pt-2 flex flex-col gap-2">
            <Button
              onClick={async () => {
                await switchPersona("admin");
                router.replace("/dashboard");
              }}
              className="w-full rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11"
            >
              Sign In as Admin CEO
            </Button>
            <Button asChild variant="ghost" className="rounded-full text-xs">
              <Link href="/portal">Open Custom Login Portal &rarr;</Link>
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  const queueList = queueStatus?.queue_list || [];
  const waitingTickets = queueStatus?.waiting_tickets || [];
  const onlineDocs =
    queueStatus?.online_doctors_count ??
    (queueStatus?.online_doctors?.filter((d) => d.is_online).length ?? 0);
  const totalWaiting =
    queueStatus?.total_waiting ??
    (queueList.length || waitingTickets.length);
  const totalServed = adminStats?.summary?.total_served_today ?? 24;

  // Compute exact greedy estimated waiting time for the next walk-in arrival (position: totalWaiting + 1)
  const nextArrivalWaitTime = (() => {
    const rawOnlineDocs = queueStatus?.online_doctors || [];
    const activeDocs = rawOnlineDocs.filter((d) => d.is_online);
    if (activeDocs.length === 0) return null;

    // Simulation slots based on on-duty doctors
    const slots = activeDocs.map((d) => {
      const avgTime = d.avg_time && d.avg_time > 0 ? d.avg_time : 3;
      const remainingTime =
        d.status === "IN_CONSULTATION"
          ? Math.max(0, avgTime - (d.elapsed_minutes || 0))
          : 0;
      return {
        avgTime,
        nextAvailableTime: remainingTime,
      };
    });

    // Allocate preceding patients currently waiting in lobby
    const patientsAhead = totalWaiting;
    for (let i = 0; i < patientsAhead; i++) {
      slots.sort((a, b) => {
        if (a.nextAvailableTime !== b.nextAvailableTime) {
          return a.nextAvailableTime - b.nextAvailableTime;
        }
        return a.avgTime - b.avgTime;
      });
      slots[0].nextAvailableTime += slots[0].avgTime;
    }

    slots.sort((a, b) => a.nextAvailableTime - b.nextAvailableTime);
    return slots[0].nextAvailableTime;
  })();

  const hourlyData =
    adminStats?.hourly_distribution && adminStats.hourly_distribution.length > 0
      ? adminStats.hourly_distribution.map((h: HourlyPatientFlow) => {
          const heightVal =
            h.patient_count === 0
              ? "6%"
              : `${Math.min(100, Math.max(22, Math.round(20 + h.height_percentage * 0.8)))}%`;
          return {
            label: h.hour_label,
            value: h.patient_count,
            height: heightVal,
            isPeak: h.is_peak,
          };
        })
      : [
          { label: "08:00", value: 0, height: "6%", isPeak: false },
          { label: "09:00", value: 0, height: "6%", isPeak: false },
          { label: "10:00", value: 0, height: "6%", isPeak: false },
          { label: "11:00", value: 0, height: "6%", isPeak: false },
          { label: "12:00", value: 0, height: "6%", isPeak: false },
          { label: "13:00", value: 0, height: "6%", isPeak: false },
          { label: "14:00", value: 0, height: "6%", isPeak: false },
        ];

  const totalDailyFlow =
    hourlyData.reduce((acc, curr) => acc + curr.value, 0) ||
    (adminStats?.summary?.total_served_today ?? 0);
  const activeHoursCount = hourlyData.filter((h) => h.value > 0).length || 1;
  const avgPatientsPerHour = (totalDailyFlow / activeHoursCount).toFixed(1);

  return (
    <div className="space-y-8 pb-10">
      {/* Top Welcome Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
            Welcome back, {user.name}
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Operational workspace active for {user.role} ({user.username}).
          </p>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 rounded-full bg-white dark:bg-slate-800 px-4 py-2 border border-slate-200/80 shadow-2xs text-xs font-medium text-slate-700 dark:text-slate-300">
            <Calendar className="h-3.5 w-3.5 text-slate-400" />
            <span>Today, 29 Aug 2026</span>
          </div>

          <Button asChild className="rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-10 px-5 shadow-sm">
            <Link href="/admin">
              <BarChart3 className="h-3.5 w-3.5 mr-1.5" />
              Doctor Management
            </Link>
          </Button>
        </div>
      </div>

      {/* Main Operational Dashboard 2-Column Grid */}
      <div className="grid gap-6 lg:grid-cols-12 items-start">
        {/* Left Column: Examination Rooms (5 cols) */}
        <div className="lg:col-span-5">
          {/* Examination Rooms Card (Live Room Occupancy) */}
          <Card className="rounded-xl shadow-sm border-slate-200 dark:border-slate-800">
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <CardTitle className="text-base font-bold text-slate-900 dark:text-white">Examination Rooms</CardTitle>
                <p className="text-xs text-slate-400">Live consultation & room occupancy</p>
              </div>

              <div className="flex items-center gap-1.5 rounded-full bg-emerald-50 dark:bg-emerald-950/60 border border-emerald-200/60 dark:border-emerald-800/40 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-400">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                <span>{onlineDocs} Online</span>
              </div>
            </CardHeader>

            <CardContent className="space-y-3 pb-6">
              {(queueStatus?.online_doctors || [
                { id: "01919df4-8e3b-7412-a1f9-90b567c9e101", name: "Dr. Sarah Adams", avg_time: 3, is_online: false, status: "OFFLINE" },
                { id: "01919df4-8e3b-7412-a1f9-90b567c9e102", name: "Dr. Michael Chen", avg_time: 4, is_online: true, status: "IN_CONSULTATION" }
              ]).map((doc, idx) => {
                const initials = doc.name
                  ? doc.name
                      .replace(/^Dr\.?\s*/i, "")
                      .split(" ")
                      .map((n) => n[0])
                      .join("")
                      .substring(0, 2)
                      .toUpperCase()
                  : "DR";

                return (
                  <div
                    key={doc.id || idx}
                    className="p-3.5 rounded-2xl border border-slate-100 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40 flex items-center justify-between transition-all"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div
                        className={`h-9 w-9 rounded-2xl flex items-center justify-center font-bold text-xs shrink-0 ${
                          doc.is_online
                            ? "bg-emerald-100 dark:bg-emerald-900/60 text-emerald-800 dark:text-emerald-300"
                            : "bg-slate-200 dark:bg-slate-800 text-slate-500"
                        }`}
                      >
                        {initials}
                      </div>
                      <div className="min-w-0">
                        <h4 className="text-xs font-bold text-slate-900 dark:text-white truncate">{doc.name}</h4>
                        <p className="text-[11px] text-slate-400 truncate">
                          {doc.status === "IN_CONSULTATION" ? (
                            <span className="text-blue-600 dark:text-blue-400 font-medium">In consultation with patient</span>
                          ) : doc.is_online ? (
                            <span className="text-emerald-600 dark:text-emerald-400 font-medium">Ready for next patient</span>
                          ) : (
                            <span>Shift inactive &bull; Offline</span>
                          )}
                        </p>
                      </div>
                    </div>
                    <div className="text-right shrink-0 ml-2">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold ${
                          doc.status === "IN_CONSULTATION"
                            ? "bg-blue-50 text-blue-700 dark:bg-blue-950/60 dark:text-blue-400"
                            : doc.is_online
                            ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                            : "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400"
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${
                            doc.status === "IN_CONSULTATION"
                              ? "bg-blue-500 animate-pulse"
                              : doc.is_online
                              ? "bg-emerald-500"
                              : "bg-slate-400"
                          }`}
                        />
                        {doc.status === "IN_CONSULTATION"
                          ? "In Consultation"
                          : doc.is_online
                          ? "Available"
                          : "Offline"}
                      </span>
                      <span className="text-[10px] text-slate-400 block mt-0.5 font-mono">
                        Room {idx + 1}
                      </span>
                    </div>
                  </div>
                );
              })}
            </CardContent>
          </Card>
        </div>

        {/* Right Column: Patient Flow Chart + Live Queue Table (7 cols) */}
        <div className="lg:col-span-7 flex flex-col gap-6">
          {/* 1. Patient Flow Chart Card */}
          <Card className="rounded-xl shadow-sm border-slate-200 dark:border-slate-800">
            <CardHeader className="flex flex-row items-start justify-between pb-2">
              <div>
                <CardTitle className="text-base font-bold text-slate-900 dark:text-white">Patient Flow</CardTitle>
                <p className="text-xs text-slate-400 mt-1 flex items-center gap-1.5">
                  <span className="text-sm font-bold text-slate-900 dark:text-white">{totalDailyFlow} Patients Today</span>
                  <span className="text-slate-300 dark:text-slate-700">&bull;</span>
                  <span>Average {avgPatientsPerHour} patients/hour</span>
                </p>
              </div>

              <div className="flex items-center gap-1.5 rounded-full bg-slate-100 dark:bg-slate-800 border border-slate-200/60 dark:border-slate-700/50 px-3 py-1.5 text-xs font-semibold text-slate-600 dark:text-slate-300 shrink-0">
                <Clock className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                <span>08:00 – 14:00</span>
              </div>
            </CardHeader>

            <CardContent className="pt-2 pb-5">
              {/* Stylized Bar Chart Spanning Full Card Width */}
              <div className="w-full pt-4">
                <div className="h-36 w-full flex items-end justify-between gap-1.5 sm:gap-2.5 px-1 relative">
                  {hourlyData.map((bar, idx) => (
                    <div key={idx} className="flex-1 flex flex-col items-center gap-1.5 h-full justify-end group relative">
                      {/* Clean Minimalist Floating Tooltip without Badges or Indicators */}
                      <div
                        className={`absolute -top-12 left-1/2 -translate-x-1/2 z-30 transition-all duration-200 pointer-events-none ${
                          bar.isPeak && bar.value > 0
                            ? "opacity-100 scale-100 translate-y-0 group-hover:-translate-y-1"
                            : "opacity-0 scale-95 translate-y-1 group-hover:opacity-100 group-hover:scale-100 group-hover:-translate-y-1"
                        }`}
                      >
                        <div className="relative rounded-xl bg-slate-950 text-white px-2.5 py-1.5 shadow-xl border border-slate-800/90 whitespace-nowrap text-center text-[10px] leading-tight">
                          <div className="text-slate-400 font-medium">
                            {bar.isPeak && bar.value > 0 ? `Peak Hour ${bar.label}` : bar.label}
                          </div>
                          <div className={`font-bold mt-0.5 ${bar.isPeak && bar.value > 0 ? "text-emerald-400" : "text-white"}`}>
                            Intake: {bar.value} Patients
                          </div>

                          {/* Downward Arrow Pointer */}
                          <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-2 rotate-45 bg-slate-950 border-r border-b border-slate-800/90" />
                        </div>
                      </div>

                      <div
                        style={{ height: bar.height }}
                        className={`w-full max-w-[42px] rounded-t-md transition-all duration-300 ${
                          bar.isPeak && bar.value > 0
                            ? "bg-emerald-500 dark:bg-emerald-600"
                            : bar.value > 0
                            ? "bg-slate-300 dark:bg-slate-700 hover:bg-slate-400 dark:hover:bg-slate-600 cursor-pointer"
                            : "bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 cursor-pointer"
                        }`}
                      />
                      <span className="text-[11px] font-medium text-slate-400 font-mono mt-0.5">
                        {bar.label}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 2. Live Queue Card */}
          <Card className="rounded-xl shadow-sm border-slate-200 dark:border-slate-800">
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <div className="flex items-center gap-2.5">
                  <CardTitle className="text-base font-bold text-slate-900 dark:text-white">Live Queue</CardTitle>
                  <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 dark:bg-emerald-950/60 border border-emerald-200/60 dark:border-emerald-800/40 px-2.5 py-0.5 text-xs font-bold text-emerald-700 dark:text-emerald-400">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                    {totalWaiting} Waiting
                  </span>
                </div>
                <p className="text-xs text-slate-400 mt-1">Real-time checked-in patient tickets in lobby</p>
              </div>

              <div className="flex items-center gap-2">
                <Button variant="outline" size="sm" className="h-8 rounded-full text-xs font-bold px-3">
                  <SlidersHorizontal className="h-3 w-3 mr-1.5 text-slate-400" />
                  Filter
                </Button>
                <Button asChild variant="outline" size="sm" className="h-8 rounded-full text-xs font-bold px-3">
                  <Link href="/patient">View Full Queue</Link>
                </Button>
              </div>
            </CardHeader>

            <CardContent className="p-0">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-slate-100 dark:border-slate-800 text-slate-400 text-[11px] font-semibold uppercase tracking-wider">
                    <tr>
                      <th className="py-3.5 px-6">Activity / Patient</th>
                      <th className="py-3.5 px-6">Ticket Code</th>
                      <th className="py-3.5 px-6">Check-in Time</th>
                      <th className="py-3.5 px-6">Status</th>
                      <th className="py-3.5 px-6 text-right">Est. Wait</th>
                      <th className="py-3.5 px-6 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {queueList.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="py-14 text-center text-slate-400 text-xs">
                          No patients currently waiting in the clinic lobby.
                        </td>
                      </tr>
                    ) : (
                      queueList.slice(0, 8).map((t, idx) => (
                        <tr key={t.queue_number || idx} className="hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors">
                          <td className="py-4 px-6 font-semibold text-slate-900 dark:text-white">
                            <div className="flex items-center gap-3">
                              <div className="h-9 w-9 rounded-2xl bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 flex items-center justify-center font-bold text-xs">
                                {t.patient_name.charAt(0).toUpperCase()}
                              </div>
                              <div>
                                <div className="font-bold text-slate-900 dark:text-white">{t.patient_name}</div>
                                <span className="text-[10px] text-slate-400">General Consultation</span>
                              </div>
                            </div>
                          </td>
                          <td className="py-4 px-6 font-mono font-bold text-emerald-700 dark:text-emerald-400 text-sm">
                            {t.queue_number}
                          </td>
                          <td className="py-4 px-6 text-slate-500 font-mono text-xs">
                            #{idx + 1} in queue
                          </td>
                          <td className="py-4 px-6">
                            <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-amber-600 dark:text-amber-400">
                              <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />
                              Waiting in Line
                            </span>
                          </td>
                          <td className="py-4 px-6 text-right font-mono font-bold text-slate-900 dark:text-white">
                            {onlineDocs === 0 || t.estimated_wait_minutes === null || t.estimated_wait_minutes === undefined
                              ? "-"
                              : `~${t.estimated_wait_minutes} min`}
                          </td>
                          <td className="py-4 px-6 text-right">
                            <button className="text-slate-400 hover:text-slate-600 p-1">
                              <MoreHorizontal className="h-4 w-4" />
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
