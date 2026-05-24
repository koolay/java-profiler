import { useEffect, useMemo, useRef, useState } from "react";
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
  items: Array<{ view: DiagnosisView; label: string; shortLabel: string; detail: string; tip: string; icon: ReactNode }>;
}> = [
  {
    label: "Profiles",
    items: [
      { view: "cpu", label: "CPU profiles", shortLabel: "CPU", detail: "core", tip: "Core CPU time in Java methods and runtime code.", icon: <Cpu size={16} /> },
      { view: "wall", label: "Wall Clock profiles", shortLabel: "Latency", detail: "latency", tip: "Runnable, blocked, waiting, or sleeping wall time.", icon: <Activity size={16} /> },
      { view: "io", label: "I/O wait profiles", shortLabel: "I/O", detail: "blocking", tip: "Time spent waiting on I/O or blocking work.", icon: <Database size={16} /> },
      { view: "gc", label: "GC pauses", shortLabel: "GC", detail: "events", tip: "Garbage-collection pause events and allocation correlation.", icon: <Activity size={16} /> },
      { view: "memory", label: "Allocation profiles", shortLabel: "Alloc", detail: "alloc", tip: "Allocation pressure and object creation hot paths.", icon: <Flame size={16} /> },
      { view: "locks", label: "Lock diagnosis", shortLabel: "Locks", detail: "locks", tip: "Contended monitor and lock-delay hotspots.", icon: <LockKeyhole size={16} /> },
    ],
  },
  {
    label: "Health",
    items: [
      { view: "status", label: "Service status", shortLabel: "Targets", detail: "targets", tip: "Target discovery and profile eligibility status.", icon: <Activity size={16} /> },
      { view: "ingestion", label: "Ingestion health", shortLabel: "Batches", detail: "batches", tip: "Accepted, dropped, and truncated profile batches.", icon: <Database size={16} /> },
      { view: "deadlocks", label: "Deadlock diagnosis", shortLabel: "Deadlocks", detail: "events", tip: "Detected deadlock cycles and thread ownership.", icon: <AlertTriangle size={16} /> },
    ],
  },
];

const TIME_RANGE_PRESETS = [
  { label: "5m", value: 5 * 60 * 1000 },
  { label: "15m", value: 15 * 60 * 1000 },
  { label: "30m", value: 30 * 60 * 1000 },
  { label: "1h", value: 60 * 60 * 1000 },
] as const;
type TimeRangePreset = (typeof TIME_RANGE_PRESETS)[number]["label"] | "custom";

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

function formatDisplayDateTime(value: string) {
  const date = parseDateTimeLocal(value);
  return [
    date.getFullYear(),
    "/",
    pad(date.getMonth() + 1),
    "/",
    pad(date.getDate()),
    ", ",
    pad(date.getHours()),
    ":",
    pad(date.getMinutes()),
    ":",
    pad(date.getSeconds()),
  ].join("");
}

