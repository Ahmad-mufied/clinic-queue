"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { api, clearStoredToken, getStoredToken, isTokenValid, setStoredToken } from "@/lib/api";
import type { DemoPersona, User } from "@/lib/types";

export const DEMO_PERSONAS: DemoPersona[] = [
  {
    id: "admin",
    label: "Admin CEO",
    role: "admin",
    username: "admin",
    password: "password123",
    name: "Clinic Administrator",
    description: "Clinic Management & Analytics",
  },
  {
    id: "doctor_a",
    label: "Dr. Sarah Adams",
    role: "doctor",
    username: "doctor_a",
    password: "password123",
    name: "Dr. Sarah Adams",
    doctor_id: "01919df4-8e3b-7412-a1f9-90b567c9e101",
    description: "General Practitioner • Room 1",
  },
  {
    id: "doctor_b",
    label: "Dr. Michael Chen",
    role: "doctor",
    username: "doctor_b",
    password: "password123",
    name: "Dr. Michael Chen",
    doctor_id: "01919df4-8e3b-7412-a1f9-90b567c9e102",
    description: "Specialist Physician • Room 2",
  },
  {
    id: "patient_john",
    label: "John Doe",
    role: "patient",
    username: "patient_john",
    password: "password123",
    name: "John Doe",
    description: "Walk-in Patient",
  },
  {
    id: "patient_lucas",
    label: "Lucas Smith",
    role: "patient",
    username: "patient_lucas",
    password: "password123",
    name: "Lucas Smith",
    description: "Registered Patient",
  },
];

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isMounted: boolean;
  activePersonaId: string | null;
  login: (username: string, password: string) => Promise<User>;
  register: (username: string, password: string, name: string) => Promise<User>;
  logout: () => void;
  switchPersona: (personaId: string) => Promise<User>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const USER_STORAGE_KEY = "clinic_queue_user";
const PERSONA_STORAGE_KEY = "clinic_queue_persona";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [activePersonaId, setActivePersonaId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isMounted, setIsMounted] = useState<boolean>(false);

  useEffect(() => {
    const handleUnauthorized = () => {
      logout();
    };
    window.addEventListener("clinic:unauthorized", handleUnauthorized);
    return () => window.removeEventListener("clinic:unauthorized", handleUnauthorized);
  }, []);

  useEffect(() => {
    const savedToken = getStoredToken();
    const savedUser = localStorage.getItem(USER_STORAGE_KEY);
    const savedPersona = localStorage.getItem(PERSONA_STORAGE_KEY);

    if (savedToken && isTokenValid(savedToken) && savedUser) {
      try {
        const parsedUser = JSON.parse(savedUser);
        setToken(savedToken);
        setUser(parsedUser);
        if (savedPersona) setActivePersonaId(savedPersona);
        setIsMounted(true);

        // Validate credentials against server
        api.getMe()
          .then((me) => {
            setUser(me);
            localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(me));
          })
          .catch(() => {
            logout();
          });
        return;
      } catch {
        clearStoredToken();
        setUser(null);
        setToken(null);
        setActivePersonaId(null);
        setIsMounted(true);
        return;
      }
    }

    // No valid token: stay unauthenticated and wait for explicit login
    clearStoredToken();
    setUser(null);
    setToken(null);
    setActivePersonaId(null);
    setIsMounted(true);
  }, []);

  const login = async (username: string, password: string): Promise<User> => {
    setIsLoading(true);
    try {
      const resp = await api.login({ username, password });
      setToken(resp.token);
      setUser(resp.user);
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(resp.user));

      const matchedPersona = DEMO_PERSONAS.find((p) => p.username === username);
      if (matchedPersona) {
        setActivePersonaId(matchedPersona.id);
        localStorage.setItem(PERSONA_STORAGE_KEY, matchedPersona.id);
      }
      return resp.user;
    } finally {
      setIsLoading(false);
    }
  };

  const register = async (username: string, password: string, name: string): Promise<User> => {
    setIsLoading(true);
    try {
      const resp = await api.register({ username, password, name });
      setToken(resp.token);
      setUser(resp.user);
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(resp.user));
      return resp.user;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    clearStoredToken();
    localStorage.removeItem(USER_STORAGE_KEY);
    localStorage.removeItem(PERSONA_STORAGE_KEY);
    localStorage.removeItem("clinic_queue_ticket");
    localStorage.removeItem("clinic_doctor_workspace");
    localStorage.removeItem("clinic_admin_stats");
    setUser(null);
    setToken(null);
    setActivePersonaId(null);
    if (typeof window !== "undefined") {
      window.location.href = "/portal";
    }
  };

  const switchPersona = async (personaId: string): Promise<User> => {
    localStorage.removeItem("clinic_queue_ticket");
    localStorage.removeItem("clinic_doctor_workspace");
    localStorage.removeItem("clinic_admin_stats");
    const persona = DEMO_PERSONAS.find((p) => p.id === personaId);
    if (!persona) throw new Error("Persona not found");
    return await login(persona.username, persona.password);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isLoading,
        isMounted,
        activePersonaId,
        login,
        register,
        logout,
        switchPersona,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
