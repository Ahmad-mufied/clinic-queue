"use client";

import React, { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth, DEMO_PERSONAS } from "@/hooks/use-auth";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { BrandLogo } from "@/components/brand-logo";
import {
  Layers,
  Lock,
  User,
  ShieldCheck,
  Stethoscope,
  Users,
  ArrowRight,
  Loader2,
  Tv,
  Eye,
  EyeOff,
} from "lucide-react";
import { toast } from "sonner";

function PortalForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectPath = searchParams.get("redirect");

  const { user, login, register, switchPersona, isLoading } = useAuth();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [selectedRoleTab, setSelectedRoleTab] = useState<"all" | "admin" | "doctor" | "patient">("all");
  const [activePreviewPersona, setActivePreviewPersona] = useState(DEMO_PERSONAS[0]);

  // Form states
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [name, setName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Handle manual login or registration submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      toast.error("Please enter both username and password");
      return;
    }

    setIsSubmitting(true);
    try {
      if (mode === "login") {
        const loggedUser = await login(username, password);
        toast.success("Authentication successful");
        if (redirectPath) {
          router.push(redirectPath);
        } else if (loggedUser.role === "admin") {
          router.push("/dashboard");
        } else if (loggedUser.role === "doctor") {
          router.push("/doctor");
        } else {
          router.push("/patient");
        }
      } else {
        if (!name) {
          toast.error("Please enter full name for registration");
          setIsSubmitting(false);
          return;
        }
        await register(username, password, name);
        toast.success("Patient account created successfully");
        router.push("/patient");
      }
    } catch (err: any) {
      toast.error("Authentication Failed", {
        description: err.message || "Invalid credentials. Please verify and try again.",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleQuickDemoSelect = async (personaId: string) => {
    setIsSubmitting(true);
    try {
      const persona = DEMO_PERSONAS.find((p) => p.id === personaId);
      if (persona) setActivePreviewPersona(persona);
      await switchPersona(personaId);
      toast.success(`Signed in as ${persona?.name || "Demo User"}`);
      if (persona?.role === "admin") {
        router.push("/dashboard");
      } else if (persona?.role === "doctor") {
        router.push("/doctor");
      } else {
        router.push("/patient");
      }
    } catch (err: any) {
      toast.error("Sign in failed", { description: err.message });
    } finally {
      setIsSubmitting(false);
    }
  };

  const filteredPersonas = DEMO_PERSONAS.filter((p) => {
    if (selectedRoleTab === "all") return true;
    return p.role === selectedRoleTab;
  });

  return (
    <div className="w-full max-w-5xl bg-white dark:bg-slate-900 rounded-[38px] sm:rounded-[48px] shadow-2xl shadow-emerald-950/12 border border-emerald-100/70 dark:border-slate-800 p-4 sm:p-6 lg:p-7 grid lg:grid-cols-12 gap-6 relative transition-all items-stretch overflow-hidden">
      {/* 1. Main Section: Brand Header & Persona Grid (7 Cols) */}
      <div className="lg:col-span-7 flex flex-col justify-between space-y-4 py-1 px-1">
        {/* Modern Minimalist Header */}
        <div className="flex items-center justify-between pb-3.5 border-b border-slate-100 dark:border-slate-800/80">
          <div className="flex items-center gap-3.5">
            <BrandLogo size="md" />
            <div>
              <h1 className="text-lg sm:text-xl font-bold tracking-tight text-slate-900 dark:text-white">
                SmartClinic OS
              </h1>
              <p className="text-xs text-slate-400 font-normal mt-0.5">
                Sign in to your clinical workspace or select a demo profile.
              </p>
            </div>
          </div>
        </div>

        {/* 1-Click Persona Access Panel */}
        <div className="p-4 sm:p-5 rounded-[26px] bg-slate-50/60 dark:bg-slate-800/30 border border-slate-100/80 dark:border-slate-800/60 flex-1 flex flex-col justify-start space-y-3.5">
          <div className="flex items-center justify-between flex-wrap gap-2">
            <div>
              <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block">
                Quick Demo Access
              </span>
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Select a pre-configured profile
              </span>
            </div>

            {/* Sleek Minimalist Segmented Tabs */}
            <div className="flex bg-slate-200/50 dark:bg-slate-800/80 p-0.5 rounded-xl text-xs">
              {(
                [
                  { key: "all", label: "All (5)" },
                  { key: "admin", label: "Admin (1)" },
                  { key: "doctor", label: "Doctors (2)" },
                  { key: "patient", label: "Patients (2)" },
                ] as const
              ).map((tab) => (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setSelectedRoleTab(tab.key)}
                  className={`px-3 py-1 rounded-lg transition-all text-xs font-medium ${
                    selectedRoleTab === tab.key
                      ? "bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-2xs font-semibold"
                      : "text-slate-500 hover:text-slate-900 dark:hover:text-slate-300"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Unified Persona Grid for All Tabs */}
          <div className="grid gap-2.5 sm:grid-cols-2 min-h-[280px] content-start">
            {filteredPersonas.map((p) => {
              const isAdmin = p.role === "admin";
              const isDoctor = p.role === "doctor";
              const isSelected = activePreviewPersona.id === p.id;

              return (
                <div
                  key={p.id}
                  onClick={() => {
                    if (isSubmitting) return;
                    setActivePreviewPersona(p);
                    setUsername(p.username);
                    setPassword(p.password);
                    handleQuickDemoSelect(p.id);
                  }}
                  className={`p-3.5 rounded-2xl border transition-all duration-200 cursor-pointer text-left relative flex flex-col justify-between group ${
                    isSelected
                      ? "bg-emerald-50/40 dark:bg-emerald-950/30 border-emerald-500/80 shadow-xs ring-1 ring-emerald-500/20"
                      : "bg-white dark:bg-slate-900 border-slate-200/70 dark:border-slate-800 hover:border-emerald-400/80 hover:shadow-xs hover:-translate-y-0.5"
                  }`}
                >
                  <div className="flex items-start gap-3">
                    <div
                      className={`h-9 w-9 rounded-xl flex items-center justify-center shrink-0 border transition-transform duration-200 group-hover:scale-105 ${
                        isAdmin
                          ? "bg-purple-50 dark:bg-purple-950/40 text-purple-600 dark:text-purple-400 border-purple-100/80 dark:border-purple-900/50"
                          : isDoctor
                          ? "bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border-emerald-100/80 dark:border-emerald-900/50"
                          : "bg-sky-50 dark:bg-sky-950/40 text-sky-600 dark:text-sky-400 border-sky-100/80 dark:border-sky-900/50"
                      }`}
                    >
                      {isAdmin ? (
                        <ShieldCheck className="h-4 w-4" strokeWidth={2} />
                      ) : isDoctor ? (
                        <Stethoscope className="h-4 w-4" strokeWidth={2} />
                      ) : (
                        <User className="h-4 w-4" strokeWidth={2} />
                      )}
                    </div>
                    <div className="overflow-hidden flex-1 min-w-0">
                      <div className="text-xs font-bold text-slate-800 dark:text-slate-100 truncate">
                        {p.label}
                      </div>
                      <div className="text-[11px] text-slate-400 dark:text-slate-400 font-normal truncate mt-0.5">
                        {p.description}
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 pt-2 border-t border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <span className="font-mono text-[10px] text-slate-400">@{p.username}</span>
                    <div
                      className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-semibold bg-slate-100 group-hover:bg-emerald-600 text-slate-700 group-hover:text-white dark:bg-slate-800 dark:text-slate-300 dark:group-hover:bg-emerald-600 dark:group-hover:text-white transition-all shadow-2xs"
                    >
                      <span>Sign In</span>
                      <ArrowRight className="h-2.5 w-2.5 transition-transform group-hover:translate-x-0.5" strokeWidth={2} />
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* 2. Right Panel: Credentials Form (5 Cols) */}
      <div className="lg:col-span-5 rounded-[32px] bg-slate-50/80 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 p-6 flex flex-col justify-between space-y-5 relative">
        <div className="space-y-4">
          {/* Header Title */}
          <div>
            <h2 className="text-lg font-bold text-slate-900 dark:text-white tracking-tight">
              {mode === "login" ? "Sign In to Account" : "Register Patient Account"}
            </h2>
            <p className="text-xs text-slate-400 mt-0.5 font-normal">
              {mode === "login"
                ? "Enter your credentials below to enter workspace."
                : "Create a new patient account to take tickets."}
            </p>
          </div>

          {/* Mode Switcher (Unified Segmented Style) */}
          <div className="flex bg-slate-100 dark:bg-slate-800/80 p-1 rounded-2xl text-xs font-semibold">
            <button
              type="button"
              onClick={() => setMode("login")}
              className={`flex-1 py-1.5 rounded-xl transition-all text-center ${
                mode === "login"
                  ? "bg-white dark:bg-slate-900 text-emerald-800 dark:text-emerald-300 shadow-2xs font-bold"
                  : "text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 font-medium"
              }`}
            >
              Sign In
            </button>
            <button
              type="button"
              onClick={() => setMode("register")}
              className={`flex-1 py-1.5 rounded-xl transition-all text-center ${
                mode === "register"
                  ? "bg-white dark:bg-slate-900 text-emerald-800 dark:text-emerald-300 shadow-2xs font-bold"
                  : "text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 font-medium"
              }`}
            >
              Register
            </button>
          </div>

          {/* Credentials Form */}
          <form onSubmit={handleSubmit} className="space-y-3 pt-1">
            {mode === "register" && (
              <div className="space-y-1">
                <label className="text-[11px] font-semibold text-slate-700 dark:text-slate-300">
                  Full Name
                </label>
                <div className="relative">
                  <User className="absolute left-3.5 top-3 h-4 w-4 text-slate-400" strokeWidth={1.75} />
                  <Input
                    placeholder="e.g. John Doe"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    disabled={isSubmitting}
                    className="pl-10 h-10 rounded-2xl text-xs bg-white dark:bg-slate-900 border-slate-200/80 focus:border-emerald-500 focus:ring-emerald-500/20"
                  />
                </div>
              </div>
            )}

            <div className="space-y-1">
              <label className="text-[11px] font-semibold text-slate-700 dark:text-slate-300">
                Username
              </label>
              <div className="relative">
                <User className="absolute left-3.5 top-3 h-4 w-4 text-slate-400" strokeWidth={1.75} />
                <Input
                  placeholder="admin, doctor_a, patient_john"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={isSubmitting}
                  className="pl-10 h-10 rounded-2xl text-xs font-mono bg-white dark:bg-slate-900 border-slate-200/80 focus:border-emerald-500 focus:ring-emerald-500/20"
                />
              </div>
            </div>

            <div className="space-y-1">
              <label className="text-[11px] font-semibold text-slate-700 dark:text-slate-300">
                Password
              </label>
              <div className="relative">
                <Lock className="absolute left-3.5 top-3 h-4 w-4 text-slate-400" strokeWidth={1.75} />
                <Input
                  type={showPassword ? "text" : "password"}
                  placeholder="password123"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isSubmitting}
                  className="pl-10 pr-10 h-10 rounded-2xl text-xs bg-white dark:bg-slate-900 border-slate-200/80 focus:border-emerald-500 focus:ring-emerald-500/20"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-2.5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 transition-colors cursor-pointer"
                  title={showPassword ? "Hide password" : "Show password"}
                  tabIndex={-1}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" strokeWidth={1.75} />
                  ) : (
                    <Eye className="h-4 w-4" strokeWidth={1.75} />
                  )}
                </button>
              </div>
            </div>

            <Button
              type="submit"
              disabled={isSubmitting}
              className="w-full h-10 rounded-2xl bg-gradient-to-r from-emerald-600 via-emerald-700 to-teal-700 hover:from-emerald-700 hover:to-teal-800 text-white font-bold text-xs tracking-tight shadow-md shadow-emerald-900/10 active:scale-[0.99] transition-all mt-2 cursor-pointer"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Authenticating...
                </>
              ) : mode === "login" ? (
                <>
                  <span>Sign In to Workspace</span>
                  <ArrowRight className="ml-1.5 h-3.5 w-3.5" strokeWidth={2} />
                </>
              ) : (
                <>
                  <span>Create Account</span>
                  <ArrowRight className="ml-1.5 h-3.5 w-3.5" strokeWidth={2} />
                </>
              )}
            </Button>
          </form>
        </div>

        {/* Public Waiting Room TV Banner link */}
        <div className="pt-2 text-center border-t border-slate-200/60 dark:border-slate-700/50">
          <Link
            href="/display"
            className="inline-flex items-center gap-2 text-xs font-medium text-slate-500 hover:text-emerald-700 dark:hover:text-emerald-400 transition-colors py-1 px-3 rounded-full hover:bg-emerald-50/60 dark:hover:bg-emerald-950/40"
          >
            <Tv className="h-3.5 w-3.5 text-emerald-600" strokeWidth={2} />
            <span>Public Waiting Room TV Monitor &rarr;</span>
          </Link>
        </div>
      </div>
    </div>
  );
}

export default function PortalPage() {
  return (
    <div className="min-h-screen flex items-center justify-center p-3 sm:p-5 lg:p-8 bg-gradient-to-br from-[#dcf0e5] via-[#edf7f2] to-[#e1f2e8] dark:bg-slate-950 relative overflow-hidden">
      {/* Decorative Illustrated Botanical Accents Matching MedBoard */}
      <div className="absolute top-0 left-0 w-80 h-80 bg-emerald-300/20 dark:bg-emerald-950/20 rounded-full blur-3xl pointer-events-none -translate-x-1/2 -translate-y-1/2" />
      <div className="absolute bottom-0 right-0 w-96 h-96 bg-teal-300/20 dark:bg-teal-950/20 rounded-full blur-3xl pointer-events-none translate-x-1/2 translate-y-1/2" />

      <Suspense
        fallback={
          <div className="text-center py-12">
            <Loader2 className="h-6 w-6 animate-spin mx-auto text-emerald-600" />
            <span className="text-xs text-slate-400 mt-2 block">Loading MedBoard portal...</span>
          </div>
        }
      >
        <PortalForm />
      </Suspense>
    </div>
  );
}
