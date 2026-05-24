import { useEffect, useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata, TopStackRow } from "../../api/types";
import { Flamegraph } from "../../visualization/flamegraph";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
  topRows?: TopStackRow[];
  profileWindow?: { start: Date; end: Date; durationMs: number };
  profileType?: string;
  analysisLabel?: string;
  title?: string;
  description?: string;
  valueLabel?: string;
  selfColumnLabel?: string;
  totalColumnLabel?: string;
  flamegraphEmptyMessage?: string;
};

type HotFrame = {
  name: string;
  symbol: string;
  fullSymbol: string;
  className: string;
  line?: number;
  self: number;
  total: number;
  selfDisplay?: string;
  totalDisplay?: string;
  selfPercent: number | string;
  totalPercent: number | string;
};

type ViewMode = "top-table" | "flame-graph" | "both";
type SortKey = "total" | "self" | "symbol";
const noApplicationFramesMessage = "No application Java frames were found in this profile. Use the flame graph to inspect runtime or native frames.";

export function HotCodeView({ root, metadata, topRows, profileWindow, profileType = "java_cpu_nanoseconds", analysisLabel = "CPU profile analysis", title = "Single Pod CPU profile", description = "Top table ranks Java methods by CPU time. Values are rendered from nanoseconds into incident-readable time and average cores.", valueLabel = "CPU time", selfColumnLabel = "Self CPU", totalColumnLabel = "Total CPU", flamegraphEmptyMessage }: Props) {
  const fallbackFrames = useMemo(() => collectHotJavaFrames(root), [root]);
  const hotFrames = useMemo(() => (topRows && topRows.length > 0 ? topRows.map(topRowToHotFrame) : fallbackFrames), [fallbackFrames, topRows]);
  const [selectedName, setSelectedName] = useState<string | undefined>();
  const [searchQuery, setSearchQuery] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("both");
  const [sortKey, setSortKey] = useState<SortKey>("total");
  const visibleFrames = useMemo(() => filterHotFrames(hotFrames, searchQuery), [hotFrames, searchQuery]);
  const sortedFrames = useMemo(() => sortHotFrames(visibleFrames, sortKey), [visibleFrames, sortKey]);
  const selected = selectedName ? hotFrames.find((frame) => frameMatchesName(frame, selectedName)) : undefined;
  const fallbackSelected = sortedFrames[0];
  const summaryFrame = selected ?? fallbackSelected;
  const highlightQuery = selected ? selected.fullSymbol || selected.name || selected.symbol : "";
  const valueFormatter = (value: number) => formatProfileValue(profileType, value, profileWindow);

  useEffect(() => {
    if (!selected) return;
    const activeRow = Array.from(document.querySelectorAll<HTMLButtonElement>("button[data-frame-name]")).find((button) => frameMatchesName(selected, button.dataset.frameName ?? ""));
    if (activeRow && typeof activeRow.scrollIntoView === "function") {
      activeRow.scrollIntoView({ block: "nearest" });
    }
  }, [selected?.name]);

  if (hotFrames.length === 0) {
    const emptyMessage = hasProfileSamples(root) ? noApplicationFramesMessage : (flamegraphEmptyMessage ?? noApplicationFramesMessage);
    return (
      <section className="profile-analysis profile-analysis-wide" aria-label={analysisLabel}>
        <h2>{title}</h2>
        <p className="flamegraph-empty">{emptyMessage}</p>
        <div className="profile-flamegraph">
          <Flamegraph root={root} metadata={metadata} profileType={profileType} />
        </div>
      </section>
    );
  }

  return (
    <section className="profile-analysis profile-analysis-wide" aria-label={analysisLabel}>
      <div className="profile-toolbar profile-toolbar-compact">
        <div className="profile-toolbar-copy">
          <h2>{title}</h2>
          <p>{description}</p>
        </div>
        <div className="profile-view-toggle" aria-label="Profile view mode">
          <button className={viewMode === "top-table" ? "active" : ""} aria-pressed={viewMode === "top-table"} onClick={() => setViewMode("top-table")}>Top Table</button>
          <button className={viewMode === "flame-graph" ? "active" : ""} aria-pressed={viewMode === "flame-graph"} onClick={() => setViewMode("flame-graph")}>Flame Graph</button>
          <button className={viewMode === "both" ? "active" : ""} aria-pressed={viewMode === "both"} onClick={() => setViewMode("both")}>Both</button>
        </div>
      </div>
      <div className="cpu-unit-strip" aria-label="CPU profile units">
        <div>
          <span>Value unit</span>
          <strong>{valueLabel}</strong>
        </div>
        <div>
          <span>Percent basis</span>
          <strong>Returned profile</strong>
        </div>
        <div>
          <span>Window</span>
          <strong>{profileWindow ? formatWindow(profileWindow) : "selected range"}</strong>
        </div>
      </div>
      {summaryFrame && (
        <div className="selected-hot-frame" role="region" aria-label="Selected hot Java frame">
          <div>
            <span>{selected ? "Selected Java frame" : "Top Java frame"}</span>
            <strong>{summaryFrame.fullSymbol}</strong>
          </div>
          <dl>
            <div>
              <dt>{selfColumnLabel}</dt>
              <dd>{summaryFrame.selfDisplay ?? valueFormatter(summaryFrame.self)} <span>{formatPercent(summaryFrame.selfPercent)}</span></dd>
            </div>
            <div>
              <dt>{totalColumnLabel}</dt>
              <dd>{summaryFrame.totalDisplay ?? valueFormatter(summaryFrame.total)} <span>{formatPercent(summaryFrame.totalPercent)}</span></dd>
            </div>
          </dl>
          <p>{hotFrameDiagnosis(summaryFrame)}</p>
        </div>
      )}
      <div className={viewMode === "both" ? "profile-grid profile-grid-wide" : "profile-stack"}>
        {viewMode !== "flame-graph" && <TopTable frames={sortedFrames} selected={selected ?? fallbackSelected} explicitSelection={Boolean(selected)} sortKey={sortKey} onSort={setSortKey} onSelect={setSelectedName} valueFormatter={valueFormatter} selfColumnLabel={selfColumnLabel} totalColumnLabel={totalColumnLabel} />}
        {viewMode !== "top-table" && (
          <div className="profile-flamegraph">
            <Flamegraph
              root={root}
              metadata={metadata}
              profileType={profileType}
              emptyMessage={flamegraphEmptyMessage}
              highlightQuery={highlightQuery}
              searchQuery={searchQuery}
              onSearchQueryChange={setSearchQuery}
              onReset={() => setSelectedName(undefined)}
              formatValue={valueFormatter}
              valueLabel={valueLabel}
              showSelectedDetail={false}
              onFrameSelect={setSelectedName}
              inspectorTotalLabel={totalColumnLabel}
              inspectorSelfLabel={selfColumnLabel}
              detailTotalPercentLabel={`${totalColumnLabel} %`}
              detailSelfPercentLabel={`${selfColumnLabel} %`}
            />
          </div>
        )}
      </div>
    </section>
  );
}

