"use client";

import React, { useState, useEffect, useRef } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import Link from "next/link";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  ShieldCheck,
  ArrowLeft,
  Loader2,
  Code2,
  Eye,
  CheckCircle2,
  Search,
  Calendar,
  ArrowUpDown,
  Activity,
  Users,
  RotateCcw,
  X,
  SlidersHorizontal,
  Copy,
  Check,
  Globe,
  Clock,
  FileJson,
} from "lucide-react";
import { formatDateTime, formatDate, formatTime } from "@/lib/utils";
import type { AuditLog } from "@/lib/types";

function getHumanIdentityHandle(log: AuditLog): { label: string; handle: string } {
  if (log.details?.username) {
    return { label: "Account Handle", handle: `@${log.details.username}` };
  }
  if (log.details?.queue_number) {
    return { label: "Ticket Reference", handle: `Ticket #${log.details.queue_number}` };
  }

  const docId = log.details?.doctor_id || log.user_id;
  if (docId === "01919df4-8e3b-7412-a1f9-90b567c9e101" || log.actor_name?.includes("Sarah Adams")) {
    return { label: "Doctor Handle", handle: "@doctor_a" };
  }
  if (docId === "01919df4-8e3b-7412-a1f9-90b567c9e102" || log.actor_name?.includes("Michael Chen") || log.actor_name?.includes("Brian Miller")) {
    return { label: "Doctor Handle", handle: "@doctor_b" };
  }

  if (log.role === "admin" || log.actor_name?.toLowerCase().includes("admin")) {
    return { label: "Admin Handle", handle: "@admin" };
  }

  if (log.actor_name?.includes("John")) {
    return { label: "Patient Handle", handle: "@patient_john" };
  }
  if (log.actor_name?.includes("Lucas")) {
    return { label: "Patient Handle", handle: "@patient_lucas" };
  }

  if (log.role === "doctor" && log.actor_name) {
    const slug = log.actor_name
      .toLowerCase()
      .trim()
      .replace(/^dr\.?\s*/i, "")
      .replace(/[^a-z0-9]+/g, "_")
      .replace(/^_+|_+$/g, "");
    return { label: "Doctor Handle", handle: `@${slug || "doctor"}` };
  }

  if (log.role === "patient") {
    if (
      log.actor_name &&
      !log.actor_name.toLowerCase().includes("walk-in") &&
      !log.actor_name.toLowerCase().includes("anonymous")
    ) {
      const slug = log.actor_name
        .toLowerCase()
        .trim()
        .replace(/[^a-z0-9]+/g, "_")
        .replace(/^_+|_+$/g, "");
      return { label: "Patient Handle", handle: `@${slug}` };
    }
    return { label: "Identity", handle: "Walk-in Guest" };
  }

  return { label: "Identity", handle: log.actor_name || "System" };
}

