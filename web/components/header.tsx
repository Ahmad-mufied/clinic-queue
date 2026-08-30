"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useAuth, DEMO_PERSONAS } from "@/hooks/use-auth";
import { useSSE } from "@/hooks/use-sse";
import { Button } from "@/components/ui/button";
import { BrandLogo } from "@/components/brand-logo";
import {
  Search,
  Bell,
  Calendar,
  UserCheck,
  Check,
  ChevronDown,
  Menu,
  X,
  LogOut,
  User as UserIcon,
  Radio,
  Ticket,
  Stethoscope,
  Activity,
  ShieldCheck,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function Header() {
  const { user, isMounted, activePersonaId, switchPersona, logout, isLoading } = useAuth();
  const { isConnected, notifications, markAllAsRead, clearNotifications } = useSSE();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const currentDateFormatted = new Date().toLocaleDateString("en-US", {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  });

  const userRole = user?.role || "patient";

  // Filter notifications strictly according to the active persona role:
  const visibleNotifications = notifications.filter((notif) => {
    if (userRole === "admin") {
      return true; // Super Admin / CEO has full visibility into all events & forensic audit trail
    }
    if (userRole === "doctor") {
      // Doctors see queue, consultation, doctor shifts, and target speed configs (never audit logs)
      return (
        notif.category === "doctor" ||
        notif.category === "consultation" ||
        notif.category === "queue" ||
        notif.type === "DOCTOR_CONFIG_UPDATED"
      );
    }
    // Patients & Public walk-ins only see consultation, doctor shifts, and queue synchronizations
    return notif.category === "queue" || notif.category === "doctor" || notif.category === "consultation";
  });

  const visibleUnreadCount = visibleNotifications.filter((n) => !n.read).length;

  const homeHref = isMounted && user?.role === "doctor" ? "/doctor" : isMounted && user?.role === "patient" ? "/patient" : "/dashboard";

  return (
    <header className="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-slate-200/80 bg-white/90 dark:bg-slate-900/90 px-4 sm:px-8 backdrop-blur-md">
      {/* Left: Brand Logo */}
      <div className="flex items-center gap-3">
        <Link href={homeHref} className="flex items-center gap-2.5 font-extrabold text-sm text-slate-900 dark:text-white">
          <BrandLogo size="sm" />
          <span className="font-black text-sm tracking-tight">SmartClinic</span>
        </Link>
      </div>

      {/* Right Controls matching reference header */}
      <div className="flex items-center gap-3">
        {/* Date Indicator Pill */}
        <div className="hidden md:flex items-center gap-2 rounded-full bg-[#f1f3f6] dark:bg-slate-800 px-4 py-1.5 text-xs text-slate-600 dark:text-slate-300 font-medium">
          <Calendar className="h-3.5 w-3.5 text-slate-400" />
          <span suppressHydrationWarning>{currentDateFormatted}</span>
        </div>

        {/* Real-Time SSE Notification Center Popover */}
        <DropdownMenu onOpenChange={(open) => open && markAllAsRead()}>
          <DropdownMenuTrigger asChild>
            <button
              className="relative flex h-9 w-9 items-center justify-center rounded-full bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-500 hover:text-slate-900 transition-colors cursor-pointer"
              title={isConnected ? "Real-time SSE Stream Active" : "SSE Connecting..."}
            >
              <Bell className="h-4 w-4" />
              {visibleUnreadCount > 0 ? (
                <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-emerald-600 text-[9px] font-bold text-white shadow-xs animate-in zoom-in">
                  {visibleUnreadCount > 9 ? "9+" : visibleUnreadCount}
                </span>
              ) : (
                <span
                  className={`absolute top-0 right-0 h-2.5 w-2.5 rounded-full ring-2 ring-white dark:ring-slate-900 ${
                    isConnected ? "bg-emerald-500 animate-pulse" : "bg-rose-500"
                  }`}
                />
              )}
            </button>
          </DropdownMenuTrigger>

          <DropdownMenuContent align="end" className="w-80 sm:w-96 rounded-2xl p-0 shadow-2xl border border-slate-200/80 overflow-hidden">
            {/* Header */}
            <div className="p-3.5 bg-slate-50/80 dark:bg-slate-800/80 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="font-extrabold text-xs text-slate-900 dark:text-white">
                  Real-time Event Stream
                </span>
                <span
                  className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full ${
                    isConnected
                      ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400"
                      : "bg-rose-50 text-rose-700 dark:bg-rose-950/60"
                  }`}
                >
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${
                      isConnected ? "bg-emerald-500 animate-pulse" : "bg-rose-500"
                    }`}
                  />
                  {isConnected ? "NATS Live" : "Reconnecting"}
                </span>
              </div>

              {visibleNotifications.length > 0 && (
                <button
                  onClick={clearNotifications}
                  className="text-[11px] font-semibold text-slate-400 hover:text-rose-600 transition-colors"
                >
                  Clear All
                </button>
              )}
            </div>

            {/* Notification Event List */}
            <div className="max-h-80 overflow-y-auto divide-y divide-slate-100 dark:divide-slate-800">
              {visibleNotifications.length === 0 ? (
                <div className="py-10 px-4 text-center space-y-2">
                  <div className="h-10 w-10 rounded-2xl bg-slate-100 dark:bg-slate-800 text-slate-400 flex items-center justify-center mx-auto">
                    <Radio className="h-5 w-5" />
                  </div>
                  <p className="text-xs font-bold text-slate-700 dark:text-slate-300">
                    No Live Events Yet
                  </p>
                  <p className="text-[11px] text-slate-400 max-w-xs mx-auto">
                    {userRole === "patient"
                      ? "Queue and doctor status updates will stream here automatically."
                      : userRole === "doctor"
                      ? "Patient calls and shift updates will stream here automatically."
                      : "Domain mutations and audit trail events will stream here automatically."}
                  </p>
                </div>
              ) : (
                visibleNotifications.map((notif) => {
                  const isConsultation = notif.category === "consultation";
                  const isDoctor = notif.category === "doctor";
                  const isQueue = notif.category === "queue";
                  const isAdmin = notif.category === "admin";

                  return (
                    <div
                      key={notif.id}
                      className={`p-3.5 flex items-start gap-3 transition-colors ${
                        notif.read ? "bg-white dark:bg-slate-900" : "bg-emerald-50/30 dark:bg-emerald-950/20"
                      }`}
                    >
                      <div
                        className={`h-8 w-8 rounded-xl flex items-center justify-center shrink-0 mt-0.5 ${
                          isConsultation
                            ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-300"
                            : isDoctor
                            ? "bg-blue-100 text-blue-800 dark:bg-blue-900/60 dark:text-blue-300"
                            : isQueue
                            ? "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-300"
                            : "bg-purple-100 text-purple-800 dark:bg-purple-900/60 dark:text-purple-300"
                        }`}
                      >
                        {isConsultation ? (
                          <Stethoscope className="h-4 w-4" />
                        ) : isDoctor ? (
                          <Activity className="h-4 w-4" />
                        ) : isQueue ? (
                          <Ticket className="h-4 w-4" />
                        ) : (
                          <ShieldCheck className="h-4 w-4" />
                        )}
                      </div>

                      <div className="flex-1 overflow-hidden">
                        <div className="flex items-center justify-between gap-2">
                          <h5 className="text-xs font-bold text-slate-900 dark:text-white truncate">
                            {notif.title}
                          </h5>
                          <span className="text-[10px] font-mono text-slate-400 shrink-0">
                            {notif.timeFormatted}
                          </span>
                        </div>
                        <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
                          {notif.description}
                        </p>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Interactive User Profile Dropdown with Logout */}
        {!isMounted ? (
          <div className="h-9 w-9 rounded-full bg-slate-100 dark:bg-slate-800 animate-pulse" />
        ) : user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex items-center gap-2.5 p-1 sm:px-2.5 sm:py-1.5 rounded-full hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer group">
                <div className="h-9 w-9 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold text-xs shadow-sm group-hover:ring-2 group-hover:ring-emerald-400/40 transition-all">
                  {user.name.charAt(0)}
                </div>
                <div className="hidden xl:flex flex-col text-left leading-tight">
                  <span className="text-xs font-bold text-slate-900 dark:text-white">{user.name}</span>
                  <span className="text-[10px] text-slate-400 capitalize">{user.role}</span>
                </div>
                <ChevronDown className="hidden sm:block h-3.5 w-3.5 text-slate-400 group-hover:text-slate-600 transition-colors" />
              </button>
            </DropdownMenuTrigger>

            <DropdownMenuContent align="end" className="w-64 rounded-2xl p-2 shadow-xl border border-slate-200/80">
              <DropdownMenuLabel className="p-2">
                <div className="flex items-center gap-3">
                  <div className="h-10 w-10 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold text-sm shrink-0">
                    {user.name.charAt(0)}
                  </div>
                  <div className="overflow-hidden">
                    <p className="text-xs font-extrabold text-slate-900 dark:text-white truncate">{user.name}</p>
                    <p className="text-[11px] text-slate-400 font-mono truncate">@{user.username}</p>
                    <span className="inline-block mt-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400 capitalize">
                      {user.role}
                    </span>
                  </div>
                </div>
              </DropdownMenuLabel>

              {/* Universal Persona Switcher Section */}
              <DropdownMenuSeparator />
              <div className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                Switch Test Persona
              </div>
              {DEMO_PERSONAS.map((p) => {
                const isSelected = user.username === p.username;
                return (
                  <DropdownMenuItem
                    key={p.id}
                    onClick={async () => {
                      await switchPersona(p.id);
                      if (p.role === "admin") {
                        window.location.href = "/dashboard";
                      } else if (p.role === "doctor") {
                        window.location.href = "/doctor";
                      } else {
                        window.location.href = "/patient";
                      }
                    }}
                    disabled={isLoading}
                    className={`flex items-center justify-between rounded-xl px-2.5 py-1.5 text-xs cursor-pointer ${
                      isSelected ? "bg-emerald-50 text-emerald-800 font-bold dark:bg-emerald-950/50" : ""
                    }`}
                  >
                    <div className="flex flex-col">
                      <span className="font-semibold text-slate-900 dark:text-white">{p.label}</span>
                      <span className="text-[10px] text-slate-400">{p.name} ({p.role})</span>
                    </div>
                    {isSelected && <Check className="h-4 w-4 text-emerald-600" />}
                  </DropdownMenuItem>
                );
              })}

              <DropdownMenuSeparator />

              <DropdownMenuItem
                onClick={logout}
                className="flex items-center gap-2 rounded-xl px-2.5 py-2 text-xs font-bold text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30 cursor-pointer focus:bg-rose-50 focus:text-rose-600"
              >
                <LogOut className="h-4 w-4" />
                <span>Log out</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <Button asChild size="sm" className="rounded-full bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs h-9 px-4">
            <Link href="/portal">Sign In</Link>
          </Button>
        )}
      </div>

      {/* Mobile Drawer Menu with Role-Based Filtering */}
      {mobileMenuOpen && (
        <div className="absolute top-16 left-0 right-0 border-b bg-white dark:bg-slate-900 p-4 shadow-xl lg:hidden flex flex-col gap-2 font-semibold text-xs">
          {user?.role === "admin" && (
            <>
              <Link
                href="/dashboard"
                onClick={() => setMobileMenuOpen(false)}
                className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
              >
                Dashboard
              </Link>
              <Link
                href="/admin"
                onClick={() => setMobileMenuOpen(false)}
                className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
              >
                Doctor Management
              </Link>
              <Link
                href="/admin/audit"
                onClick={() => setMobileMenuOpen(false)}
                className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
              >
                Audit Trail
              </Link>
            </>
          )}

          {user?.role === "doctor" && (
            <Link
              href="/doctor"
              onClick={() => setMobileMenuOpen(false)}
              className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
            >
              Doctor Room
            </Link>
          )}

          {user?.role === "patient" && (
            <Link
              href="/patient"
              onClick={() => setMobileMenuOpen(false)}
              className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
            >
              My Queue & Ticket
            </Link>
          )}

          <Link
            href="/display"
            onClick={() => setMobileMenuOpen(false)}
            className="px-3 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-100 rounded-xl"
          >
            Waiting Room TV
          </Link>
        </div>
      )}
    </header>
  );
}
