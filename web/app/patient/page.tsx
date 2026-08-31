"use client";

import React, { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useSSE } from "@/hooks/use-sse";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Ticket,
  Clock,
  CheckCircle2,
  Users,
  ArrowRight,
  Loader2,
  AlertCircle,
  MoreHorizontal,
  Stethoscope,
  ShieldCheck,
} from "lucide-react";
import { toast } from "sonner";
import { formatDateTime } from "@/lib/utils";

export default function PatientPortalPage() {
  const { user, token, switchPersona, isMounted } = useAuth();
  const { isConnected } = useSSE();
  const queryClient = useQueryClient();
  const [useAccountName, setUseAccountName] = useState<boolean>(true);
  const [customNameInput, setCustomNameInput] = useState<string>("");

  // Query: General Public Queue Status (SSE-aware adaptive polling)
  const { data: queueStatus } = useQuery({
    queryKey: ["queue-status"],
    queryFn: () => api.getQueueStatus(),
    refetchInterval: isConnected ? 30000 : 3000,
  });

  // Query: User Active Ticket (SSE-aware adaptive polling)
  const { data: ticketData, isLoading: isTicketLoading } = useQuery({
    queryKey: ["my-ticket", user?.id],
    queryFn: () => api.getMyTicket(),
    enabled: !!token,
    retry: false,
    refetchInterval: isConnected ? 30000 : 3000,
    initialData: () => {
      if (typeof window === "undefined") return undefined;
      try {
        const saved = localStorage.getItem("clinic_queue_ticket");
        return saved ? { ticket: JSON.parse(saved) } : undefined;
      } catch {
        return undefined;
      }
    },
  });

  const activeTicket = ticketData?.ticket;

  const callingDoctor = queueStatus?.online_doctors?.find(
    (d) =>
      d.status === "IN_CONSULTATION" &&
      d.current_patient &&
      activeTicket?.patient_name &&
      d.current_patient.trim().toLowerCase() === activeTicket.patient_name.trim().toLowerCase()
  );
  const callingDoctorIdx = queueStatus?.online_doctors?.findIndex((d) => d.id === callingDoctor?.id);
  const assignedRoomName =
    callingDoctorIdx !== undefined && callingDoctorIdx !== -1
      ? `Room ${callingDoctorIdx + 1}`
      : "Room 1";
  const assignedDoctorName = callingDoctor?.name || (activeTicket as any)?.doctor_name || "Dr. Sarah Adams";

  // Persist active ticket into localStorage
  useEffect(() => {
    if (ticketData?.ticket) {
      localStorage.setItem("clinic_queue_ticket", JSON.stringify(ticketData.ticket));
    } else if (ticketData && !ticketData.ticket) {
      localStorage.removeItem("clinic_queue_ticket");
    }
  }, [ticketData]);

  // Live timer for active consultation
  const [activeSeconds, setActiveSeconds] = useState(0);
  useEffect(() => {
    if (activeTicket?.status !== "IN_CONSULTATION") {
      setActiveSeconds(0);
      return;
    }
    const calledTime = activeTicket.called_at 
      ? new Date(activeTicket.called_at).getTime() 
      : Date.now() - (callingDoctor?.elapsed_minutes || 0) * 60000;
    
    const updateTimer = () => {
      const diff = Math.floor((Date.now() - calledTime) / 1000);
      setActiveSeconds(Math.max(0, diff));
    };
    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [activeTicket?.status, activeTicket?.called_at, callingDoctor?.elapsed_minutes]);

  const formatTime = (secs: number) => {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
  };

  // Mutation: Join Queue
  const joinMutation = useMutation({
    mutationFn: (name: string) => api.joinQueue(name),
    onSuccess: (data) => {
      toast.success("Queue Ticket Generated!", {
        description: `Your ticket number is ${data.ticket.queue_number}`,
      });
      setCustomNameInput("");
      localStorage.setItem("clinic_queue_ticket", JSON.stringify(data.ticket));
      queryClient.setQueryData(["my-ticket", user?.id], { ticket: data.ticket });
      queryClient.invalidateQueries({ queryKey: ["my-ticket"] });
      queryClient.invalidateQueries({ queryKey: ["queue-status"] });
    },
    onError: (err: any) => {
      toast.error("Failed to Join Queue", {
        description: err.message || "Please check patient name and try again.",
      });
    },
  });

  const effectivePatientName = user && useAccountName ? user.name : customNameInput.trim();

  const handleJoinQueue = (e: React.FormEvent) => {
    e.preventDefault();
    const name = effectivePatientName;
    if (!name) {
      toast.error("Please enter a patient name");
      return;
    }
    joinMutation.mutate(name);
  };

  const onlineDocs =
    queueStatus?.online_doctors_count ??
    (queueStatus?.online_doctors?.filter((d) => d.is_online).length ?? 0);
  const queueList = queueStatus?.queue_list || [];
  const totalWaiting = queueStatus?.total_waiting ?? queueList.length;

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

  return (
    <div className="space-y-8 pb-10">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
            Patient Online Queue & Live Countdown
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Take a walk-in queue ticket online and track minute-accurate wait times calculated in real-time.
          </p>
        </div>

        <div className="inline-flex items-center gap-2 rounded-full bg-white dark:bg-slate-900 border border-slate-200/90 dark:border-slate-800 px-3.5 py-1.5 shadow-2xs shrink-0 whitespace-nowrap">
          <span className="relative flex h-2 w-2 shrink-0">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">
            {onlineDocs} Doctors Online
          </span>
        </div>
      </div>

      {/* Offline Alert Notice */}
      {queueStatus?.notice && (
        <div className="rounded-2xl border border-amber-500/30 bg-amber-50 dark:bg-amber-950/20 p-4 text-amber-900 dark:text-amber-300 flex items-center gap-3">
          <AlertCircle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
          <div>
            <h4 className="text-xs font-bold uppercase tracking-wider">Clinic Operational Notice</h4>
            <p className="text-xs mt-0.5">{queueStatus.notice}</p>
          </div>
        </div>
      )}

      <div className="max-w-2xl mx-auto space-y-6">
        {!isMounted ? null : activeTicket ? (
          <div className="rounded-2xl bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 p-6 sm:p-8 shadow-sm space-y-8 relative overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/60 pb-6">
              <div className="flex items-center gap-2.5">
                <Ticket className="h-4 w-4 text-slate-400" />
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  Clinic Ticket
                </span>
              </div>
              <div className="flex items-center gap-2">
                {activeTicket.status === "IN_CONSULTATION" ? (
                  <span className="relative flex h-2 w-2 shrink-0">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                  </span>
                ) : (
                  <span className="relative flex h-2 w-2 shrink-0">
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                  </span>
                )}
                <span className="text-xs font-medium text-slate-600 dark:text-slate-300">
                  {activeTicket.status === "IN_CONSULTATION" ? "In Session" : "Waiting in Lobby"}
                </span>
              </div>
            </div>

            {/* Big Queue Display */}
            <div className="text-center py-2">
              <span className="text-[11px] font-medium text-slate-500 uppercase tracking-widest block mb-2">
                Queue Number
              </span>
              <div className="text-6xl sm:text-7xl md:text-8xl font-medium tracking-tight text-slate-900 dark:text-white my-2 font-mono tabular-nums">
                {activeTicket.queue_number}
              </div>
              <p className="text-sm text-slate-500 mt-3 truncate px-2">
                Patient: <span className="font-medium text-slate-900 dark:text-slate-100">{activeTicket.patient_name}</span>
              </p>
            </div>

            {/* Position & Estimated Wait Time Stats */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 border-t border-slate-100 dark:border-slate-800/60 pt-6">
              <div className="text-center sm:border-r border-b sm:border-b-0 border-slate-100 dark:border-slate-800/60 pb-5 sm:pb-0 mb-2 sm:mb-0 sm:pr-4">
                <span className="text-xs text-slate-500 block font-medium mb-1.5">
                  {activeTicket.status === "IN_CONSULTATION" ? "Assigned Doctor" : "Queue Position"}
                </span>
                <div className="text-xl sm:text-2xl font-semibold text-slate-900 dark:text-white my-1 truncate">
                  {activeTicket.status === "IN_CONSULTATION"
                    ? assignedDoctorName
                    : `#${activeTicket.position_in_queue ?? activeTicket.position ?? 1}`}
                </div>
                <span className="text-xs text-slate-500 block truncate">
                  {activeTicket.status === "IN_CONSULTATION"
                    ? `${assignedRoomName}`
                    : (activeTicket.position_in_queue ?? activeTicket.position ?? 1) > 1
                    ? `${(activeTicket.position_in_queue ?? activeTicket.position ?? 1) - 1} patient${(activeTicket.position_in_queue ?? activeTicket.position ?? 1) - 1 === 1 ? "" : "s"} ahead`
                    : "Next in line!"}
                </span>
              </div>

              <div className="text-center sm:pl-4">
                <span className="text-xs text-slate-500 block font-medium mb-1.5">
                  {activeTicket.status === "IN_CONSULTATION" ? "Consultation Time" : "Estimated Wait"}
                </span>
                <div className="text-xl sm:text-2xl font-semibold text-slate-900 dark:text-white my-1 font-mono tabular-nums">
                  {activeTicket.status === "IN_CONSULTATION"
                    ? formatTime(activeSeconds)
                    : onlineDocs === 0 || activeTicket.estimated_wait_time_minutes === null || activeTicket.estimated_wait_time_minutes === undefined
                    ? "-"
                    : `~${activeTicket.estimated_wait_time_minutes}m`}
                </div>
                <span className="text-xs text-slate-500 block">
                  {activeTicket.status === "IN_CONSULTATION"
                    ? `Target: ${callingDoctor?.avg_time || 3}m`
                    : onlineDocs === 0
                    ? "Doctor offline"
                    : "Live Countdown"}
                </span>
              </div>
            </div>

            {/* Bottom Instructions */}
            {activeTicket.status === "IN_CONSULTATION" ? (
              <div className="rounded-2xl bg-emerald-50 dark:bg-emerald-500/10 p-5 border border-emerald-100 dark:border-emerald-500/20 flex flex-col sm:flex-row sm:items-center gap-4 animate-in fade-in slide-in-from-bottom-2 duration-500">
                <div className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
                  <div className="absolute inset-0 rounded-full border border-emerald-500 animate-ping opacity-30"></div>
                  <ArrowRight className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-emerald-900 dark:text-emerald-300">
                    Your Ticket Is Called
                  </h4>
                  <p className="text-xs text-emerald-700 dark:text-emerald-400/80 mt-0.5 leading-relaxed">
                    Please proceed to <strong className="font-semibold text-emerald-900 dark:text-emerald-200">{assignedRoomName}</strong> for your consultation with <strong className="font-semibold text-emerald-900 dark:text-emerald-200">{assignedDoctorName}</strong>.
                  </p>
                </div>
              </div>
            ) : (
              <div className="text-xs text-slate-400 text-center pt-2">
                Countdown updates automatically in real-time.
              </div>
            )}
          </div>
        ) : isMounted && user?.role === "doctor" ? (
          /* Doctor View: Quick link to Examination Room */
          <Card className="p-8 text-center space-y-4">
            <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
              <Stethoscope className="h-6 w-6" />
            </div>
            <CardTitle className="text-base font-bold text-slate-900 dark:text-white">
              Doctor Examination Room
            </CardTitle>
            <p className="text-xs text-slate-400 max-w-sm mx-auto">
              You are viewing the clinic waiting lobby as a Practitioner. Open your examination room to call the next waiting patient.
            </p>
            <Button asChild className="w-full rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11">
              <Link href="/doctor">
                <span>Open Doctor Examination Room</span>
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
          </Card>
        ) : isMounted && user?.role === "admin" ? (
          /* Admin View: Quick link to Doctor Management */
          <Card className="p-8 text-center space-y-4">
            <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
              <ShieldCheck className="h-6 w-6" />
            </div>
            <CardTitle className="text-base font-bold text-slate-900 dark:text-white">
              Executive Queue Oversight
            </CardTitle>
            <p className="text-xs text-slate-400 max-w-sm mx-auto">
              Administrators monitor clinic queue throughput and configure doctor target speeds from the Doctor Management panel.
            </p>
            <Button asChild className="w-full rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11">
              <Link href="/admin">
                <span>Go to Doctor Management Panel</span>
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
          </Card>
        ) : (
          /* Join Queue Card for Patient */
          <Card className="overflow-hidden border-slate-200/80 dark:border-slate-800 shadow-sm">
            <CardHeader className="pb-5">
              <div className="flex flex-row items-center justify-between gap-4">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="h-11 w-11 rounded-2xl bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 flex items-center justify-center shrink-0">
                    <Ticket className="h-5 w-5" />
                  </div>
                  <div className="min-w-0">
                    <CardTitle className="text-base font-bold text-slate-900 dark:text-white">
                      Take Clinic Queue Ticket
                    </CardTitle>
                    <p className="text-xs text-slate-400 mt-0.5">
                      {totalWaiting === 0
                        ? "Lobby is clear • Immediate room admission"
                        : `${totalWaiting} patient${totalWaiting === 1 ? "" : "s"} waiting in lobby`}
                    </p>
                  </div>
                </div>

                {/* Top-Right Modern Crisp Live ETA Tag */}
                <div className="flex items-center gap-2 rounded-xl bg-slate-50 dark:bg-slate-800/80 border border-slate-200/80 dark:border-slate-700/60 px-3 py-1.5 shrink-0 whitespace-nowrap">
                  <Clock className="h-3.5 w-3.5 text-slate-400 dark:text-slate-500 shrink-0" />
                  <span className="text-[11px] font-medium text-slate-500 dark:text-slate-400">Est. Wait:</span>
                  <span className="text-xs font-bold text-slate-900 dark:text-white font-mono">
                    {nextArrivalWaitTime === null
                      ? "-"
                      : nextArrivalWaitTime === 0
                      ? "0 min"
                      : `~${nextArrivalWaitTime} min`}
                  </span>
                </div>
              </div>
            </CardHeader>

            <form onSubmit={handleJoinQueue}>
              <CardContent className="space-y-4 pt-0">
                {isMounted && user ? (
                  <div className="space-y-4">
                    {/* Clear Segmented Control: Who is this ticket for? */}
                    <div className="space-y-2">
                      <label className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                        Who is this ticket for?
                      </label>
                      <div className="grid grid-cols-2 gap-2 p-1 rounded-2xl bg-slate-100 dark:bg-slate-800 text-xs font-semibold">
                        <button
                          type="button"
                          onClick={() => setUseAccountName(true)}
                          className={`py-2 px-3 rounded-xl transition-all flex items-center justify-center gap-2 cursor-pointer ${
                            useAccountName
                              ? "bg-white dark:bg-slate-900 text-emerald-800 dark:text-emerald-300 font-bold shadow-xs"
                              : "text-slate-500 hover:text-slate-800 dark:hover:text-slate-200"
                          }`}
                        >
                          <span className={`h-2 w-2 rounded-full ${useAccountName ? "bg-emerald-500" : "bg-transparent"}`} />
                          <span>Myself ({user.name})</span>
                        </button>

                        <button
                          type="button"
                          onClick={() => setUseAccountName(false)}
                          className={`py-2 px-3 rounded-xl transition-all flex items-center justify-center gap-2 cursor-pointer ${
                            !useAccountName
                              ? "bg-white dark:bg-slate-900 text-emerald-800 dark:text-emerald-300 font-bold shadow-xs"
                              : "text-slate-500 hover:text-slate-800 dark:hover:text-slate-200"
                          }`}
                        >
                          <span className={`h-2 w-2 rounded-full ${!useAccountName ? "bg-emerald-500" : "bg-transparent"}`} />
                          <span>Someone Else / Family</span>
                        </button>
                      </div>
                    </div>

                    {useAccountName ? (
                      /* Selected: Myself Card */
                      <div className="p-4 rounded-2xl bg-emerald-50/70 dark:bg-emerald-950/30 border border-emerald-200/60 dark:border-emerald-800/40 flex items-center justify-between animate-in fade-in duration-200">
                        <div className="flex items-center gap-3">
                          <div className="h-10 w-10 rounded-2xl bg-emerald-600 text-white flex items-center justify-center font-bold text-xs shadow-xs">
                            {user.name.charAt(0)}
                          </div>
                          <div>
                            <div className="text-xs font-bold text-slate-900 dark:text-white">
                              {user.name}
                            </div>
                            <div className="text-[11px] text-emerald-700 dark:text-emerald-400 font-medium">
                              Logged in as @{user.username} &bull; Verified Patient
                            </div>
                          </div>
                        </div>
                        <span className="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-700 bg-white dark:bg-slate-900 dark:text-emerald-300 px-3 py-1 rounded-full border border-emerald-200 dark:border-emerald-800">
                          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600" />
                          Primary Account
                        </span>
                      </div>
                    ) : (
                      /* Selected: Someone Else Input */
                      <div className="space-y-2 animate-in fade-in duration-200">
                        <label className="text-xs font-bold text-slate-700 dark:text-slate-300">
                          Patient Full Name
                        </label>
                        <Input
                          placeholder="e.g. Mary Smith (Child, Parent, or Family Member)"
                          value={customNameInput}
                          onChange={(e) => setCustomNameInput(e.target.value)}
                          disabled={joinMutation.isPending}
                          className="h-12 rounded-2xl text-xs border-slate-200"
                          autoFocus
                        />
                        <p className="text-[11px] text-slate-400">
                          This ticket will be booked and managed under your account (@{user.username}).
                        </p>
                      </div>
                    )}
                  </div>
                ) : (
                  /* Guest Mode: Name Input + 1-Click Sign-in Options */
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <label className="text-xs font-bold text-slate-700 dark:text-slate-300">
                        Patient Full Name
                      </label>
                      <Input
                        placeholder="e.g. John Doe"
                        value={customNameInput}
                        onChange={(e) => setCustomNameInput(e.target.value)}
                        disabled={joinMutation.isPending}
                        className="h-12 rounded-2xl text-xs border-slate-200"
                      />
                      <p className="text-[11px] text-slate-400">
                        Walk-in Guest mode. Name will appear on public waiting screens.
                      </p>
                    </div>

                    <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2.5">
                      <span className="text-[11px] text-slate-500 font-medium">
                        Have a registered account?
                      </span>
                      <div className="flex gap-1.5 w-full sm:w-auto">
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => switchPersona("patient_john")}
                          className="h-7 text-[11px] rounded-full font-bold px-2.5 flex-1 sm:flex-none border-slate-300"
                        >
                          Sign In as John
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => switchPersona("patient_lucas")}
                          className="h-7 text-[11px] rounded-full font-bold px-2.5 flex-1 sm:flex-none border-slate-300"
                        >
                          Sign In as Lucas
                        </Button>
                      </div>
                    </div>
                  </div>
                )}
              </CardContent>

              <CardFooter className="pt-2">
                <Button
                  type="submit"
                  disabled={joinMutation.isPending}
                  className="w-full h-12 rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs shadow-sm"
                >
                  {joinMutation.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Generating Queue Ticket...
                    </>
                  ) : (
                    <>
                      <span>
                        {isMounted && user && useAccountName
                          ? `Take Ticket for Myself (${user.name})`
                          : effectivePatientName
                          ? `Take Ticket for ${effectivePatientName}`
                          : "Take Queue Ticket"}
                      </span>
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </>
                  )}
                </Button>
              </CardFooter>
            </form>
          </Card>
        )}
      </div>
    </div>
  );
}
