import { useMemo, useState } from "react";
import { Activity, AlertTriangle, Copy, Cpu, Database, Flame, LockKeyhole, Loader2, Share2 } from "lucide-react";
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
import { SelectorField } from "./service-overview-selector-field";

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

const DEFAULT_WINDOW_MS = 60 * 60 * 1000;

function pad(value: number) {
  return value.toString().padStart(2, "0");
}

function formatDateTimeLocal(date: Date) {
  return [
    date.getFullYear(),
    "-",
    pad(date.getMonth() + 1),
    "-",
    pad(date.getDate()),
    "T",
    pad(date.getHours()),
    ":",
    pad(date.getMinutes()),
    ":",
    pad(date.getSeconds()),
  ].join("");
}

function parseDateTimeLocal(value: string) {
  const [datePart = "", timePart = ""] = value.split("T");
  const [year = "0", month = "0", day = "0"] = datePart.split("-");
  const [hours = "0", minutes = "0", seconds = "0"] = timePart.split(":");
  return new Date(Number(year), Number(month) - 1, Number(day), Number(hours), Number(minutes), Number(seconds));
}

type ServiceOverviewProps = {
  activeView: DiagnosisView;
  onViewChange: (view: DiagnosisView) => void;
};

export function ServiceOverview({ activeView, onViewChange }: ServiceOverviewProps) {
  const [namespace, setNamespace] = useState("");
  const [service, setService] = useState("");
  const [pod, setPod] = useState("");
  const [windowStart, setWindowStart] = useState(() => formatDateTimeLocal(new Date(Date.now() - DEFAULT_WINDOW_MS)));
  const [windowEnd, setWindowEnd] = useState(() => formatDateTimeLocal(new Date()));
  const [copyStatus, setCopyStatus] = useState("");
  const handleNamespaceChange = (nextNamespace: string) => {
    setNamespace(nextNamespace);
    if (nextNamespace.trim() !== namespace.trim()) {
      setService("");
      setPod("");
    }
  };
  const handleServiceChange = (nextService: string) => {
    setService(nextService);
    if (nextService.trim() !== service.trim()) {
      setPod("");
    }
  };
  const updateWindowStart = (nextStart: string) => {
    const start = parseDateTimeLocal(nextStart);
    const end = parseDateTimeLocal(windowEnd);
    setWindowStart(nextStart);
    if (start.getTime() > end.getTime()) {
      setWindowEnd(nextStart);
    }
  };
  const updateWindowEnd = (nextEnd: string) => {
    const start = parseDateTimeLocal(windowStart);
    const end = parseDateTimeLocal(nextEnd);
    setWindowEnd(nextEnd);
    if (end.getTime() < start.getTime()) {
      setWindowStart(nextEnd);
    }
  };
  const selectorParams = useMemo(() => {
    return new URLSearchParams({
      start: parseDateTimeLocal(windowStart).toISOString(),
      end: parseDateTimeLocal(windowEnd).toISOString(),
      limit: "5000",
    });
  }, [windowEnd, windowStart]);
  const params = useMemo(() => {
    const value = new URLSearchParams({
      namespace,
      service,
      profile_type: profileTypeFor(activeView),
      start: parseDateTimeLocal(windowStart).toISOString(),
      end: parseDateTimeLocal(windowEnd).toISOString(),
    });
    if (pod.trim()) {
      value.set("pod", pod.trim());
    }
    return value;
  }, [activeView, namespace, pod, service, windowEnd, windowStart]);
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
      `start=${parseDateTimeLocal(windowStart).toISOString()}`,
      `end=${parseDateTimeLocal(windowEnd).toISOString()}`,
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
          <SelectorField
            candidates={selectorCatalog.namespaces}
            label="Namespace"
            loading={selectorSummary.loading}
            onChange={handleNamespaceChange}
            placeholder="All namespaces"
            value={namespace}
          />
          <SelectorField
            candidates={selectorCatalog.services}
            label="Service"
            loading={selectorSummary.loading}
            onChange={handleServiceChange}
            placeholder="All services"
            value={service}
          />
          <SelectorField
            candidates={selectorCatalog.pods}
            label="Pod"
            loading={selectorSummary.loading}
            onChange={setPod}
            placeholder="single Java Pod"
            value={pod}
          />
          <div className="context-field context-time-range">
            <span>Time range</span>
            <div className="time-range-inputs">
              <label>
                <span>From</span>
                <input type="datetime-local" step={1} value={windowStart} onChange={(event) => updateWindowStart(event.target.value)} />
              </label>
              <label>
                <span>To</span>
                <input type="datetime-local" step={1} value={windowEnd} onChange={(event) => updateWindowEnd(event.target.value)} />
              </label>
            </div>
          </div>
          <div className="context-field context-field-note">
            {selectorSummary.loading ? (
              <span className="selector-status" aria-live="polite">
                <Loader2 size={12} className="selector-spinner" />
                Refreshing live suggestions
              </span>
            ) : selectorSummary.error ? (
              <span className="warning">Selector suggestions unavailable. You can still type any value.</span>
            ) : (
              <span>Choose from live suggestions or type a value directly.</span>
            )}
          </div>
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
