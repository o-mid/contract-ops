"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { Activity, Cable, Home, Settings } from "lucide-react";
import type { AppView } from "../../app/useUrlFilters";
import { cn } from "../../lib/cn";

type AppShellProps = {
  view: AppView;
  onNavigate: (view: AppView) => void;
  children: ReactNode;
  streamLive: boolean;
};

const nav: { id: AppView; label: string; icon: typeof Activity }[] = [
  { id: "events", label: "Activity", icon: Activity },
  { id: "connections", label: "Connections", icon: Cable },
  { id: "settings", label: "Settings", icon: Settings }
];

export function AppShell({ view, onNavigate, children, streamLive }: AppShellProps) {
  return (
    <div className="min-h-screen bg-[#06080f] text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-[90rem]">
        <aside className="hidden w-60 shrink-0 flex-col border-r border-white/[0.06] bg-[#06080f] p-4 md:flex">
          <Link href="/" className="mb-8 block rounded-xl px-2 py-1 transition hover:bg-white/[0.04]">
            <p className="text-[11px] font-bold uppercase tracking-[0.2em] text-indigo-400">Contract Ops</p>
            <p className="mt-1 text-sm text-slate-500">Integrations control plane</p>
          </Link>
          <nav className="flex flex-1 flex-col gap-1" aria-label="Console">
            {nav.map((item) => {
              const Icon = item.icon;
              const active = view === item.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onNavigate(item.id)}
                  className={cn(
                    "flex items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-medium transition",
                    active
                      ? "bg-indigo-500/15 text-indigo-100 ring-1 ring-indigo-400/25"
                      : "text-slate-400 hover:bg-white/[0.04] hover:text-white"
                  )}
                  aria-current={active ? "page" : undefined}
                >
                  <Icon className="h-4 w-4 shrink-0" aria-hidden />
                  {item.label}
                </button>
              );
            })}
          </nav>
          <div className="mt-4 space-y-2">
            <Link
              href="/"
              className="flex items-center gap-2 rounded-xl px-3 py-2 text-xs text-slate-500 transition hover:bg-white/[0.04] hover:text-slate-300"
            >
              <Home className="h-3.5 w-3.5" aria-hidden />
              Marketing site
            </Link>
            <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] px-3 py-2.5 text-xs text-slate-500">
              <span
                className={cn(
                  "mr-2 inline-block h-2 w-2 rounded-full",
                  streamLive ? "bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.7)]" : "bg-slate-600"
                )}
                aria-hidden
              />
              {streamLive ? "Live stream" : "Connecting…"}
            </div>
          </div>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex items-center justify-between border-b border-white/[0.06] px-4 py-3 md:hidden">
            <p className="text-[11px] font-bold uppercase tracking-[0.2em] text-indigo-400">Contract Ops</p>
            <select
              className="rounded-lg border border-white/10 bg-white/[0.03] px-2 py-1.5 text-sm"
              value={view}
              onChange={(event) => onNavigate(event.target.value as AppView)}
              aria-label="Section"
            >
              {nav.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.label}
                </option>
              ))}
            </select>
          </header>
          <main className="flex-1 p-4 md:p-8 lg:p-10">{children}</main>
        </div>
      </div>
    </div>
  );
}
