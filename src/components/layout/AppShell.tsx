import type { ReactNode } from "react";
import { Activity, Cable, Settings } from "lucide-react";
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
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-[90rem]">
        <aside className="hidden w-56 shrink-0 flex-col border-r border-slate-800 bg-slate-950 p-4 md:flex">
          <div className="mb-8 px-2">
            <p className="text-xs font-bold uppercase tracking-widest text-indigo-400">
              Contract Ops
            </p>
            <p className="mt-1 text-sm text-slate-400">Integrations control plane</p>
          </div>
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
                    "flex items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm font-medium transition",
                    active
                      ? "bg-indigo-600/15 text-indigo-200 ring-1 ring-indigo-500/30"
                      : "text-slate-300 hover:bg-slate-900 hover:text-white"
                  )}
                  aria-current={active ? "page" : undefined}
                >
                  <Icon className="h-4 w-4 shrink-0 opacity-80" aria-hidden />
                  {item.label}
                </button>
              );
            })}
          </nav>
          <div className="mt-auto rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-400">
            <span
              className={cn(
                "mr-2 inline-block h-2 w-2 rounded-full",
                streamLive ? "bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" : "bg-slate-600"
              )}
              aria-hidden
            />
            {streamLive ? "Live stream connected" : "Stream idle"}
          </div>
        </aside>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex items-center justify-between border-b border-slate-800 px-4 py-3 md:hidden">
            <div>
              <p className="text-xs font-bold uppercase tracking-widest text-indigo-400">
                Contract Ops
              </p>
            </div>
            <select
              className="rounded-lg border border-slate-700 bg-slate-900 px-2 py-1.5 text-sm"
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
          <main className="flex-1 p-4 md:p-8">{children}</main>
        </div>
      </div>
    </div>
  );
}
