"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight, Cable, Lock, Radio, Sparkles, Workflow } from "lucide-react";
import { AnimatedArchitecture } from "./AnimatedArchitecture";
import { HeroVisual } from "./HeroVisual";
import { MarketingBackground } from "./MarketingBackground";

const features = [
  {
    icon: Radio,
    title: "Live activity feed",
    body: "Server-sent events push a fresh page whenever activity_events changes."
  },
  {
    icon: Lock,
    title: "Sealed credentials",
    body: "Envelope encryption with per-secret data keys; fingerprints only over the wire."
  },
  {
    icon: Workflow,
    title: "Sync control plane",
    body: "Backfill windows, leases, and connector health in one Go API."
  },
  {
    icon: Cable,
    title: "Connector registry",
    body: "Scaffold vendors with make new-connector; demo with fakevendor."
  }
];

const fadeUp = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 }
};

export function LandingPage() {
  return (
    <div className="relative min-h-screen text-slate-100">
      <MarketingBackground />

      <header className="sticky top-0 z-20 border-b border-white/[0.06] bg-[#06080f]/80 backdrop-blur-lg">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link href="/" className="group">
            <p className="text-[11px] font-bold uppercase tracking-[0.22em] text-indigo-400">Contract Ops</p>
            <p className="text-sm text-slate-500 transition group-hover:text-slate-300">
              Integrations control plane
            </p>
          </Link>
          <nav className="flex items-center gap-2 sm:gap-4">
            <a
              href="https://github.com/o-mid/contract-ops"
              className="hidden text-sm text-slate-400 transition hover:text-white sm:inline"
            >
              GitHub
            </a>
            <Link
              href="/console"
              className="inline-flex items-center gap-2 rounded-full bg-white px-4 py-2 text-sm font-semibold text-slate-900 transition hover:bg-slate-100"
            >
              Open console
              <ArrowRight className="h-4 w-4" aria-hidden />
            </Link>
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 pb-24 pt-12 sm:pt-16">
        <section className="grid items-center gap-14 lg:grid-cols-2 lg:gap-12">
          <motion.div {...fadeUp} transition={{ duration: 0.55 }}>
            <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-indigo-500/30 bg-indigo-500/10 px-3 py-1 text-xs font-medium text-indigo-200">
              <Sparkles className="h-3.5 w-3.5" aria-hidden />
              Open source · Next.js console
            </div>
            <h1 className="text-4xl font-semibold leading-[1.08] tracking-tight text-white sm:text-5xl lg:text-[3.25rem]">
              Integrations you can operate, not just deploy
            </h1>
            <p className="mt-5 max-w-xl text-lg leading-relaxed text-slate-400">
              A clean control plane for vendor connections, sync jobs, and a live activity feed — styled
              like modern workflow products, without the heavyweight framework lock-in.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link
                href="/console"
                className="rounded-full bg-indigo-500 px-6 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/25 transition hover:bg-indigo-400"
              >
                Launch console
              </Link>
              <a
                href="https://api-production-4b82.up.railway.app/healthz"
                className="rounded-full border border-slate-700/80 bg-white/[0.03] px-6 py-2.5 text-sm text-slate-300 transition hover:border-slate-500 hover:text-white"
              >
                API health
              </a>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.12 }}
          >
            <HeroVisual />
          </motion.div>
        </section>

        <motion.section
          id="architecture"
          className="mt-24 scroll-mt-24"
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true, margin: "-80px" }}
          transition={{ duration: 0.5 }}
        >
          <AnimatedArchitecture />
        </motion.section>

        <section className="mt-24 grid gap-5 sm:grid-cols-2">
          {features.map((feature, index) => {
            const Icon = feature.icon;
            return (
              <motion.article
                key={feature.title}
                initial={{ opacity: 0, y: 16 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.06, duration: 0.4 }}
                whileHover={{ y: -2 }}
                className="group rounded-2xl border border-white/[0.06] bg-white/[0.02] p-6 transition hover:border-indigo-500/30 hover:bg-white/[0.04]"
              >
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/15 text-indigo-300 transition group-hover:bg-indigo-500/25">
                  <Icon className="h-5 w-5" aria-hidden />
                </div>
                <h3 className="mt-5 text-lg font-semibold text-white">{feature.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-400">{feature.body}</p>
              </motion.article>
            );
          })}
        </section>

        <motion.section
          className="mt-24 overflow-hidden rounded-2xl border border-white/[0.08] bg-gradient-to-b from-indigo-950/50 to-slate-950 p-8 text-center sm:p-10"
          initial={{ opacity: 0, scale: 0.98 }}
          whileInView={{ opacity: 1, scale: 1 }}
          viewport={{ once: true }}
        >
          <h2 className="text-2xl font-semibold text-white">Live on Railway</h2>
          <p className="mx-auto mt-3 max-w-lg text-slate-400">
            Run <code className="rounded bg-black/30 px-1.5 py-0.5 text-indigo-200">make up</code> locally,
            or use the hosted API and console below.
          </p>
          <dl className="mt-8 flex flex-col gap-4 text-sm sm:flex-row sm:justify-center sm:gap-12">
            <div>
              <dt className="text-xs uppercase tracking-wider text-slate-500">API</dt>
              <dd className="mt-1">
                <a className="text-indigo-300 hover:underline" href="https://api-production-4b82.up.railway.app">
                  api-production-4b82.up.railway.app
                </a>
              </dd>
            </div>
            <div>
              <dt className="text-xs uppercase tracking-wider text-slate-500">Console</dt>
              <dd className="mt-1">
                <a className="text-indigo-300 hover:underline" href="https://web-production-a614a.up.railway.app">
                  web-production-a614a.up.railway.app
                </a>
              </dd>
            </div>
          </dl>
        </motion.section>
      </main>
    </div>
  );
}
