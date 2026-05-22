import { useMemo, useState } from "react";
import { Activity, AlertTriangle, Copy, Cpu, Database, Flame, LockKeyhole, Share2 } from "lucide-react";
import type { ReactNode } from "react";
import { getServiceSelectors } from "../api/client";
import type { ProfileType, ServiceSelectors } from "../api/types";
import { CpuView } from "../features/cpu/cpu-view";
import { WallClockView } from "../features/wall-clock/wall-clock-view";
import { IOView } from "../features/io/io-view";
import { GCView } from "../features/gc/gc-view";
import { MemoryView } from "../features/memory/memory-view";
import { LocksView } from "../features/locks/locks-view";
import { DeadlocksView } from "../features/deadlocks/deadlocks-view";
import { TargetStatusView } from "../features/status/target-status-view";
import { IngestionHealthView } from "../features/ingestion/ingestion-health-view";
import type { DiagnosisView } from "../app";
import { useAPI } from "../api/use-api";
import { buildSelectorCatalog } from "./service-overview-selectors";

const tabs = ["memory", "cpu", "wall", "io", "gc", "locks", "deadlocks", "status", "ingestion"] as const;
type Tab = (typeof tabs)[number];

const navigationGroups: Array<{
  label: string;
  items: Array<{ view: DiagnosisView; label: string; shortLabel: string; detail: string; icon: ReactNode }>;
}> = [
  {
    label: "Profiles",
    items: [
      { view: "cpu", label: "CPU profiles", shortLabel: "CPU", detail: "core", icon: <Cpu size={16} /> },
      { view: "wall", label: "Wall Clock profiles", shortLabel: "Latency", detail: "latency", icon: <Activity size={16} /> },
      { view: "io", label: "I/O wait profiles", shortLabel: "I/O", detail: "blocking", icon: <Database size={16} /> },
      { view: "gc", label: "GC pauses", shortLabel: "GC", detail: "events", icon: <Activity size={16} /> },
      { view: "memory", label: "Allocation profiles", shortLabel: "Alloc", detail: "alloc", icon: <Flame size={16} /> },
      { view: "locks", label: "Lock diagnosis", shortLabel: "Locks", detail: "locks", icon: <LockKeyhole size={16} /> },
    ],
  },
  {
    label: "Health",
    items: [
      { view: "status", label: "Service status", shortLabel: "Targets", detail: "targets", icon: <Activity size={16} /> },
      { view: "ingestion", label: "Ingestion health", shortLabel: "Batches", detail: "batches", icon: <Database size={16} /> },
      { view: "deadlocks", label: "Deadlock diagnosis", shortLabel: "Deadlocks", detail: "events", icon: <AlertTriangle size={16} /> },
    ],
  },
];

type ServiceOverviewProps = {
  activeView: DiagnosisView;
  onViewChange: (view: DiagnosisView) => void;
};

