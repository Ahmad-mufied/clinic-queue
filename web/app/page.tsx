"use client";

import React, { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import PortalPage from "@/app/portal/page";
import { Loader2 } from "lucide-react";

export default function RootPage() {
  const { user, isLoading, isMounted } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isMounted && !isLoading && user) {
      if (user.role === "admin") {
        router.replace("/dashboard");
      } else if (user.role === "doctor") {
        router.replace("/doctor");
      } else if (user.role === "patient") {
        router.replace("/patient");
      }
    }
  }, [user, isLoading, isMounted, router]);

  if (!isMounted) {
    return null;
  }

  // When unauthenticated, render the Portal/Login switcher immediately at root
  if (!user) {
    return <PortalPage />;
  }

  // Brief redirection state when authenticated
  return (
    <div className="min-h-[50vh] flex flex-col items-center justify-center space-y-3">
      <Loader2 className="h-6 w-6 animate-spin text-emerald-600" />
      <p className="text-xs text-slate-400 font-medium">
        Redirecting to {user.role === "admin" ? "dashboard" : `${user.role} workspace`}...
      </p>
    </div>
  );
}
