"use client";

import React, { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import { Sidebar } from "@/components/sidebar";
import { Header } from "@/components/header";
import { Loader2 } from "lucide-react";

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isMounted } = useAuth();

  // Public standalone pages that do not use the internal sidebar/header shell
  const isStandalone =
    pathname === "/portal" ||
    pathname === "/login" ||
    pathname === "/auth" ||
    pathname === "/display" ||
    (!user && pathname === "/");

  // Redirect to portal only after client is mounted and confirmed unauthenticated on protected routes
  useEffect(() => {
    if (isMounted && !user && !isStandalone) {
      router.replace("/portal");
    }
  }, [user, isMounted, isStandalone, router]);

  // Standalone public routes (TV Display, Fullscreen Login, or Unauthenticated Root) render immediately
  if (isStandalone) {
    return <>{children}</>;
  }

  // If not mounted yet or unauthenticated on protected routes, show loading while redirecting to /login
  if (!isMounted || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#f4f6f8] dark:bg-slate-950">
        <div className="flex flex-col items-center gap-2">
          <Loader2 className="h-6 w-6 animate-spin text-emerald-600" />
          <span className="text-xs text-slate-400 font-medium">Redirecting to login...</span>
        </div>
      </div>
    );
  }

  // Single-workspace clinical/patient views without redundant multi-menu sidebars
  const isPatientView = pathname === "/patient" || user?.role === "patient";
  const isDoctorView = pathname === "/doctor" || user?.role === "doctor";
  const isCockpitView = isPatientView || isDoctorView;

  // Instant zero-delay workspace layout
  return (
    <div className="flex min-h-screen" suppressHydrationWarning>
      {!isCockpitView && <Sidebar />}
      <div className="flex-1 flex flex-col min-w-0 overflow-x-hidden">
        {isCockpitView ? (
          <Header />
        ) : (
          <div className="lg:hidden">
            <Header />
          </div>
        )}
        <main
          className={`flex-1 p-4 sm:p-6 lg:p-8 w-full ${
            isPatientView ? "max-w-3xl mx-auto" : isDoctorView ? "max-w-7xl mx-auto" : "max-w-full"
          }`}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