export function ServiceOverview({ activeView, onViewChange }: ServiceOverviewProps) {
  const [namespace, setNamespace] = useState("java-profiler-qa");
  const [service, setService] = useState("jdk17-http-demo");
  const [pod, setPod] = useState("");
  const [rangeMinutes, setRangeMinutes] = useState(60);
  const [copyStatus, setCopyStatus] = useState("");
  const selectorParams = useMemo(() => {
    const end = new Date();
    const start = new Date(end.getTime() - rangeMinutes * 60_000);
    return new URLSearchParams({
      start: start.toISOString(),
      end: end.toISOString(),
      limit: "5000",
    });
  }, [rangeMinutes]);
  const params = useMemo(() => {
    const end = new Date();
    const start = new Date(end.getTime() - rangeMinutes * 60_000);
    const value = new URLSearchParams({
      namespace,
      service,
      profile_type: profileTypeFor(activeView),
      start: start.toISOString(),
      end: end.toISOString(),
    });
    if (pod.trim()) {
      value.set("pod", pod.trim());
    }
    return value;
  }, [namespace, service, pod, activeView, rangeMinutes]);
  const emptySummary: ServiceSelectors = { targets: [] };
  const selectorSummary = useAPI(() => getServiceSelectors(selectorParams), [selectorParams.toString()], emptySummary);
  const selectorCatalog = useMemo(
    () => buildSelectorCatalog(selectorSummary.data?.targets ?? [], { namespace, service, pod }),
    [namespace, pod, selectorSummary.data, service],
  );
  const copyContext = async () => {
    const context = [
      `view=${activeView}`,
      `namespace=${namespace}`,
      `service=${service}`,
      `pod=${pod.trim() || "<service-query>"}`,
      `range=${rangeMinutes}m`,
      `profile_type=${profileTypeFor(activeView)}`,
    ].join("\n");
    await navigator.clipboard?.writeText(context);
    setCopyStatus("Context copied");
  };
  const shareView = async () => {
    const url = new URL(window.location.href);
    url.searchParams.set("view", activeView);
    for (const [key, value] of params.entries()) url.searchParams.set(key, value);
    await navigator.clipboard?.writeText(url.toString());
    setCopyStatus("Permalink copied");
  };

  return (
    <div className="workbench-shell">
      <header className="workbench-topbar">
        <div className="workbench-brand">
          <div className="brand-mark">JVM</div>
          <div>
            <strong>Java Profiler</strong>
            <span>Incident workbench</span>
          </div>
        </div>
        <div className="context-fields context-fields-topbar" aria-label="Service context">
          <label className="context-field">
            <span>Namespace</span>
            <input
              autoComplete="off"
              list="namespace-candidates"
              spellCheck={false}
              value={namespace}
              onChange={(event) => setNamespace(event.target.value)}
            />
            <datalist id="namespace-candidates">
              {selectorCatalog.namespaces.map((option) => (
                <option key={option} value={option} />
              ))}
            </datalist>
          </label>
          <label className="context-field">
            <span>Service</span>
            <input
              autoComplete="off"
              list="service-candidates"
              spellCheck={false}
              value={service}
              onChange={(event) => setService(event.target.value)}
            />
            <datalist id="service-candidates">
              {selectorCatalog.services.map((option) => (
                <option key={option} value={option} />
              ))}
            </datalist>
          </label>
          <label className="context-field">
            <span>Pod</span>
            <input
              aria-label="Pod filter"
              autoComplete="off"
              list="pod-candidates"
              placeholder="single Java Pod"
              spellCheck={false}
              value={pod}
              onChange={(event) => setPod(event.target.value)}
            />
            <datalist id="pod-candidates">
              {selectorCatalog.pods.map((option) => (
                <option key={option} value={option} />
              ))}
            </datalist>
          </label>
          <label className="context-field context-range">
            <span>Range</span>
            <select value={rangeMinutes} onChange={(event) => setRangeMinutes(Number(event.target.value))}>
              <option value={15}>Last 15m</option>
              <option value={30}>Last 30m</option>
              <option value={60}>Last 1h</option>
              <option value={360}>Last 6h</option>
            </select>
          </label>
        </div>
        <div className="workbench-actions">
          <button type="button" onClick={copyContext}><Copy size={15} />Copy</button>
          <button className="primary-action" type="button" onClick={shareView}><Share2 size={15} />Share</button>
        </div>
        <span className="sr-only" aria-live="polite">{copyStatus}</span>
      </header>

      <aside className="workbench-side" aria-label="Java profiler navigation">
        {navigationGroups.map((group) => (
          <div className="nav-group" key={group.label}>
            <div className="side-title">{group.label}</div>
            {group.items.map((item) => (
              <button
                key={item.view}
                aria-label={item.label}
                aria-pressed={item.view === activeView}
                className={`workbench-nav-item${item.view === activeView ? " active" : ""}`}
                onClick={() => onViewChange(item.view)}
                type="button"
              >
                <span className="nav-label">{item.icon}{item.shortLabel}</span>
                <span className="nav-count">{item.detail}</span>
              </button>
            ))}
          </div>
        ))}
        <div className="scope-card">
          <div className="side-title">Scope</div>
          <div className="scope-row"><span>Target</span><strong>{pod.trim() ? "Single Pod" : "Service query"}</strong></div>
          <div className="scope-row"><span>Runtime</span><strong>UI v0.1.0 · API Connected</strong></div>
        </div>
      </aside>

      <section className="evidence-main">
        <div className="evidence-health-strip" aria-label="Evidence health">
          <div className="health-chip health-chip-ok">
            <span>Collection</span>
            <strong>{collectionLabelForView(activeView)}</strong>
          </div>
          <div className="health-chip">
            <span>Target scope</span>
            <strong>{pod.trim() ? "Single Pod" : "Service query"}</strong>
          </div>
          <div className="health-chip">
            <span>Sample rate</span>
            <strong>99Hz target</strong>
          </div>
          <div className="health-chip">
            <span>Baseline</span>
            <strong>Pod quota when available</strong>
          </div>
        </div>
        <div className="diagnosis-content">
          {activeView === "memory" && <MemoryView params={params} />}
          {activeView === "cpu" && <CpuView params={params} />}
          {activeView === "wall" && <WallClockView params={params} />}
          {activeView === "io" && <IOView params={params} />}
          {activeView === "gc" && <GCView params={params} />}
          {activeView === "locks" && <LocksView params={params} />}
          {activeView === "deadlocks" && <DeadlocksView params={params} />}
          {activeView === "status" && <TargetStatusView params={params} />}
          {activeView === "ingestion" && <IngestionHealthView />}
        </div>
      </section>
    </div>
  );
}

function profileTypeFor(tab: Tab): ProfileType {
  if (tab === "cpu") return "java_cpu_nanoseconds";
  if (tab === "wall") return "java_wall_clock_nanoseconds";
  if (tab === "io") return "java_io_wait_nanoseconds";
  if (tab === "locks") return "java_lock_delay_nanoseconds";
  return "java_allocation_bytes";
}

function collectionLabelForView(view: DiagnosisView) {
  if (view === "cpu") return "CPU profile";
  if (view === "wall") return "Wall Clock profile";
  if (view === "io") return "I/O wait profile";
  if (view === "gc") return "GC evidence";
  if (view === "locks") return "Lock profile";
  if (view === "memory") return "Allocation profile";
  if (view === "deadlocks") return "Deadlock evidence";
  if (view === "status") return "Target status";
  return "Ingestion health";
}