function formatDurationLabel(startValue: string, endValue: string) {
  const start = parseDateTimeLocal(startValue).getTime();
  const end = parseDateTimeLocal(endValue).getTime();
  const totalMinutes = Math.max(0, Math.round((end - start) / 60000));

  if (totalMinutes < 60) {
    return `${Math.max(totalMinutes, 1)}m`;
  }

  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
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
  const [windowStart, setWindowStart] = useState(() => formatDateTimeLocal(new Date(Date.now() - 15 * 60 * 1000)));
  const [windowEnd, setWindowEnd] = useState(() => formatDateTimeLocal(new Date()));
  const [timeRangePreset, setTimeRangePreset] = useState<TimeRangePreset>("15m");
  const [customRangeOpen, setCustomRangeOpen] = useState(false);
  const [copyStatus, setCopyStatus] = useState("");
  const timeRangeRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onPointerDown(event: PointerEvent) {
      if (customRangeOpen && timeRangeRef.current && !timeRangeRef.current.contains(event.target as Node)) {
        setCustomRangeOpen(false);
      }
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setCustomRangeOpen(false);
      }
    }

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [customRangeOpen]);
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
    setTimeRangePreset("custom");
    setCustomRangeOpen(true);
    if (start.getTime() > end.getTime()) {
      setWindowEnd(nextStart);
    }
  };
  const updateWindowEnd = (nextEnd: string) => {
    const start = parseDateTimeLocal(windowStart);
    const end = parseDateTimeLocal(nextEnd);
    setWindowEnd(nextEnd);
    setTimeRangePreset("custom");
    setCustomRangeOpen(true);
    if (end.getTime() < start.getTime()) {
      setWindowStart(nextEnd);
    }
  };
  const setPresetWindow = (preset: TimeRangePreset) => {
    if (preset === "custom") {
      setTimeRangePreset("custom");
      setCustomRangeOpen((current) => !current);
      return;
    }

    const presetConfig = TIME_RANGE_PRESETS.find((item) => item.label === preset);
    if (!presetConfig) {
      return;
    }

    const end = new Date();
    const start = new Date(end.getTime() - presetConfig.value);
    setWindowStart(formatDateTimeLocal(start));
    setWindowEnd(formatDateTimeLocal(end));
    setTimeRangePreset(preset);
    setCustomRangeOpen(false);
  };
  const selectorParams = useMemo(() => {
    return new URLSearchParams({
      profile_type: profileTypeFor(activeView),
      start: parseDateTimeLocal(windowStart).toISOString(),
      end: parseDateTimeLocal(windowEnd).toISOString(),
      limit: "5000",
    });
  }, [activeView, windowEnd, windowStart]);
  const historicalSelectorParams = useMemo(() => {
    return new URLSearchParams({
      profile_type: profileTypeFor(activeView),
      limit: "5000",
    });
  }, [activeView]);
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
  const historicalSelectorSummary = useAPI(() => getServiceSelectors(historicalSelectorParams), [historicalSelectorParams.toString()], emptySummary);
  const currentTargets = selectorSummary.data?.targets ?? [];
  const historicalTargets = historicalSelectorSummary.data?.targets ?? [];
  const usingHistoricalSuggestions = currentTargets.length === 0 && historicalTargets.length > 0;
  const selectorTargets = usingHistoricalSuggestions ? historicalTargets : currentTargets;
  const selectorCatalog = useMemo(
    () => buildSelectorCatalog(selectorTargets, { namespace, service, pod }),
    [namespace, pod, selectorTargets, service],
  );
  const customRangeSummary = formatDurationLabel(windowStart, windowEnd);
  const customRangeDisplay = `${formatDisplayDateTime(windowStart)} → ${formatDisplayDateTime(windowEnd)}`;
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
        <div className="workbench-context">
          <div className="context-fields context-fields-topbar" aria-label="Service context">
            <SelectorField
              className="selector-field selector-field-namespace"
              candidates={selectorCatalog.namespaces}
              label="Namespace"
              loading={selectorSummary.loading}
              onChange={handleNamespaceChange}
              placeholder="All namespaces"
              value={namespace}
            />
            <span className="context-arrow" aria-hidden="true">
              →
            </span>
            <SelectorField
              className="selector-field selector-field-service"
              candidates={selectorCatalog.services}
              label="Service"
              loading={selectorSummary.loading}
              onChange={handleServiceChange}
              placeholder="All services"
              value={service}
            />
            <span className="context-arrow" aria-hidden="true">
              →
            </span>
            <SelectorField
              className="selector-field selector-field-pod"
              candidates={selectorCatalog.pods}
              label="Pod"
              loading={selectorSummary.loading}
              onChange={setPod}
              placeholder="single Java Pod"
              value={pod}
            />
            <span className="context-arrow context-arrow-time" aria-hidden="true">
              |
            </span>
            <div className="context-field context-time-range" ref={timeRangeRef}>
              <span>Time range</span>
              <div className="time-range-presets" role="group" aria-label="Time range presets">
                {TIME_RANGE_PRESETS.map((preset) => (
                  <button
                    className={timeRangePreset === preset.label ? "active" : ""}
                    key={preset.label}
                    onClick={() => setPresetWindow(preset.label)}
                    type="button"
                  >
                    {preset.label}
                  </button>
                ))}
                <button
                  aria-expanded={customRangeOpen}
                  aria-label="Custom"
                  className={`time-range-custom-toggle${timeRangePreset === "custom" ? " active" : ""}`}
                  title={`Custom time range: ${customRangeDisplay} (${customRangeSummary})`}
                  onClick={() => setPresetWindow("custom")}
                  type="button"
                >
                  Custom
                </button>
              </div>
              {timeRangePreset === "custom" ? <div className="time-range-summary">{customRangeDisplay}</div> : null}
              {customRangeOpen ? (
                <div className="time-range-popover" aria-label="Custom time range">
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
              ) : null}
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
            ) : usingHistoricalSuggestions ? (
              <span className="warning">No selector suggestions have samples in this time range. Showing historical targets; adjust the range or type a value directly.</span>
            ) : (
              <span>Choose from live suggestions or type a value directly.</span>
            )}
          </div>
        </div>
        <div className="workbench-actions">
          <button className="ghost-action" type="button" onClick={copyContext}><Copy size={14} />Copy Link</button>
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
                title={item.tip}
                onClick={() => onViewChange(item.view)}
                type="button"
              >
                <span className="nav-label">{item.icon}{item.shortLabel}</span>
                <span className="nav-count" title={item.tip}>{item.detail}</span>
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
