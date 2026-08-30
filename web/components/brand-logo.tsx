import React from "react";
import { Layers } from "lucide-react";

interface BrandLogoProps {
  size?: "sm" | "md" | "lg";
  className?: string;
  iconClassName?: string;
}

export function BrandLogo({ size = "md", className = "", iconClassName = "" }: BrandLogoProps) {
  const sizeClasses = {
    sm: "h-8 w-8 rounded-xl",
    md: "h-10 w-10 rounded-2xl",
    lg: "h-12 w-12 rounded-2xl",
  };

  const iconSizes = {
    sm: "h-4 w-4",
    md: "h-5 w-5",
    lg: "h-6 w-6",
  };

  return (
    <div
      className={`flex items-center justify-center bg-gradient-to-tr from-[#047857] to-[#10b981] text-white shadow-sm shadow-emerald-900/15 shrink-0 ring-4 ring-emerald-50/80 dark:ring-slate-800/80 ${sizeClasses[size]} ${className}`}
    >
      <Layers className={`${iconSizes[size]} ${iconClassName}`} strokeWidth={2} />
    </div>
  );
}