function hasProfileSamples(root: FlamegraphNode) {
  if (root.value > 0) return true;
  return (root.children ?? []).some(hasProfileSamples);
}

export function collectHotJavaFrames(root: FlamegraphNode): HotFrame[] {
  const totals = new Map<string, { frame: ParsedFrame; self: number; total: number; hottestLine?: { line: number; samples: number; name: string } }>();
  const totalSamples = Math.max(1, root.value);
  const visit = (node: FlamegraphNode) => {
    const parsed = parseJavaFrame(node.name);
    if (parsed && isApplicationFrame(parsed)) {
      const key = `${parsed.fullClassName}.${parsed.method}`;
      const current = totals.get(key);
      const total = Math.max(0, node.value);
      const childTotal = (node.children ?? []).reduce((sum, child) => sum + Math.max(0, child.value), 0);
      const self = Math.max(0, total - childTotal);
      const hottestLine =
        parsed.line && total >= (current?.hottestLine?.samples ?? -1)
          ? { line: parsed.line, samples: total, name: parsed.name }
          : current?.hottestLine;
      totals.set(key, { frame: parsed, self: (current?.self ?? 0) + self, total: (current?.total ?? 0) + total, hottestLine });
    }
    for (const child of node.children ?? []) visit(child);
  };
  visit(root);
  return Array.from(totals.values())
    .map(({ frame, self, total, hottestLine }) => ({
      ...frame,
      name: hottestLine?.name ?? frame.name,
      line: hottestLine?.line ?? frame.line,
      symbol: `${frame.className}.${frame.method}`,
      fullSymbol: `${frame.fullClassName}.${frame.method}`,
      self,
      total,
      selfPercent: (self / totalSamples) * 100,
      totalPercent: (total / totalSamples) * 100,
    }));
}

