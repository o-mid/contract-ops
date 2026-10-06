"use client";

type Node = { id: string; label: string; sub: string; x: number; y: number };

const nodes: Node[] = [
  { id: "browser", label: "Console", sub: "EventSource", x: 120, y: 200 },
  { id: "api", label: "API", sub: "Go / chi", x: 320, y: 88 },
  { id: "worker", label: "Worker", sub: "Sync jobs", x: 520, y: 200 },
  { id: "db", label: "Postgres", sub: "LISTEN", x: 720, y: 88 }
];

const paths = [
  { id: "a", d: "M 120 200 Q 220 120 320 88", delay: 0 },
  { id: "b", d: "M 320 88 Q 420 88 520 200", delay: 0.5 },
  { id: "c", d: "M 520 200 Q 620 120 720 88", delay: 1 },
  { id: "d", d: "M 720 88 Q 520 40 320 88", delay: 1.5 },
  { id: "e", d: "M 320 88 Q 220 160 120 200", delay: 2 }
];

export function AnimatedArchitecture() {
  return (
    <div className="relative overflow-hidden rounded-2xl border border-white/[0.08] bg-slate-950/60 p-6 shadow-xl backdrop-blur-md sm:p-8">
      <div className="mb-6 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 className="text-lg font-semibold text-white">Control plane topology</h3>
          <p className="text-sm text-slate-400">Public SSE feed on the left; sync and credentials on the right.</p>
        </div>
        <p className="text-xs font-medium uppercase tracking-widest text-indigo-300/80">Animated arch</p>
      </div>

      <svg viewBox="0 0 840 260" className="h-auto w-full" role="img" aria-label="Architecture diagram">
        <defs>
          <linearGradient id="line-grad" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#818cf8" stopOpacity="0.2" />
            <stop offset="50%" stopColor="#22d3ee" stopOpacity="0.9" />
            <stop offset="100%" stopColor="#34d399" stopOpacity="0.2" />
          </linearGradient>
          <filter id="glow">
            <feGaussianBlur stdDeviation="2" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {paths.map((path) => (
          <g key={path.id}>
            <path
              d={path.d}
              fill="none"
              stroke="url(#line-grad)"
              strokeWidth="2"
              strokeLinecap="round"
              opacity="0.55"
            />
            <path
              d={path.d}
              fill="none"
              stroke="rgba(165,180,252,0.25)"
              strokeWidth="8"
              strokeLinecap="round"
            />
            <circle r="5" fill="#e0e7ff" filter="url(#glow)">
              <animateMotion
                dur="3.2s"
                repeatCount="indefinite"
                begin={`${path.delay}s`}
                path={path.d}
              />
            </circle>
          </g>
        ))}

        {nodes.map((node) => (
          <g key={node.id} transform={`translate(${node.x}, ${node.y})`}>
            <ellipse rx="52" ry="36" fill="#0f172a" stroke="#6366f1" strokeWidth="1.5" />
            <text
              textAnchor="middle"
              y="-4"
              className="fill-slate-100 text-[13px] font-semibold"
              style={{ fontFamily: "system-ui, sans-serif" }}
            >
              {node.label}
            </text>
            <text
              textAnchor="middle"
              y="14"
              className="fill-slate-500 text-[10px]"
              style={{ fontFamily: "system-ui, sans-serif" }}
            >
              {node.sub}
            </text>
          </g>
        ))}
      </svg>
    </div>
  );
}
