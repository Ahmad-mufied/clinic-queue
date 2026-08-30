"use client";

import React, { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useSSE } from "@/hooks/use-sse";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Stethoscope,
  User,
  Clock,
  CheckCircle2,
  PhoneCall,
  Loader2,
  Users,
  Activity,
  ArrowRight,
  AlertCircle,
  Radio,
  Timer,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";
import { formatSecondsToTimer } from "@/lib/utils";

export default function DoctorWorkspacePage() {
  const { user, token, isLoading: isAuthLoading, isMounted, switchPersona } = useAuth();
  const { isConnected } = useSSE();
  const queryClient = useQueryClient();
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  // Query: Doctor Workspace Status (SSE-aware adaptive polling)
  const { data: workspace, isLoading: isWorkspaceLoading } = useQuery({
    queryKey: ["doctor-workspace", user?.id],
    queryFn: () => api.getDoctorWorkspace(),
    enabled: !!token && user?.role === "doctor",
    refetchInterval: isConnected ? 30000 : 3000,
    initialData: () => {
      if (typeof window === "undefined") return undefined;
      try {
        const saved = localStorage.getItem("clinic_doctor_workspace");
        return saved ? JSON.parse(saved) : undefined;
      } catch {
        return undefined;
      }
    },
  });

  // Query: Queue Status for Waiting List (SSE-aware adaptive polling)
  const { data: queueStatus } = useQuery({
    queryKey: ["queue-status"],
    queryFn: () => api.getQueueStatus(),
    refetchInterval: isConnected ? 30000 : 3000,
  });

  const activeSession = workspace?.active_session;

  // Persist workspace to localStorage
  useEffect(() => {
    if (workspace) {
      localStorage.setItem("clinic_doctor_workspace", JSON.stringify(workspace));
    }
  }, [workspace]);

  // Active Session Timer
  useEffect(() => {
    if (!activeSession || !activeSession.started_at) {
      setElapsedSeconds(0);
      return;
    }

    const startTime = new Date(activeSession.started_at).getTime();
    const updateTimer = () => {
      const now = Date.now();
      const diffSecs = Math.max(0, Math.floor((now - startTime) / 1000));
      setElapsedSeconds(diffSecs);
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [activeSession]);

  // Mutation: Toggle Shift Status
  const toggleStatusMutation = useMutation({
    mutationFn: (isOnline: boolean) => api.toggleDoctorStatus(isOnline),
    onSuccess: (data) => {
      toast.success(data.is_online ? "Shift Started: You are now ONLINE" : "Shift Ended: You are now OFFLINE");
      queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
      queryClient.invalidateQueries({ queryKey: ["queue-status"] });
      queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
    },
    onError: (err: any) => {
      toast.error("Failed to update status", { description: err.message });
    },
  });

  // Mutation: Call Next Patient
  const callNextMutation = useMutation({
    mutationFn: () => api.callNextPatient(),
    onSuccess: (data: any) => {
      if (data.message && data.message.includes("empty")) {
        toast.info("Queue is empty", { description: "No waiting patients currently in the lobby." });
      } else {
        toast.success(`Patient Called: ${data.patient_name}`, {
          description: `Queue ticket admitted to examination room.`,
        });
      }
      queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
      queryClient.invalidateQueries({ queryKey: ["queue-status"] });
      queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
    },
    onError: (err: any) => {
      toast.error("Failed to call next patient", { description: err.message });
    },
  });

  // Mutation: Finish Consultation
  const finishMutation = useMutation({
    mutationFn: () => api.finishConsultation(),
    onSuccess: (data: any) => {
      const duration = data?.actual_duration_minutes ?? data?.duration_minutes ?? 0;
      toast.success("Consultation Completed", {
        description: `Duration: ${duration} minutes. Examination room is now AVAILABLE.`,
      });
      queryClient.invalidateQueries({ queryKey: ["doctor-workspace"] });
      queryClient.invalidateQueries({ queryKey: ["queue-status"] });
      queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
    },
    onError: (err: any) => {
      toast.error("Failed to complete consultation", { description: err.message });
    },
  });

  if (!isMounted) {
    return null;
  }

  if (!token || user?.role !== "doctor") {
    return (
      <div className="mx-auto max-w-md px-4 py-16 text-center">
        <Card className="p-8 space-y-4">
          <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
            <Stethoscope className="h-6 w-6" />
          </div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">Practitioner-Only Room</h2>
          <p className="text-xs text-slate-500">
            {user?.role === "admin"
              ? "Super Admin / CEO monitors doctor performance and configures consultation parameters from Doctor Management, but does not conduct medical consultations."
              : "You must be authenticated as a Doctor to open an examination room and call waiting patients."}
          </p>
          <div className="pt-2 flex flex-col gap-2">
            {user?.role === "admin" ? (
              <Button asChild className="rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11">
                <Link href="/admin">Go to Doctor Management Panel &rarr;</Link>
              </Button>
            ) : (
              <>
                <Button
                  onClick={() => switchPersona("doctor_a")}
                  className="rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11"
                >
                  Sign In as Dr. Sarah Adams (Doctor A)
                </Button>
                <Button
                  onClick={() => switchPersona("doctor_b")}
                  variant="outline"
                  className="rounded-full font-bold text-xs h-11"
                >
                  Sign In as Dr. Brian Miller (Doctor B)
                </Button>
              </>
            )}
          </div>
        </Card>
      </div>
    );
  }

  const isDoctorA =
    workspace?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e101" ||
    user?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e101";
  const isDoctorB =
    workspace?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e102" ||
    user?.doctor_id === "01919df4-8e3b-7412-a1f9-90b567c9e102";

  const roomName = isDoctorA ? "Room 1" : isDoctorB ? "Room 2" : "Examination Suite";
  const doctorName = workspace?.doctor_name || user?.name || "Practitioner";
  const targetAvgPace = workspace?.avg_consultation_time || 3;

  const waitingList = queueStatus?.queue_list || [];
  const waitingCount = queueStatus?.total_waiting ?? waitingList.length;
  const isQueueEmpty = waitingCount === 0;
  const isOffline = !workspace?.is_online;
  const nextPatient = waitingList.length > 0 ? waitingList[0] : null;

  return (
    <div className="space-y-6 pb-12">
      {/* Top Header Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800 rounded-3xl p-5 sm:p-6 shadow-xs">
        <div className="flex items-center gap-3.5">
          <div className="h-11 w-11 rounded-2xl bg-emerald-600 text-white flex items-center justify-center shadow-xs shrink-0">
            <Stethoscope className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-white">
              {roomName} &bull; Doctor Workspace
            </h1>
            <p className="text-xs text-slate-500 mt-0.5">
              Doctor: <strong className="text-slate-800 dark:text-slate-200 font-semibold">{doctorName}</strong> &bull; Target Pace: <strong className="text-slate-800 dark:text-slate-200 font-semibold">{targetAvgPace} min / patient</strong>
            </p>
          </div>
        </div>

        {/* Shift Status Toggle */}
        <div className="flex items-center gap-3 self-start sm:self-auto bg-slate-50 dark:bg-slate-800/80 px-4 py-2.5 rounded-2xl border border-slate-200/70 dark:border-slate-700">
          <div className="flex flex-col text-right">
            <span className="text-xs font-bold text-slate-900 dark:text-white">
              {workspace?.is_online ? "Active on Shift" : "Shift Offline"}
            </span>
            <span className="text-[10px] text-slate-400">
              {workspace?.is_online ? "Accepting patients" : "On break"}
            </span>
          </div>
          <Switch
            checked={workspace?.is_online || false}
            onCheckedChange={(checked) => toggleStatusMutation.mutate(checked)}
            disabled={toggleStatusMutation.isPending || !!activeSession}
          />
        </div>
      </div>

      {/* Main 2-Column Grid */}
      <div className="grid gap-6 lg:grid-cols-12">
        {/* Left Column: Examination Room (7 Cols) */}
        <div className="lg:col-span-7 space-y-6">
          {!isMounted || (isWorkspaceLoading && !workspace) ? (
            <div className="rounded-3xl bg-slate-100 dark:bg-slate-800/60 p-8 h-[380px] animate-pulse border border-slate-200/80 dark:border-slate-800 flex flex-col items-center justify-center gap-3">
              <div className="h-14 w-14 rounded-2xl bg-slate-200 dark:bg-slate-700 animate-pulse" />
              <div className="h-4 w-48 rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse" />
            </div>
          ) : activeSession ? (
            /* Active Consultation Card */
            (() => {
              const targetSec = targetAvgPace * 60;
              const isOvertime = elapsedSeconds > targetSec;
              const remainingSec = Math.max(0, targetSec - elapsedSeconds);
              const overtimeSec = isOvertime ? elapsedSeconds - targetSec : 0;
              const progress = Math.min(100, Math.round((elapsedSeconds / targetSec) * 100));

              return (
                <div className="rounded-3xl bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 p-6 sm:p-8 shadow-sm space-y-8 relative overflow-hidden">
                  {/* Active Header */}
                  <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/60 pb-6">
                    <div className="flex items-center gap-2.5">
                      <Activity className="h-4 w-4 text-emerald-500 animate-pulse" />
                      <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                        Consultation Session
                      </span>
                    </div>

                    <div className="flex items-center gap-2">
                      <span className="relative flex h-2 w-2 shrink-0">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                      </span>
                      <span className="text-xs font-medium text-slate-600 dark:text-slate-300">
                        In Progress
                      </span>
                    </div>
                  </div>

                  {/* Patient Info */}
                  <div className="text-center py-2">
                    <span className="text-[11px] font-medium text-slate-400 uppercase tracking-widest block mb-2">
                      Current Patient
                    </span>
                    <div className="text-5xl sm:text-6xl font-medium tracking-tight text-slate-900 dark:text-white my-2">
                      {activeSession.patient_name}
                    </div>
                    <p className="text-sm text-slate-500 mt-3 flex items-center justify-center gap-2">
                      <span className="bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded text-xs font-mono text-slate-600 dark:text-slate-300">
                        Ticket {activeSession.ticket?.queue_number || `#${activeSession.ticket_id.substring(0, 4)}`}
                      </span>
                    </p>
                  </div>

                  {/* Stopwatch Timer Grid */}
                  <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl overflow-hidden border border-slate-100 dark:border-slate-800 p-px">
                    <div className="bg-white dark:bg-slate-950 p-6 sm:p-8 text-center">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-slate-400 font-medium">Elapsed Time</span>
                        <span className={`text-[10px] font-semibold px-2 py-0.5 rounded ${isOvertime ? "bg-amber-100 text-amber-800 dark:bg-amber-500/20 dark:text-amber-300" : "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400"}`}>
                          {isOvertime ? `Overtime: +${formatSecondsToTimer(overtimeSec)}` : `${formatSecondsToTimer(remainingSec)} remaining`}
                        </span>
                      </div>
                      
                      {/* Digits */}
                      <div className="py-4">
                        <div className={`text-6xl sm:text-7xl font-medium font-mono tracking-tight ${isOvertime ? "text-amber-500" : "text-slate-900 dark:text-white"}`}>
                          {formatSecondsToTimer(elapsedSeconds)}
                        </div>
                      </div>

                      {/* Progress Bar */}
                      <div className="space-y-2 mt-4">
                        <div className="w-full bg-slate-100 dark:bg-slate-800/80 rounded-full h-1.5 overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all duration-1000 ease-linear ${
                              isOvertime ? "bg-amber-500" : "bg-emerald-500"
                            }`}
                            style={{ width: `${progress}%` }}
                          />
                        </div>
                        <div className="flex justify-between text-[10px] text-slate-400 font-medium px-1">
                          <span>00:00 Started</span>
                          <span>Target: {targetAvgPace} min</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Complete Button */}
                  <Button
                    onClick={() => finishMutation.mutate()}
                    disabled={finishMutation.isPending}
                    size="lg"
                    className="w-full h-14 rounded-xl font-semibold bg-slate-900 hover:bg-slate-800 text-white dark:bg-white dark:text-slate-900 shadow-sm transition-transform active:scale-[0.99]"
                  >
                    {finishMutation.isPending ? (
                      <>
                        <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                        Saving Record...
                      </>
                    ) : (
                      <>
                        <CheckCircle2 className="mr-2 h-5 w-5" />
                        Complete Consultation & Free Room
                      </>
                    )}
                  </Button>
                </div>
              );
            })()
          ) : (
            /* Idle Examination Room Card */
            <Card className="p-8 sm:p-10 space-y-6 rounded-3xl border-slate-200/80 dark:border-slate-800 shadow-sm text-center">
              <div className="h-14 w-14 rounded-2xl bg-emerald-50 dark:bg-emerald-950/60 flex items-center justify-center mx-auto text-emerald-700 dark:text-emerald-400">
                <Stethoscope className="h-7 w-7" />
              </div>

              <div className="space-y-1.5">
                <h3 className="text-xl font-bold tracking-tight text-slate-900 dark:text-white">
                  {isOffline
                    ? "Doctor is Currently Offline"
                    : isQueueEmpty
                    ? "No Patients Waiting in Queue"
                    : "Examination Room is Ready"}
                </h3>

                <p className="text-xs text-slate-500 max-w-md mx-auto leading-relaxed">
                  {isOffline
                    ? "Turn on your shift switch above to begin admitting waiting patients into your room."
                    : isQueueEmpty
                    ? "The waiting lobby is clear. The button will activate automatically as soon as a new patient joins."
                    : `There ${waitingCount === 1 ? "is" : "are"} ${waitingCount} patient${
                        waitingCount === 1 ? "" : "s"
                      } waiting in the lobby.`}
                </p>
              </div>

              {/* Next Patient Preview Card */}
              {!isOffline && nextPatient && (
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/70 dark:border-slate-700 flex items-center justify-between text-left max-w-md mx-auto">
                  <div className="flex items-center gap-3">
                    <div className="h-9 w-9 rounded-xl bg-slate-200 dark:bg-slate-700 text-slate-800 dark:text-slate-200 flex items-center justify-center font-mono font-bold text-xs shrink-0">
                      #1
                    </div>
                    <div>
                      <div className="text-xs font-bold text-slate-900 dark:text-white">
                        {nextPatient.patient_name}
                      </div>
                      <span className="text-[10px] text-slate-400">
                        {nextPatient.estimated_wait_minutes !== null && nextPatient.estimated_wait_minutes !== undefined
                          ? `Estimated wait: ~${nextPatient.estimated_wait_minutes} min`
                          : "Ready to be called"}
                      </span>
                    </div>
                  </div>

                  <div className="font-mono text-xs font-bold text-emerald-700 dark:text-emerald-400">
                    {nextPatient.queue_number}
                  </div>
                </div>
              )}

              {/* Primary Call Next Button */}
              <div className="pt-2 max-w-md mx-auto">
                <Button
                  onClick={() => callNextMutation.mutate()}
                  disabled={callNextMutation.isPending || isOffline || isQueueEmpty}
                  className={`w-full h-12 rounded-full font-bold text-xs shadow-md cursor-pointer transition-all active:scale-[0.99] ${
                    isOffline || isQueueEmpty
                      ? "bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500 cursor-not-allowed shadow-none border border-slate-200/60 dark:border-slate-700"
                      : "bg-slate-900 hover:bg-slate-800 text-white"
                  }`}
                >
                  {callNextMutation.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Admitting Patient...
                    </>
                  ) : isOffline ? (
                    <>
                      <PhoneCall className="mr-2 h-4 w-4 opacity-50" />
                      Shift is Offline
                    </>
                  ) : isQueueEmpty ? (
                    <>
                      <CheckCircle2 className="mr-2 h-4 w-4 text-emerald-600" />
                      Queue is Empty (No Patients)
                    </>
                  ) : (
                    <>
                      <PhoneCall className="mr-2 h-4 w-4" />
                      <span>
                        Call Next Patient: {nextPatient ? `${nextPatient.patient_name} (${nextPatient.queue_number})` : `${waitingCount} waiting`}
                      </span>
                    </>
                  )}
                </Button>
              </div>
            </Card>
          )}
        </div>

        {/* Right Column: Waiting Patients List (5 Cols) */}
        <div className="lg:col-span-5 space-y-6">
          <Card className="rounded-3xl border-slate-200/80 dark:border-slate-800 overflow-hidden shadow-sm">
            <CardHeader className="py-4 px-5 flex flex-row items-center justify-between border-b border-slate-100 dark:border-slate-800">
              <CardTitle className="text-sm font-bold text-slate-900 dark:text-white">
                Waiting Patients
              </CardTitle>
              <span className="text-xs text-slate-400 font-medium">
                {waitingCount} in lobby
              </span>
            </CardHeader>

            <CardContent className="p-0">
              <div className="divide-y divide-slate-100 dark:divide-slate-800">
                {waitingList.length === 0 ? (
                  <div className="py-14 text-center px-4 space-y-2">
                    <CheckCircle2 className="h-8 w-8 text-slate-300 mx-auto" />
                    <div className="text-xs font-bold text-slate-700 dark:text-slate-300">
                      Lobby is Clear
                    </div>
                    <p className="text-[11px] text-slate-400">
                      No patients currently waiting in line.
                    </p>
                  </div>
                ) : (
                  waitingList.map((t, idx) => (
                    <div
                      key={t.queue_number || idx}
                      className="p-4 flex items-center justify-between hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <div className="h-8 w-8 rounded-xl bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 flex items-center justify-center font-mono font-bold text-xs">
                          #{idx + 1}
                        </div>
                        <div>
                          <div className="font-mono font-bold text-xs text-emerald-700 dark:text-emerald-400">
                            {t.queue_number}
                          </div>
                          <div className="text-xs font-semibold text-slate-800 dark:text-slate-200">
                            {t.patient_name}
                          </div>
                        </div>
                      </div>

                      <div className="text-right">
                        <span className="text-xs font-mono font-bold text-slate-700 dark:text-slate-300">
                          {t.estimated_wait_minutes !== null && t.estimated_wait_minutes !== undefined
                            ? `~${t.estimated_wait_minutes}m`
                            : "-"}
                        </span>
                        <span className="text-[10px] text-slate-400 block font-mono">
                          {idx === 0 ? "Next in line" : "Waiting"}
                        </span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
