import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center space-y-4 font-mono">
      <div className="text-4xl font-extrabold text-zinc-900 dark:text-white">404</div>
      <div className="text-sm font-semibold text-zinc-700 dark:text-zinc-300 uppercase tracking-wider">
        Page Not Found
      </div>
      <p className="text-xs text-zinc-500 font-sans max-w-sm">
        The page you are looking for does not exist or has been relocated.
      </p>
      <Button asChild className="rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs h-10 px-5">
        <Link href="/dashboard">Return to Dashboard</Link>
      </Button>
    </div>
  );
}
