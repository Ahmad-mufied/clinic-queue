"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useSSE } from "@/hooks/use-sse";
import { BrandLogo } from "@/components/brand-logo";
import {
  Tv,
  Stethoscope,
  Users,
  Volume2,
  Clock,
  Activity,
  Layers,
  ArrowLeft,
  Calendar,
  Sparkles,
  Radio,
  CheckCircle2,
} from "lucide-react";

export default function WaitingRoomDisplayPage() {
  const [currentTime, setCurrentTime] = useState<string>("");
  const [currentDate, setCurrentDate] = useState<string>("");
  const { isConnected } = useSSE();

  useEffect(() => {
    const update = () => {
      const now = new Date();
      setCurrentTime(
        now.toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          hour12: true,
        })
      );
      setCurrentDate(
        now.toLocaleDateString("en-US", {
          weekday: "long",
          month: "short",
          day: "numeric",
          year: "numeric",
        })
      );
    };
    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, []);

  // Query: General Queue Status with continuous fallback refetch (SSE-aware adaptive polling)
  const { data: queueStatus } = useQuery({
    queryKey: ["queue-status"],
    queryFn: () => api.getQueueStatus(),
    refetchInterval: isConnected ? 30000 : 3000,
  });

  const onlineDoctorsList = queueStatus?.online_doctors || [];
  const onlineDoctors = queueStatus?.online_doctors_count ?? onlineDoctorsList.filter((d) => d.is_online).length;
  const inConsultationDoctors = onlineDoctorsList.filter(
    (d) => d.status === "IN_CONSULTATION" || Boolean(d.current_patient)
  );
  const queueList = queueStatus?.queue_list || [];
  const totalWaiting = queueStatus?.total_waiting ?? queueList.length;

  return (
    <div className="min-h-screen bg-[#f1f4f9] dark:bg-slate-950 text-slate-900 dark:text-slate-50 p-6 sm:p-10 flex flex-col justify-between select-none">
      {/* Top Header Bar matching Oripio Clean Navigation */}
      <header className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-6 border-b border-slate-200/80 dark:border-slate-800">
        <div className="flex items-center gap-4">
          <Link href="/portal" title="Return to Portal">
            <BrandLogo size="lg" className="hover:scale-105 transition-transform" />
          </Link>
          <div>
            <h1 className="text-xl sm:text-2xl font-black tracking-tight text-slate-900 dark:text-white">
              SmartClinic Display
            </h1>
            <p className="text-xs text-slate-500 font-medium mt-0.5">
              Real-Time Patient Calling & Deterministic Queue Allocation
            </p>
          </div>
        </div>

        {/* Center: Minimalist Connection Status Indicator */}
        <div
          className={`hidden lg:flex items-center gap-2 rounded-full px-3.5 py-1.5 border shadow-2xs text-xs font-semibold transition-colors ${
            isConnected
              ? "bg-white dark:bg-slate-900 border-slate-200/80 dark:border-slate-800 text-slate-700 dark:text-slate-300"
              : "bg-amber-50 dark:bg-amber-950/40 border-amber-200 dark:border-amber-900 text-amber-700 dark:text-amber-400"
          }`}
        >
          <span
            className={`h-2 w-2 rounded-full ${
              isConnected ? "bg-emerald-500 animate-pulse" : "bg-amber-500 animate-ping"
            }`}
          />
          <span>{isConnected ? "Connected" : "Reconnecting..."}</span>
        </div>

        {/* Right: Date & Crisp Digital Clock */}
        <div className="flex items-center gap-4">
          <div className="hidden sm:flex flex-col text-right">
            <span className="text-xs text-slate-400 font-medium">Date</span>
            <span className="text-xs font-bold text-slate-700 dark:text-slate-300">{currentDate || "Loading date..."}</span>
          </div>

          <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800 px-5 py-2.5 shadow-sm text-right">
            <span className="text-[10px] text-slate-400 font-bold uppercase tracking-wider block">
              Current Time
            </span>
            <span
              suppressHydrationWarning
              className="text-2xl sm:text-3xl font-black font-mono tracking-tight text-emerald-700 dark:text-emerald-400"
            >
              {currentTime || "--:--:--"}
            </span>
          </div>
        </div>
      </header>

      {/* Main Split Display Screen */}
      <main className="grid gap-8 lg:grid-cols-12 my-8 flex-1 items-stretch">
        {/* Left Column: Now Calling / Active Consultation Rooms (7 Cols) */}
        <div className="lg:col-span-7 flex flex-col space-y-4">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 sm:h-11 sm:w-11 items-center justify-center rounded-2xl bg-emerald-600 text-white shadow-sm">
                <Volume2 className="h-5 w-5 animate-pulse" />
              </div>
              <div>
                <h2 className="text-lg sm:text-xl font-black tracking-tight text-slate-900 dark:text-white">
                  Now Calling & Consultation
                </h2>
                <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 font-medium">
                  Patients currently attending doctor rooms
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 rounded-2xl bg-white dark:bg-slate-900 px-4 py-2 border border-slate-200/80 dark:border-slate-800 shadow-xs">
              <span
                className={`h-2.5 w-2.5 rounded-full ${
                  inConsultationDoctors.length > 0 ? "bg-emerald-500 animate-pulse" : "bg-slate-300 dark:bg-slate-600"
                }`}
              />
              <span className="font-mono text-xs sm:text-sm font-bold text-slate-800 dark:text-slate-200">
                {inConsultationDoctors.length} {inConsultationDoctors.length === 1 ? "Room" : "Rooms"} Active
              </span>
            </div>
          </div>

          {inConsultationDoctors.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-12 text-center shadow-sm">
              <div className="h-20 w-20 rounded-3xl bg-emerald-50 dark:bg-emerald-950/50 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mb-5 shadow-inner">
                <Stethoscope className="h-10 w-10" />
              </div>
              <h3 className="text-xl sm:text-2xl font-black text-slate-900 dark:text-white tracking-tight">
                All Examination Rooms Ready
              </h3>
              <p className="text-xs sm:text-sm text-slate-500 mt-2 max-w-md leading-relaxed">
                Practitioners are ready and on standby. Next patient will be called shortly.
              </p>
              <div className="mt-6 flex items-center gap-2 text-xs font-bold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/60 px-4 py-2 rounded-full border border-emerald-200 dark:border-emerald-800">
                <CheckCircle2 className="h-4 w-4" />
                <span>Next ticket will be called automatically</span>
              </div>
            </div>
          ) : (
            <div className="grid gap-5 sm:grid-cols-2 flex-1">
              {inConsultationDoctors.map((doc, idx) => {
                const roomName =
                  doc.id === "01919df4-8e3b-7412-a1f9-90b567c9e101"
                    ? "Room 1"
                    : doc.id === "01919df4-8e3b-7412-a1f9-90b567c9e102"
                    ? "Room 2"
                    : onlineDoctorsList.findIndex((d) => d.id === doc.id) !== -1
                    ? `Room ${onlineDoctorsList.findIndex((d) => d.id === doc.id) + 1}`
                    : `Room ${idx + 1}`;

                return (
                  <div
                    key={doc.id}
                    className="rounded-2xl bg-emerald-700 dark:bg-emerald-900 border border-emerald-600 dark:border-emerald-800 text-white p-7 flex flex-col justify-between shadow-sm relative overflow-hidden group"
                  >
                    <div className="flex items-center justify-center border-b border-emerald-600/80 pb-4 relative z-10">
                      <span className="bg-emerald-800 text-white font-black uppercase text-sm sm:text-base rounded-full px-6 py-1.5 tracking-widest border border-emerald-600">
                        {roomName}
                      </span>
                    </div>

                    <div className="text-center py-8 relative z-10 space-y-2">
                      <span className="text-xs font-bold uppercase tracking-widest text-emerald-100/80 block">
                        Attending Practitioner
                      </span>
                      <div className="text-2xl sm:text-3xl font-black tracking-tight text-white">
                        {doc.name}
                      </div>
                      <div className="text-lg font-extrabold text-emerald-200 pt-1">
                        Patient: {doc.current_patient || "Consultation in progress"}
                      </div>
                    </div>

                    <div className="pt-4 border-t border-emerald-600/80 text-center relative z-10">
                      <span className="inline-flex items-center gap-1.5 text-xs font-bold text-emerald-50 bg-emerald-800/80 px-4 py-1.5 rounded-full">
                        <Clock className="h-3.5 w-3.5 text-emerald-300" />
                        <span>Session Duration: {doc.elapsed_minutes ?? 0}m elapsed</span>
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Right Column: Upcoming Queue in Line (5 Cols) */}
        <div className="lg:col-span-5 flex flex-col space-y-4">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 sm:h-11 sm:w-11 items-center justify-center rounded-2xl bg-emerald-600 text-white shadow-sm">
                <Users className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-lg sm:text-xl font-black tracking-tight text-slate-900 dark:text-white">
                  Upcoming in Line
                </h2>
                <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 font-medium">
                  Next patients on the waiting board
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 rounded-2xl bg-white dark:bg-slate-900 px-4 py-2 border border-slate-200/80 dark:border-slate-800 shadow-xs">
              <span
                className={`h-2.5 w-2.5 rounded-full ${
                  totalWaiting > 0 ? "bg-amber-500 animate-pulse" : "bg-slate-300 dark:bg-slate-600"
                }`}
              />
              <span className="font-mono text-xs sm:text-sm font-bold text-slate-800 dark:text-slate-200">
                {totalWaiting} Waiting
              </span>
            </div>
          </div>

          <div className="flex-1 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6 flex flex-col justify-between shadow-sm">
            {queueList.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center text-center p-8 space-y-3">
                <div className="h-16 w-16 rounded-2xl bg-slate-100 dark:bg-slate-800 text-slate-400 flex items-center justify-center">
                  <Users className="h-8 w-8" />
                </div>
                <h4 className="text-sm font-bold text-slate-700 dark:text-slate-300">
                  No Patients in Queue
                </h4>
                <p className="text-xs text-slate-400 max-w-xs">
                  New checked-in walk-in tickets will automatically appear here.
                </p>
              </div>
            ) : (
              <div className="space-y-3 flex-1 overflow-y-auto max-h-[500px] pr-1">
                {queueList.slice(0, 6).map((ticket, idx) => (
                  <div
                    key={ticket.queue_number || idx}
                    className="flex items-center justify-between p-4 rounded-2xl border border-slate-100 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40 hover:border-emerald-500/40 hover:bg-emerald-50/20 transition-all"
                  >
                    <div className="flex items-center gap-3.5">
                      <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-white dark:bg-slate-700 text-slate-800 dark:text-white font-mono font-black text-xs shadow-2xs border border-slate-200/60 dark:border-slate-600">
                        #{idx + 1}
                      </div>
                      <div>
                        <div className="font-mono text-xl font-black text-slate-900 dark:text-white tracking-tight">
                          {ticket.queue_number}
                        </div>
                        <div className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                          {ticket.patient_name}
                        </div>
                      </div>
                    </div>

                    <div className="text-right">
                      <span className="text-[10px] text-slate-400 font-bold uppercase tracking-wider block">
                        Est. Wait
                      </span>
                      <span className="text-sm font-extrabold font-mono text-emerald-700 dark:text-emerald-400">
                        {ticket.estimated_wait_minutes !== null && ticket.estimated_wait_minutes !== undefined
                          ? `~${ticket.estimated_wait_minutes} min`
                          : "-"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

          </div>
        </div>
      </main>

      {/* Modern Ticker Footer */}
      <footer className="pt-6 border-t border-slate-200/80 dark:border-slate-800 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-4">
        <div className="flex items-center gap-2 font-medium">
          <span className="font-bold text-slate-900 dark:text-white">SmartClinic Healthcare OS</span>
          <span>&bull;</span>
          <span>Live Public Queue Monitor</span>
        </div>

        <div className="flex items-center gap-4">
          <span className="text-[11px] text-slate-400">
            Please approach your designated room when called.
          </span>
          <Link
            href="/portal"
            className="inline-flex items-center gap-1.5 text-xs font-bold text-emerald-700 dark:text-emerald-400 hover:underline"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            <span>Sign In to Portal</span>
          </Link>
        </div>
      </footer>
    </div>
  );
}