type ParsedFrame = {
  name: string;
  fullClassName: string;
  className: string;
  method: string;
  line?: number;
};

function parseJavaFrame(name: string): ParsedFrame | undefined {
  const normalized = name.replaceAll("/", ".");
  const lineMatch = /:(\d+)$/.exec(normalized);
  const line = lineMatch ? Number(lineMatch[1]) : undefined;
  const frameWithoutLine = lineMatch ? normalized.slice(0, -lineMatch[0].length) : normalized;
  const methodSeparator = frameWithoutLine.lastIndexOf(".");
  if (methodSeparator < 0) return undefined;
  const className = frameWithoutLine.slice(0, methodSeparator);
  const method = frameWithoutLine.slice(methodSeparator + 1);
  if (!method || method.includes("[")) return undefined;
  return { name, fullClassName: className, className: className.split(".").slice(-1)[0] ?? className, method, line };
}

function isApplicationFrame(frame: ParsedFrame) {
  if (frame.name.includes("$$Lambda")) return false;
  const normalizedName = frame.name.toLowerCase();
  const normalizedMethod = frame.method.toLowerCase();
  const normalizedClass = frame.fullClassName.toLowerCase();
  if (
    normalizedName.includes(".so") ||
    normalizedName.includes("[vdso]") ||
    normalizedName.includes("pthread") ||
    normalizedName.includes("clock_gettime") ||
    normalizedName.includes("stubroutine") ||
    normalizedName.includes("vtablestub") ||
    normalizedName.includes("itable stub") ||
    normalizedName.includes("i2c adapter") ||
    normalizedName.includes("c2i adapter") ||
    normalizedName.includes("adapter for")
  ) {
    return false;
  }
  if (/\s/.test(frame.method) || /\s/.test(frame.className)) return false;
  if (/^(read|write|open|close|poll|select|epoll|recv|send|accept|connect|syscall|nanosleep)$/.test(normalizedMethod)) return false;
  if (/^\d/.test(frame.method) || /^\d/.test(frame.className)) return false;
  if (!/^[A-Z_$]/.test(frame.className)) return false;
  const excludedPrefixes = [
    "java.",
    "javax.",
    "jdk.",
    "sun.",
    "com.sun.",
    "org.graalvm.",
    "lib",
  ];
  if (excludedPrefixes.some((prefix) => normalizedClass.startsWith(prefix))) return false;
  const excludedExactClasses = new Set(["i2c", "c2i", "itable", "vtable", "stubroutines"]);
  return !excludedExactClasses.has(normalizedClass) && !excludedExactClasses.has(frame.className.toLowerCase());
}

function sortHotFrames(frames: HotFrame[], sortKey: SortKey) {
  return [...frames].sort((left, right) => {
    if (sortKey === "symbol") return left.symbol.localeCompare(right.symbol) || right.total - left.total;
    if (sortKey === "self") return right.self - left.self || right.total - left.total || left.symbol.localeCompare(right.symbol);
    return right.total - left.total || right.self - left.self || left.symbol.localeCompare(right.symbol);
  });
}

