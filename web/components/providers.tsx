"use client";

import React, { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { AuthProvider } from "@/hooks/use-auth";
import { SSEProvider } from "@/hooks/use-sse";
import { Toaster } from "@/components/ui/sonner";

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 1000 * 5,
            refetchOnWindowFocus: false,
            retry: (failureCount, error: any) => {
              if (error?.status === 401 || error?.status === 403 || error?.status === 404) {
                return false;
              }
              return failureCount < 2;
            },
          },
        },
      })
  );

  // Global browser safety for unhandledrejection events (e.g. from EventSource / Chrome Extensions / Network blips)
  useEffect(() => {
    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      if (
        event.reason instanceof Event ||
        typeof event.reason === "undefined" ||
        event.reason?.message?.includes?.("fetch") ||
        event.reason?.message?.includes?.("Failed to fetch") ||
        event.reason?.name === "TypeError"
      ) {
        event.preventDefault();
      }
    };

    window.addEventListener("unhandledrejection", handleUnhandledRejection);
    return () => {
      window.removeEventListener("unhandledrejection", handleUnhandledRejection);
    };
  }, []);

  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <SSEProvider>
            {children}
            <Toaster
              position="top-right"
              expand={true}
              closeButton={true}
              duration={4000}
              visibleToasts={4}
              gap={10}
            />
          </SSEProvider>
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
