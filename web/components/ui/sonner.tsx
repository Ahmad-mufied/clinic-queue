"use client";

import { useTheme } from "next-themes";
import { Toaster as Sonner } from "sonner";

type ToasterProps = React.ComponentProps<typeof Sonner>;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-white dark:group-[.toaster]:bg-slate-900 group-[.toaster]:text-slate-900 dark:group-[.toaster]:text-slate-100 group-[.toaster]:border group-[.toaster]:border-slate-200/80 dark:group-[.toaster]:border-slate-800 group-[.toaster]:shadow-xl group-[.toaster]:rounded-2xl group-[.toaster]:p-4 group-[.toaster]:gap-3 group-[.toaster]:font-sans",
          title: "font-semibold text-xs text-slate-900 dark:text-slate-100 tracking-tight",
          description: "text-[11px] text-slate-500 dark:text-slate-400 mt-0.5 leading-relaxed",
          actionButton:
            "group-[.toast]:bg-emerald-700 group-[.toast]:text-white font-semibold text-xs rounded-xl px-3 py-1.5 shadow-sm hover:bg-emerald-800 transition-colors",
          cancelButton:
            "group-[.toast]:bg-slate-100 dark:group-[.toast]:bg-slate-800 group-[.toast]:text-slate-700 dark:group-[.toast]:text-slate-300 font-semibold text-xs rounded-xl px-3 py-1.5 hover:bg-slate-200 transition-colors",
          closeButton:
            "hover:scale-105 active:scale-95 transition-transform",
          icon: "group-[.toast]:scale-110",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