function filterHotFrames(frames: HotFrame[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return frames;
  return frames.filter((frame) =>
    [frame.symbol, frame.fullSymbol, frame.name, frame.className]
      .filter(Boolean)
      .some((value) => value.toLowerCase().includes(normalizedQuery)),
  );
}

function normalizeFrameIdentity(value: string) {
  return value.trim().replaceAll("/", ".").toLowerCase();
}

function frameMatchesName(frame: HotFrame, name: string) {
  const normalizedName = normalizeFrameIdentity(name);
  return [frame.name, frame.fullSymbol, frame.symbol]
    .filter(Boolean)
    .some((candidate) => {
      const normalizedCandidate = normalizeFrameIdentity(candidate);
      return normalizedCandidate === normalizedName || normalizedCandidate.includes(normalizedName) || normalizedName.includes(normalizedCandidate);
    });
}

function TopTable({
  frames,
  selected,
  explicitSelection,
  sortKey,
  onSort,
  onSelect,
  valueFormatter,
  selfColumnLabel,
  totalColumnLabel,
}: {
  frames: HotFrame[];
  selected?: HotFrame;
  explicitSelection?: boolean;
  sortKey: SortKey;
  onSort: (sortKey: SortKey) => void;
  onSelect: (name: string) => void;
  valueFormatter: (value: number) => string;
  selfColumnLabel: string;
  totalColumnLabel: string;
}) {
  return (
    <div className="top-table-wrap" role="region" aria-label="Top table">
      <table className="top-table">
        <thead>
          <tr>
            <th><button className={sortKey === "symbol" ? "active" : ""} onClick={() => onSort("symbol")}>Symbol</button></th>
            <th><button className={sortKey === "self" ? "active" : ""} onClick={() => onSort("self")}>{selfColumnLabel}</button></th>
            <th><button className={sortKey === "total" ? "active" : ""} onClick={() => onSort("total")}>{totalColumnLabel}</button></th>
          </tr>
        </thead>
        <tbody>
          {frames.length === 0 && (
            <tr>
              <td colSpan={3} className="top-table-empty">No Java frames match the current search.</td>
            </tr>
          )}
          {frames.slice(0, 20).map((frame) => (
            <tr key={frame.name} className={explicitSelection && selected && frameMatchesName(frame, selected.name) ? "active" : ""}>
              <td>
                <button data-frame-name={frame.name} onClick={() => onSelect(frame.name)}>
                  <span>{frame.symbol}</span>
                  <small>{frame.line ? `${frame.className}:${frame.line}` : frame.fullSymbol}</small>
                </button>
              </td>
              <td>{frame.selfDisplay ?? valueFormatter(frame.self)} <small>{formatPercent(frame.selfPercent)}</small></td>
              <td>{frame.totalDisplay ?? valueFormatter(frame.total)} <small>{formatPercent(frame.totalPercent)}</small></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatPercent(value: number | string) {
  return typeof value === "string" ? value : `${value.toFixed(1)}%`;
}

function hotFrameDiagnosis(frame: HotFrame) {
  if (frame.total <= 0) return "No sampled CPU was attributed to this Java frame.";
  const selfRatio = frame.self / frame.total;
  if (selfRatio >= 0.5) return "High self means this method's own work is the first optimization target.";
  if (selfRatio <= 0.25) return "High total with low self means inspect callees in the flame graph.";
  return "Self and total are both material; inspect the frame and its callees before choosing the optimization target.";
}

function topRowToHotFrame(row: TopStackRow): HotFrame {
  const parsed = parseJavaFrame(row.location) ?? parseJavaFrame(row.symbol);
  const symbol = row.symbol || parsed?.name || row.location;
  const className = parsed?.className ?? symbol.split(".").slice(-2, -1)[0] ?? symbol;
  return {
    name: row.location || symbol,
    symbol,
    fullSymbol: parsed ? `${parsed.fullClassName}.${parsed.method}` : row.location || symbol,
    className,
    line: parsed?.line,
    self: row.self,
    total: row.total,
    selfDisplay: row.self_display,
    totalDisplay: row.total_display,
    selfPercent: row.self_percent,
    totalPercent: row.total_percent,
  };
}

function formatProfileValue(profileType: string, value: number, profileWindow?: { durationMs: number }) {
  if (profileType !== "java_cpu_nanoseconds") {
    return formatDuration(value);
  }
  if (value <= 0) return "0 ns";
  const duration = formatDuration(value);
  if (!profileWindow || profileWindow.durationMs <= 0) return duration;
  const cores = value / (profileWindow.durationMs * 1_000_000);
  if (cores < 0.01) return duration;
  return `${duration} · ${formatCores(cores)}`;
}

function formatDuration(ns: number) {
  if (ns >= 60_000_000_000) return `${(ns / 60_000_000_000).toFixed(1)} min`;
  if (ns >= 1_000_000_000) return `${(ns / 1_000_000_000).toFixed(2)} s`;
  if (ns >= 1_000_000) return `${(ns / 1_000_000).toFixed(1)} ms`;
  if (ns >= 1_000) return `${(ns / 1_000).toFixed(1)} us`;
  return `${ns.toLocaleString()} ns`;
}

function formatCores(cores: number) {
  return `${cores >= 1 ? cores.toFixed(2) : cores.toFixed(3)} cores`;
}

function formatWindow(profileWindow: { start: Date; end: Date }) {
  return `${profileWindow.start.toISOString().slice(11, 16)}-${profileWindow.end.toISOString().slice(11, 16)} UTC`;
}
