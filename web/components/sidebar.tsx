"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth, DEMO_PERSONAS } from "@/hooks/use-auth";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { BrandLogo } from "@/components/brand-logo";
import {
  LayoutDashboard,
  Users,
  Stethoscope,
  BarChart3,
  ShieldCheck,
  Layers,
  Calendar,
  Clock,
  Check,
  ChevronDown,
  LogOut,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function Sidebar() {
  const pathname = usePathname();
  const { user, switchPersona, logout, isLoading: isAuthLoading, isMounted } = useAuth();
  const [currentTime, setCurrentTime] = useState<string>("");

  const currentDateFormatted = new Date().toLocaleDateString("en-US", {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  });

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setCurrentTime(
        now.toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          hour12: true,
        })
      );
    };
    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, []);

  // Query: Live waiting count for badge
  const { data: queueStatus } = useQuery({
    queryKey: ["queue-status"],
    queryFn: () => api.getQueueStatus(),
  });

  const waitingCount =
    queueStatus?.total_waiting ??
    (queueStatus?.queue_list?.length ?? 0);
  const onlineDocs =
    queueStatus?.online_doctors_count ??
    (queueStatus?.online_doctors?.filter((d) => d.is_online).length ?? 0);

  // Role-Based Navigation Matrix
  const rawMenuItems = [
    {
      label: "Dashboard",
      href: "/dashboard",
      icon: LayoutDashboard,
      badge: null,
      roles: ["admin"], // Super Admin / CEO ONLY
    },
    {
      label: "Doctor Management",
      href: "/admin",
      icon: BarChart3,
      badge: onlineDocs > 0 ? `${onlineDocs} active` : null,
      roles: ["admin"], // Super Admin / CEO ONLY
    },
    {
      label: "Audit Trail",
      href: "/admin/audit",
      icon: ShieldCheck,
      badge: null,
      roles: ["admin"], // Super Admin / CEO ONLY
    },
    {
      label: "Doctor Room",
      href: "/doctor",
      icon: Stethoscope,
      badge: onlineDocs > 0 ? `${onlineDocs} active` : null,
      roles: ["doctor"], // Doctor ONLY
    },
    {
      label: "My Queue & Ticket",
      href: "/patient",
      icon: Users,
      badge: waitingCount > 0 ? waitingCount.toString() : null,
      roles: ["patient"], // Patient ONLY
    },
  ];

  // Filter Main Menu items based on active role
  const mainMenuItems = rawMenuItems.filter((item) =>
    user ? item.roles.includes(user.role as any) : false
  );

  const homeHref = isMounted && user?.role === "doctor" ? "/doctor" : isMounted && user?.role === "patient" ? "/patient" : "/dashboard";

  return (
    <aside suppressHydrationWarning className="hidden lg:flex w-64 flex-col justify-between border-r border-slate-200/80 bg-white dark:bg-slate-900 p-5 h-screen sticky top-0 shrink-0 select-none overflow-y-auto">
      <div className="space-y-6">
        {/* Top Header: Brand Logo */}
        <Link href={homeHref} className="flex items-center gap-3 px-1">
          <BrandLogo size="md" />
          <div className="overflow-hidden">
            <h1 className="font-black text-base tracking-tight text-slate-900 dark:text-white leading-tight">
              SmartClinic
            </h1>
            <p className="text-[11px] text-slate-400 font-medium">Healthcare OS</p>
          </div>
        </Link>

        {/* Live Date & Time Indicator in Sidebar */}
        <div className="flex items-center gap-3 px-3.5 py-2.5 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200/60 dark:border-slate-800">
          <div className="h-8 w-8 rounded-xl bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 flex items-center justify-center shrink-0">
            <Clock className="h-4 w-4" />
          </div>
          <div className="flex flex-col min-w-0">
            <span className="text-xs font-bold text-slate-800 dark:text-slate-200 truncate" suppressHydrationWarning>
              {currentDateFormatted}
            </span>
            <span className="text-[11px] font-mono font-medium text-slate-400 dark:text-slate-400" suppressHydrationWarning>
              {currentTime || "--:--:--"}
            </span>
          </div>
        </div>

        {/* Section: MAIN MENU */}
        <div className="space-y-1.5">
          <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 px-3 block mb-1">
            Main Menu
          </span>
          {!isMounted ? (
            <div className="space-y-1.5 px-1 py-1">
              <div className="h-9 rounded-2xl bg-slate-100 dark:bg-slate-800 animate-pulse" />
              <div className="h-9 rounded-2xl bg-slate-100 dark:bg-slate-800 animate-pulse" />
            </div>
          ) : (
            <nav className="space-y-1">
              {mainMenuItems.map((item) => {
                const isActive =
                  item.href === "/dashboard"
                    ? pathname === "/dashboard"
                    : pathname === item.href || (item.href !== "/dashboard" && pathname.startsWith(item.href) && item.href !== "/admin");
                const Icon = item.icon;

                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`flex items-center justify-between px-3.5 py-2.5 rounded-2xl text-xs font-semibold transition-all duration-150 ${
                      isActive
                        ? "bg-slate-900 text-white shadow-sm dark:bg-emerald-600"
                        : "text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-white"
                    }`}
                  >
                    <div className="flex items-center gap-3">
                      <Icon className={`h-4 w-4 ${isActive ? "text-emerald-400 dark:text-white" : "text-slate-400"}`} />
                      <span>{item.label}</span>
                    </div>
                    {item.badge && (
                      <span
                        className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                          isActive
                            ? "bg-white/20 text-white"
                            : "bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400"
                        }`}
                      >
                        {item.badge}
                      </span>
                    )}
                  </Link>
                );
              })}
            </nav>
          )}
        </div>
      </div>

      {/* Bottom User Profile Dropdown (Pojok Kiri Bawah) */}
      <div className="pt-4 border-t border-slate-100 dark:border-slate-800">
        {!isMounted ? (
          <div className="h-12 rounded-2xl bg-slate-100 dark:bg-slate-800 animate-pulse" />
        ) : user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="w-full flex items-center justify-between p-2 rounded-2xl hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer group text-left">
                <div className="flex items-center gap-2.5 overflow-hidden">
                  <div className="h-9 w-9 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold text-xs shadow-sm group-hover:ring-2 group-hover:ring-emerald-400/40 transition-all shrink-0">
                    {user.name.charAt(0)}
                  </div>
                  <div className="flex flex-col overflow-hidden">
                    <span className="text-xs font-bold text-slate-900 dark:text-white truncate">
                      {user.name}
                    </span>
                    <span className="text-[10px] text-slate-400 capitalize">
                      {user.role}
                    </span>
                  </div>
                </div>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400 group-hover:text-slate-600 shrink-0 transition-colors" />
              </button>
            </DropdownMenuTrigger>

            <DropdownMenuContent align="start" side="top" className="w-64 rounded-2xl p-2 shadow-xl border border-slate-200/80 mb-2">
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
                    disabled={isAuthLoading}
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
          <Link
            href="/portal"
            className="flex items-center justify-center w-full py-2 rounded-xl bg-slate-900 hover:bg-slate-800 text-white font-bold text-xs"
          >
            Sign In
          </Link>
        )}
      </div>
    </aside>
  );
}
