"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useSSE } from "@/hooks/use-sse";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Users,
  Clock,
  Activity,
  BarChart3,
  Settings,
  ShieldCheck,
  ArrowRight,
  TrendingUp,
  MoreHorizontal,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { DoctorPerformance, AdminDashboardStats } from "@/lib/types";

export default function AdminAnalyticsPage() {
  const { user, token, isLoading: isAuthLoading, isMounted, switchPersona } = useAuth();
  const { isConnected } = useSSE();
  const queryClient = useQueryClient();

  const [selectedDoctor, setSelectedDoctor] = useState<DoctorPerformance | null>(null);
  const [targetMinutesInput, setTargetMinutesInput] = useState<string>("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // Query: Admin Dashboard Stats (SSE-aware adaptive polling)
  const { data: stats, isLoading: isStatsLoading } = useQuery<AdminDashboardStats>({
    queryKey: ["admin-stats"],
    queryFn: () => api.getAdminStats(),
    enabled: !!token && user?.role === "admin",
    refetchInterval: isConnected ? 30000 : 3000,
    initialData: () => {
      if (typeof window === "undefined") return undefined;
      try {
        const saved = localStorage.getItem("clinic_admin_stats");
        return saved ? JSON.parse(saved) : undefined;
      } catch {
        return undefined;
      }
    },
  });

  // Persist admin stats to localStorage
  React.useEffect(() => {
    if (stats) {
      localStorage.setItem("clinic_admin_stats", JSON.stringify(stats));
    }
  }, [stats]);

  // Mutation: Update Doctor Target Consultation Speed
  const updateConfigMutation = useMutation({
    mutationFn: ({ doctorId, minutes }: { doctorId: string; minutes: number }) =>
      api.updateDoctorConfig(doctorId, minutes),
    onSuccess: (data) => {
      toast.success("Doctor Configuration Saved", {
        description: `Target consultation time updated to ${data.avg_consultation_time} minutes.`,
      });
      setIsDialogOpen(false);
      queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
      queryClient.invalidateQueries({ queryKey: ["queue-status"] });
    },
    onError: (err: any) => {
      toast.error("Failed to update config", { description: err.message });
    },
  });

  const handleOpenEdit = (doc: DoctorPerformance) => {
    setSelectedDoctor(doc);
    setTargetMinutesInput(doc.target_avg_minutes.toString());
    setIsDialogOpen(true);
  };

  const handleSaveConfig = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDoctor) return;
    const mins = parseInt(targetMinutesInput, 10);
    if (isNaN(mins) || mins <= 0) {
      toast.error("Invalid duration", { description: "Target consultation minutes must be greater than 0." });
      return;
    }
    updateConfigMutation.mutate({ doctorId: selectedDoctor.doctor_id, minutes: mins });
  };

  if (!isMounted) {
    return null;
  }

  if (!token || user?.role !== "admin") {
    return (
      <div className="mx-auto max-w-md px-4 py-16 text-center">
        <Card className="p-8 space-y-4">
          <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
            <BarChart3 className="h-6 w-6" />
          </div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">Executive Analytics Access</h2>
          <p className="text-xs text-slate-500">
            You must be authenticated with an Administrator account to view clinic analytics.
          </p>
          <div className="pt-2 flex flex-col gap-2">
            <Button
              onClick={async () => {
                await switchPersona("admin");
                window.location.href = "/admin";
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

  const summary = stats?.summary;
  const doctorList: DoctorPerformance[] = stats?.doctor_performance || [];

  // Compute doctor-specific aggregated performance metrics
  const activePractitioners = summary?.online_doctors_count ?? doctorList.filter((d) => d.is_online).length;
  const totalConsultations = summary?.total_served_today ?? doctorList.reduce((acc, d) => acc + d.total_consultations_today, 0);
  
  const totalServedByDocs = doctorList.reduce((acc, d) => acc + d.total_consultations_today, 0);
  const weightedDurationSum = doctorList.reduce((acc, d) => acc + d.avg_actual_consultation_minutes * Math.max(d.total_consultations_today, 1), 0);
  const avgConsultationPace = doctorList.length > 0
    ? (weightedDurationSum / (totalServedByDocs || doctorList.length)).toFixed(1)
    : "0.0";

  const avgUtilization = doctorList.length > 0
    ? (doctorList.reduce((acc, d) => acc + d.utilization_rate_percentage, 0) / doctorList.length).toFixed(1)
    : "0.0";

  return (
    <div className="space-y-8 pb-10">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
            Doctor Management & Performance
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Workforce productivity, consultation duration forensics, and deterministic greedy queue parameters.
          </p>
        </div>

        <Button asChild variant="outline" className="rounded-full border-slate-200 text-xs font-bold h-10 px-4">
          <Link href="/admin/audit" className="flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-emerald-700" />
            <span>View Activity Audit Trail</span>
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </Button>
      </div>

      {/* 4 Doctor-Specific KPI Cards Grid */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {/* KPI 1: Active Practitioners */}
        <Card className="flex flex-col justify-between rounded-[22px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="flex items-center gap-2">
              <div className="h-8 w-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 flex items-center justify-center">
                <Users className="h-4 w-4" />
              </div>
              <CardTitle className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                Active Practitioners
              </CardTitle>
            </div>
            <MoreHorizontal className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">
              {isStatsLoading ? "..." : activePractitioners}
            </div>
            <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/60 px-2.5 py-0.5 rounded-full">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
              {activePractitioners} on duty in rooms
            </span>
          </CardContent>
        </Card>

        {/* KPI 2: Total Consultations */}
        <Card className="flex flex-col justify-between rounded-[22px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="flex items-center gap-2">
              <div className="h-8 w-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 flex items-center justify-center">
                <Activity className="h-4 w-4" />
              </div>
              <CardTitle className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                Total Consultations
              </CardTitle>
            </div>
            <MoreHorizontal className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">
              {isStatsLoading ? "..." : totalConsultations}
            </div>
            <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/60 px-2.5 py-0.5 rounded-full">
              <TrendingUp className="h-3 w-3" />
              Completed across all rooms
            </span>
          </CardContent>
        </Card>

        {/* KPI 3: Avg Consultation Pace */}
        <Card className="flex flex-col justify-between rounded-[22px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="flex items-center gap-2">
              <div className="h-8 w-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 flex items-center justify-center">
                <Clock className="h-4 w-4" />
              </div>
              <CardTitle className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                Avg Consultation Pace
              </CardTitle>
            </div>
            <MoreHorizontal className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="text-3xl font-black text-emerald-700 dark:text-emerald-400 tracking-tight">
              {isStatsLoading ? "..." : `${avgConsultationPace}m`}
            </div>
            <span className="text-[11px] text-slate-400">
              Empirical examination duration
            </span>
          </CardContent>
        </Card>

        {/* KPI 4: Staff Utilization Rate */}
        <Card className="flex flex-col justify-between rounded-[22px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="flex items-center gap-2">
              <div className="h-8 w-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 flex items-center justify-center">
                <BarChart3 className="h-4 w-4" />
              </div>
              <CardTitle className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                Staff Utilization Rate
              </CardTitle>
            </div>
            <MoreHorizontal className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="text-3xl font-black text-slate-900 dark:text-white tracking-tight">
              {isStatsLoading ? "..." : `${avgUtilization}%`}
            </div>
            <span className="text-[11px] text-slate-400">
              Active consultation vs shift time
            </span>
          </CardContent>
        </Card>
      </div>

      {/* Doctor Productivity Table Card */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-3">
          <div>
            <CardTitle className="text-base font-bold text-slate-900 dark:text-white">
              Doctor Performance & Greedy Parameters
            </CardTitle>
            <p className="text-xs text-slate-400">
              Configure baseline consultation durations used by the deterministic greedy queue estimation algorithm.
            </p>
          </div>
        </CardHeader>

        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-slate-100 dark:border-slate-800 text-slate-400 text-[11px] uppercase tracking-wider font-semibold">
                <tr>
                  <th className="py-3.5 px-6">Doctor</th>
                  <th className="py-3.5 px-6">Shift Status</th>
                  <th className="py-3.5 px-6 text-center">Served Today</th>
                  <th className="py-3.5 px-6 text-center">Actual Avg</th>
                  <th className="py-3.5 px-6 text-center">Target Speed</th>
                  <th className="py-3.5 px-6 text-center">Utilization</th>
                  <th className="py-3.5 px-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {doctorList.map((doc) => (
                  <tr key={doc.doctor_id} className="hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors">
                    <td className="py-4 px-6 font-bold text-slate-900 dark:text-white">
                      <div className="flex items-center gap-3">
                        <div className="h-9 w-9 rounded-2xl bg-emerald-100 dark:bg-emerald-900/60 text-emerald-800 dark:text-emerald-300 flex items-center justify-center font-bold text-xs">
                          DR
                        </div>
                        <div>
                          <div>{doc.doctor_name}</div>
                          <span
                            className="text-[10px] text-slate-400 font-medium tracking-tight cursor-default"
                            title={`Doctor UUID: ${doc.doctor_id}`}
                          >
                            @{doc.username || doc.doctor_name.toLowerCase().replace(/[^a-z0-9]/g, "_")}
                          </span>
                        </div>
                      </div>
                    </td>
                    <td className="py-4 px-6">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold ${
                          doc.is_online
                            ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60"
                            : "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300"
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${
                            doc.is_online ? "bg-emerald-500" : "bg-slate-400"
                          }`}
                        />
                        {doc.is_online ? "ONLINE" : "OFFLINE"}
                      </span>
                    </td>
                    <td className="py-4 px-6 text-center font-semibold text-slate-900 dark:text-white">
                      {doc.total_consultations_today}
                    </td>
                    <td className="py-4 px-6 text-center font-mono text-slate-600 dark:text-slate-300">
                      {doc.avg_actual_consultation_minutes}m
                    </td>
                    <td className="py-4 px-6 text-center">
                      <span className="font-mono font-bold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/60 px-3 py-1 rounded-full">
                        {doc.target_avg_minutes} min
                      </span>
                    </td>
                    <td className="py-4 px-6 text-center font-semibold text-slate-900 dark:text-white">
                      {doc.utilization_rate_percentage}%
                    </td>
                    <td className="py-4 px-6 text-right">
                      <Button
                        onClick={() => handleOpenEdit(doc)}
                        variant="outline"
                        size="sm"
                        className="h-8 rounded-full text-xs font-bold px-3"
                      >
                        <Settings className="mr-1 h-3 w-3" />
                        Configure
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Edit Config Modal */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-md rounded-[26px] p-7">
          <DialogHeader>
            <DialogTitle>Configure Target Consultation Speed</DialogTitle>
            <DialogDescription className="text-xs">
              Adjust baseline consultation duration for <strong>{selectedDoctor?.doctor_name}</strong>.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSaveConfig} className="space-y-4 py-2">
            <div className="space-y-2">
              <label className="text-xs font-bold text-slate-700 dark:text-slate-300">
                Target Consultation Minutes
              </label>
              <Input
                type="number"
                min="1"
                max="60"
                value={targetMinutesInput}
                onChange={(e) => setTargetMinutesInput(e.target.value)}
                disabled={updateConfigMutation.isPending}
                className="h-11 rounded-2xl text-xs"
              />
              <p className="text-[11px] text-slate-400">
                Current value: {selectedDoctor?.target_avg_minutes} minutes.
              </p>
            </div>

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsDialogOpen(false)}
                disabled={updateConfigMutation.isPending}
                className="rounded-full"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={updateConfigMutation.isPending}
                className="rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold"
              >
                {updateConfigMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  "Save Parameter"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
