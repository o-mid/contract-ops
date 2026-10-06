"use client";

import { motion } from "framer-motion";

const rows = [
  { type: "sync.succeeded", status: "processed", delay: 0 },
  { type: "purchase.failed", status: "failed", delay: 0.15 },
  { type: "webhook.received", status: "pending", delay: 0.3 },
  { type: "credential.verified", status: "processed", delay: 0.45 }
];

const statusColor: Record<string, string> = {
  processed: "bg-emerald-400",
  pending: "bg-amber-400",
  failed: "bg-rose-400"
};

export function HeroVisual() {
  return (
    <div className="relative aspect-[16/10] w-full overflow-hidden rounded-2xl border border-white/[0.08] bg-slate-950 shadow-[0_0_0_1px_rgba(255,255,255,0.04),0_24px_80px_rgba(0,0,0,0.55)]">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_60%_at_50%_-10%,rgba(99,102,241,0.35),transparent)]" />
      <div
        className="absolute inset-0 opacity-[0.35]"
        style={{
          backgroundImage:
            "linear-gradient(rgba(148,163,184,0.08) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.08) 1px, transparent 1px)",
          backgroundSize: "24px 24px"
        }}
      />

      <div className="relative flex h-10 items-center gap-2 border-b border-white/[0.06] px-4">
        <span className="h-2.5 w-2.5 rounded-full bg-rose-400/80" />
        <span className="h-2.5 w-2.5 rounded-full bg-amber-400/80" />
        <span className="h-2.5 w-2.5 rounded-full bg-emerald-400/80" />
        <span className="ml-2 text-xs font-medium text-slate-500">Activity — live</span>
        <motion.span
          className="ml-auto flex items-center gap-1.5 text-xs text-emerald-400"
          animate={{ opacity: [0.5, 1, 0.5] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
          SSE
        </motion.span>
      </div>

      <div className="relative space-y-2 p-4 sm:p-5">
        {rows.map((row, index) => (
          <motion.div
            key={row.type}
            initial={{ opacity: 0, x: 12 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.45, delay: 0.2 + row.delay }}
            className="flex items-center gap-3 rounded-xl border border-white/[0.06] bg-white/[0.03] px-3 py-2.5 backdrop-blur-sm"
          >
            <span className={`h-2 w-2 shrink-0 rounded-full ${statusColor[row.status]}`} />
            <div className="min-w-0 flex-1">
              <p className="truncate font-mono text-xs text-slate-200 sm:text-sm">{row.type}</p>
              <p className="truncate text-[10px] text-slate-500 sm:text-xs">evt_demo_{index + 1}</p>
            </div>
            <span className="hidden rounded-md bg-slate-800/80 px-2 py-0.5 text-[10px] uppercase tracking-wide text-slate-400 sm:inline">
              {row.status}
            </span>
          </motion.div>
        ))}
      </div>

      <motion.div
        className="pointer-events-none absolute -right-8 -top-8 h-32 w-32 rounded-full bg-cyan-500/20 blur-3xl"
        animate={{ scale: [1, 1.15, 1], opacity: [0.4, 0.7, 0.4] }}
        transition={{ duration: 5, repeat: Infinity }}
      />
    </div>
  );
}