export default function AdminAuditTrailPage() {
  const { user, token, isLoading: isAuthLoading, isMounted, switchPersona } = useAuth();
  const [limit] = useState(15);
  const [search, setSearch] = useState<string>("");
  const [startDate, setStartDate] = useState<string>("");
  const [endDate, setEndDate] = useState<string>("");
  const [sortOrder, setSortOrder] = useState<"desc" | "asc">("desc");
  const [actionFilter, setActionFilter] = useState<string>("ALL");
  const [roleFilter, setRoleFilter] = useState<string>("ALL");

  const [inspectLog, setInspectLog] = useState<AuditLog | null>(null);
  const [copied, setCopied] = useState(false);
  const loadMoreSentinelRef = useRef<HTMLDivElement>(null);

  const handleCopyJSON = () => {
    if (!inspectLog) return;
    const payload = inspectLog.details || inspectLog.metadata || {};
    navigator.clipboard.writeText(JSON.stringify(payload, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Query: Lazy Loading Infinite Audit Logs via Cursor Pagination
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
    error,
    refetch,
  } = useInfiniteQuery({
    queryKey: [
      "admin-audit-logs-infinite",
      limit,
      search,
      startDate,
      endDate,
      sortOrder,
      actionFilter,
      roleFilter,
    ],
    queryFn: ({ pageParam }) =>
      api.getAuditLogs({
        cursor: pageParam,
        limit,
        search: search.trim() || undefined,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
        sort_order: sortOrder,
        action: actionFilter !== "ALL" ? actionFilter : undefined,
        role: roleFilter !== "ALL" ? roleFilter : undefined,
      }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.next_cursor : undefined),
    enabled: !!token && user?.role === "admin",
    refetchInterval: 3000,
  });

  // Auto-fetch next page on scroll / intersection
  useEffect(() => {
    const sentinel = loadMoreSentinelRef.current;
    if (!sentinel || !hasNextPage || isFetchingNextPage) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const handleContainerScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    if (scrollHeight - scrollTop - clientHeight < 100) {
      if (hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
      }
    }
  };

  const activeFilterCount = [
    search.trim() !== "",
    startDate !== "",
    endDate !== "",
    sortOrder !== "desc",
    actionFilter !== "ALL",
    roleFilter !== "ALL",
  ].filter(Boolean).length;

  const hasActiveFilters = activeFilterCount > 0;

  const handleResetFilters = () => {
    setSearch("");
    setStartDate("");
    setEndDate("");
    setSortOrder("desc");
    setActionFilter("ALL");
    setRoleFilter("ALL");
  };

  if (!isMounted) {
    return null;
  }

  if (!token || user?.role !== "admin") {
    return (
      <div className="mx-auto max-w-md px-4 py-16 text-center">
        <Card className="p-8 space-y-4">
          <div className="h-12 w-12 rounded-2xl bg-emerald-50 text-emerald-700 flex items-center justify-center mx-auto">
            <ShieldCheck className="h-6 w-6" />
          </div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">Audit Trail Access</h2>
          <p className="text-xs text-slate-500">
            You must be authenticated with an Administrator account to inspect activity audit logs.
          </p>
          <div className="pt-2">
            <Button
              onClick={() => switchPersona("admin")}
              className="w-full rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-11"
            >
              Sign In as Admin CEO
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  const logs = data?.pages.flatMap((page) => page.logs) || [];
  const totalRecords = data?.pages[0]?.total_records ?? logs.length;

  return (
    <div className="space-y-6 pb-6">
      {/* Top Header */}
      <div>
        <div className="flex items-center gap-2 mb-1">
          <Button asChild variant="ghost" size="sm" className="h-7 px-2 -ml-2 text-xs font-semibold text-slate-500">
            <Link href="/admin">
              <ArrowLeft className="mr-1 h-3.5 w-3.5" />
              Back to Analytics
            </Link>
          </Button>
        </div>
        <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
          Clinic Activity Logging & Audit Trail
        </h1>
        <p className="text-xs sm:text-sm text-slate-500 mt-1">
          Immutable audit pipeline capturing authentication, shifts, and doctor room allocations.
        </p>
      </div>

      {/* Main Content Layout: Sidebar Filter on Left + Wide Audit Table on Right */}
      <div className="flex flex-col lg:flex-row items-start gap-6">
        {/* Left Sidebar Filter Panel */}
        <Card className="w-full lg:w-72 xl:w-80 shrink-0 p-5 rounded-[24px] border-slate-200/80 dark:border-slate-800 shadow-xs bg-white dark:bg-slate-900 space-y-4 lg:sticky lg:top-8">
          <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
            <div className="flex items-center gap-2">
              <SlidersHorizontal className="h-4 w-4 text-emerald-700 dark:text-emerald-400" />
              <h3 className="font-bold text-xs text-slate-900 dark:text-white uppercase tracking-wider">
                Filters & Search
              </h3>
            </div>
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-[11px] text-rose-600 hover:text-rose-700 font-semibold gap-1"
                onClick={handleResetFilters}
              >
                <RotateCcw className="h-3 w-3" />
                <span>Reset ({activeFilterCount})</span>
              </Button>
            )}
          </div>

          {/* Search Field */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
              Keyword Search
            </label>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-400 pointer-events-none" />
              <input
                type="text"
                placeholder="Actor, action, or IP..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full h-9 pl-9 pr-8 text-xs rounded-xl bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-emerald-600/20 focus:border-emerald-600 font-medium transition-all"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 p-0.5 rounded-full"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          </div>

          {/* Action Filter */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
              Action Type
            </label>
            <Select value={actionFilter} onValueChange={setActionFilter}>
              <SelectTrigger className="h-9 rounded-xl text-xs bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700 px-3 font-medium w-full">
                <Activity className="h-3.5 w-3.5 text-slate-400 mr-1.5 shrink-0" />
                <SelectValue placeholder="All Actions" />
              </SelectTrigger>
              <SelectContent className="rounded-2xl min-w-[200px]">
                <SelectItem value="ALL">All Actions</SelectItem>
                <SelectItem value="QUEUE_JOINED">QUEUE_JOINED</SelectItem>
                <SelectItem value="CONSULTATION_STARTED">CONSULTATION_STARTED</SelectItem>
                <SelectItem value="CONSULTATION_FINISHED">CONSULTATION_FINISHED</SelectItem>
                <SelectItem value="DOCTOR_STATUS_CHANGED">DOCTOR_STATUS_CHANGED</SelectItem>
                <SelectItem value="DOCTOR_CONFIG_UPDATED">DOCTOR_CONFIG_UPDATED</SelectItem>
                <SelectItem value="AUTH_LOGIN">AUTH_LOGIN</SelectItem>
                <SelectItem value="AUTH_REGISTER">AUTH_REGISTER</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Role Filter */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
              Actor Role
            </label>
            <Select value={roleFilter} onValueChange={setRoleFilter}>
              <SelectTrigger className="h-9 rounded-xl text-xs bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700 px-3 font-medium w-full">
                <Users className="h-3.5 w-3.5 text-slate-400 mr-1.5 shrink-0" />
                <SelectValue placeholder="All Roles" />
              </SelectTrigger>
              <SelectContent className="rounded-2xl min-w-[140px]">
                <SelectItem value="ALL">All Roles</SelectItem>
                <SelectItem value="patient">Patient</SelectItem>
                <SelectItem value="doctor">Doctor</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Sort Order */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
              Sort Order
            </label>
            <Select value={sortOrder} onValueChange={(val: "desc" | "asc") => setSortOrder(val)}>
              <SelectTrigger className="h-9 rounded-xl text-xs bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700 px-3 font-medium w-full">
                <ArrowUpDown className="h-3.5 w-3.5 text-slate-400 mr-1.5 shrink-0" />
                <SelectValue placeholder="Sort Order" />
              </SelectTrigger>
              <SelectContent className="rounded-2xl min-w-[150px]">
                <SelectItem value="desc">Newest First</SelectItem>
                <SelectItem value="asc">Oldest First</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Date Range Section */}
          <div className="space-y-1.5 pt-1">
            <div className="flex items-center justify-between">
              <label className="text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
                Date Range
              </label>
              {(startDate || endDate) && (
                <button
                  type="button"
                  onClick={() => {
                    setStartDate("");
                    setEndDate("");
                  }}
                  className="text-[10px] text-rose-500 hover:underline font-semibold"
                >
                  Clear dates
                </button>
              )}
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2 bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 rounded-xl px-3 py-1.5 text-xs">
                <span className="text-[11px] font-semibold text-slate-400 w-8">From</span>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="bg-transparent text-xs text-slate-700 dark:text-slate-200 focus:outline-none cursor-pointer flex-1"
                />
              </div>
              <div className="flex items-center gap-2 bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 rounded-xl px-3 py-1.5 text-xs">
                <span className="text-[11px] font-semibold text-slate-400 w-8">To</span>
                <input
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  className="bg-transparent text-xs text-slate-700 dark:text-slate-200 focus:outline-none cursor-pointer flex-1"
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Right Section: Expanded Audit Log Table */}
        <div className="flex-1 w-full min-w-0 space-y-4">
          <Card className="rounded-[26px] border-slate-200/80 dark:border-slate-800 overflow-hidden shadow-sm">
            <CardContent className="p-0">
              <div
                onScroll={handleContainerScroll}
                className="overflow-x-auto max-h-[calc(100vh-295px)] min-h-[380px] overflow-y-auto scrollbar-thin"
              >
            <table className="w-full text-left text-xs">
              <thead className="sticky top-0 z-10 bg-slate-50 dark:bg-slate-900 border-b border-slate-200/80 dark:border-slate-800 text-slate-500 text-[11px] uppercase tracking-wider font-bold shadow-2xs">
                <tr>
                  <th className="py-3.5 px-6">Timestamp</th>
                  <th className="py-3.5 px-6">Action</th>
                  <th className="py-3.5 px-6">Actor</th>
                  <th className="py-3.5 px-6">Role</th>
                  <th className="py-3.5 px-6">IP Address</th>
                  <th className="py-3.5 px-6 text-right">Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {isLoading ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-slate-400 text-xs">
                      <Loader2 className="h-5 w-5 animate-spin mx-auto mb-1 text-emerald-700" />
                      Loading activity records...
                    </td>
                  </tr>
                ) : isError ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-slate-500 text-xs">
                      <p className="text-rose-500 font-semibold mb-2">
                        Unable to load activity logs: {(error as Error)?.message || "Session expired or connection failed"}
                      </p>
                      <Button
                        onClick={() => {
                          switchPersona("admin").then(() => refetch());
                        }}
                        variant="outline"
                        size="sm"
                        className="rounded-full text-xs font-bold px-4"
                      >
                        Re-authenticate as Admin CEO
                      </Button>
                    </td>
                  </tr>
                ) : logs.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-slate-400 text-xs">
                      No activity logs matching the selected filter criteria.
                    </td>
                  </tr>
                ) : (
                  logs.map((log) => (
                    <tr key={log.id} className="hover:bg-slate-50/60 dark:hover:bg-slate-800/40 transition-colors">
                      <td className="py-3.5 px-6 whitespace-nowrap">
                        <div className="font-semibold text-xs text-slate-800 dark:text-slate-200">
                          {formatDate(log.created_at)}
                        </div>
                        <div className="text-[11px] font-mono text-slate-400 dark:text-slate-500">
                          {formatTime(log.created_at)}
                        </div>
                      </td>
                      <td className="py-4 px-6">
                        <span className="font-mono font-bold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-2.5 py-1 rounded-full text-[10px]">
                          {log.action}
                        </span>
                      </td>
                      <td className="py-4 px-6 font-semibold text-slate-900 dark:text-white">
                        {log.actor_name}
                      </td>
                      <td className="py-4 px-6">
                        <span
                          className={`inline-block text-[10px] uppercase font-bold px-2 py-0.5 rounded-full ${
                            log.role === "admin"
                              ? "bg-purple-50 text-purple-700"
                              : log.role === "doctor"
                              ? "bg-emerald-50 text-emerald-700"
                              : "bg-blue-50 text-blue-700"
                          }`}
                        >
                          {log.role}
                        </span>
                      </td>
                      <td className="py-4 px-6 font-mono text-slate-400">
                        {log.ip_address}
                      </td>
                      <td className="py-4 px-6 text-right">
                        <Button
                          onClick={() => setInspectLog(log)}
                          variant="outline"
                          size="sm"
                          className="h-7 rounded-full px-3 text-xs font-bold"
                        >
                          <Eye className="h-3 w-3 mr-1" />
                          JSON
                        </Button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>

            {/* Auto-load sentinel at bottom of table */}
            <div ref={loadMoreSentinelRef} className="py-3 px-6 flex items-center justify-center">
              {isFetchingNextPage && (
                <div className="flex items-center gap-2 text-xs text-slate-400">
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-emerald-700" />
                  <span>Loading older activity logs...</span>
                </div>
              )}
            </div>
          </div>
        </CardContent>

        {/* Automatic Infinite Scroll Status Footer */}
        <div className="flex flex-col sm:flex-row items-center justify-between border-t border-slate-100 dark:border-slate-800 px-6 py-3.5 gap-3 text-xs text-slate-500 bg-slate-50/40 dark:bg-slate-900/40">
          <span>
            Showing <strong>{logs.length}</strong> of <strong>{totalRecords}</strong> activity records
          </span>

          <div className="flex items-center gap-2">
            {isFetchingNextPage ? (
              <span className="inline-flex items-center text-xs text-slate-500 font-medium">
                <Loader2 className="h-3.5 w-3.5 animate-spin mr-1.5 text-emerald-700" />
                Auto-loading more records...
              </span>
            ) : hasNextPage ? (
              <span className="text-[11px] text-slate-400 font-medium">
                ↓ Scroll down inside table to auto-load older records
              </span>
            ) : logs.length > 0 ? (
              <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-800 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-3 py-1 rounded-full border border-emerald-200/60">
                <CheckCircle2 className="h-3.5 w-3.5" />
                All {totalRecords} records loaded
              </span>
            ) : null}
          </div>
        </div>
      </Card>
        </div>
      </div>

      {/* Enhanced JSON Metadata Inspector Modal */}
      <Dialog open={!!inspectLog} onOpenChange={(open) => !open && setInspectLog(null)}>
        <DialogContent className="sm:max-w-2xl rounded-[28px] p-6 sm:p-7 gap-5 max-h-[90vh] overflow-y-auto">
          {inspectLog && (
            <>
              <DialogHeader className="space-y-3 pr-8">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="inline-flex items-center gap-1 text-[11px] font-mono font-bold text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-slate-800 px-2.5 py-1 rounded-lg">
                    #{inspectLog.id.length > 8 ? `${inspectLog.id.substring(0, 8)}...` : inspectLog.id}
                  </span>
                  <span className="font-mono font-bold text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/50 border border-emerald-200/60 dark:border-emerald-800/60 px-3 py-1 rounded-full text-xs">
                    {inspectLog.action}
                  </span>
                  <div className="flex items-center gap-1.5 text-xs text-slate-400 font-mono ml-auto">
                    <Clock className="h-3.5 w-3.5" />
                    <span>{formatDateTime(inspectLog.created_at)}</span>
                  </div>
                </div>

                <div>
                  <DialogTitle className="text-lg font-bold tracking-tight text-slate-900 dark:text-white">
                    Audit Activity Inspector
                  </DialogTitle>
                  <DialogDescription className="text-xs text-slate-500 mt-0.5">
                    Immutable event record captured with structured JSONB metadata payload.
                  </DialogDescription>
                </div>
              </DialogHeader>

              {/* 4 Metadata Overview Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/70 dark:border-slate-800 space-y-1">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                    Actor
                  </span>
                  <div className="flex items-center gap-2 min-w-0">
                    <div className="h-6 w-6 rounded-full bg-emerald-600 text-white flex items-center justify-center text-[10px] font-bold shrink-0">
                      {inspectLog.actor_name?.charAt(0) || "U"}
                    </div>
                    <span className="text-xs font-bold text-slate-900 dark:text-white whitespace-normal break-words leading-tight">
                      {inspectLog.actor_name}
                    </span>
                  </div>
                </div>

                <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/70 dark:border-slate-800 space-y-1">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                    Role
                  </span>
                  <div>
                    <span
                      className={`inline-block text-[10px] uppercase font-bold px-2.5 py-0.5 rounded-full ${
                        inspectLog.role === "admin"
                          ? "bg-purple-50 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300"
                          : inspectLog.role === "doctor"
                          ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300"
                          : "bg-blue-50 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300"
                      }`}
                    >
                      {inspectLog.role}
                    </span>
                  </div>
                </div>

                <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/70 dark:border-slate-800 space-y-1">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                    IP Address
                  </span>
                  <div className="flex items-center gap-1.5 text-xs font-mono font-semibold text-slate-700 dark:text-slate-300">
                    <Globe className="h-3.5 w-3.5 text-slate-400 shrink-0" />
                    <span className="truncate">{inspectLog.ip_address || "127.0.0.1"}</span>
                  </div>
                </div>

                {(() => {
                  const identity = getHumanIdentityHandle(inspectLog);
                  return (
                    <div className="p-3.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/70 dark:border-slate-800 space-y-1">
                      <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                        {identity.label}
                      </span>
                      <span className="text-xs font-mono font-bold text-emerald-700 dark:text-emerald-400 block truncate">
                        {identity.handle}
                      </span>
                    </div>
                  );
                })()}
              </div>

              {/* JSON Metadata Viewer Card (Clean Light Aesthetic) */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-xs font-bold text-slate-700 dark:text-slate-300">
                    <FileJson className="h-4 w-4 text-emerald-600" />
                    <span>Event Payload (PostgreSQL JSONB)</span>
                  </div>
                  <Button
                    onClick={handleCopyJSON}
                    variant="outline"
                    size="sm"
                    className="h-7 px-2.5 rounded-lg text-xs font-semibold gap-1.5 hover:bg-slate-100 dark:hover:bg-slate-800"
                  >
                    {copied ? (
                      <>
                        <Check className="h-3.5 w-3.5 text-emerald-600" />
                        <span className="text-emerald-600 font-bold">Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="h-3.5 w-3.5 text-slate-400" />
                        <span>Copy JSON</span>
                      </>
                    )}
                  </Button>
                </div>

                {/* Light Mode Code Container with Clean Frame */}
                <div className="rounded-2xl bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-800 overflow-hidden shadow-xs">
                  <div className="flex items-center justify-between px-4 py-2 bg-slate-100/90 dark:bg-slate-800/80 border-b border-slate-200 dark:border-slate-700">
                    <div className="flex items-center gap-1.5">
                      <div className="h-2.5 w-2.5 rounded-full bg-slate-300 dark:bg-slate-600" />
                      <div className="h-2.5 w-2.5 rounded-full bg-slate-300 dark:bg-slate-600" />
                      <div className="h-2.5 w-2.5 rounded-full bg-slate-300 dark:bg-slate-600" />
                    </div>
                    <span className="text-[10px] font-mono text-slate-400 font-medium">payload.json</span>
                  </div>
                  <div className="p-4 overflow-x-auto max-h-72 font-mono text-xs text-slate-800 dark:text-slate-200 leading-relaxed scrollbar-thin">
                    <pre>{JSON.stringify(inspectLog.details || inspectLog.metadata || {}, null, 2)}</pre>
                  </div>
                </div>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
