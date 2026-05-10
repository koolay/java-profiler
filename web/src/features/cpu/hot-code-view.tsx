import { useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata, TopStackRow } from "../../api/types";
import { Flamegraph } from "../../visualization/flamegraph";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
  topRows?: TopStackRow[];
};

type HotFrame = {
  name: string;
  symbol: string;
  fullSymbol: string;
  className: string;
  line?: number;
  self: number;
  total: number;
  selfPercent: number | string;
  totalPercent: number | string;
};

type ViewMode = "top-table" | "flame-graph" | "both";
type SortKey = "total" | "self" | "symbol";

export function HotCodeView({ root, metadata, topRows }: Props) {
  const fallbackFrames = useMemo(() => collectHotJavaFrames(root), [root]);
  const hotFrames = useMemo(() => (topRows && topRows.length > 0 ? topRows.map(topRowToHotFrame) : fallbackFrames), [fallbackFrames, topRows]);
  const [selectedName, setSelectedName] = useState<string | undefined>();
  const [searchQuery, setSearchQuery] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("both");
  const [sortKey, setSortKey] = useState<SortKey>("total");
  const visibleFrames = useMemo(() => filterHotFrames(hotFrames, searchQuery), [hotFrames, searchQuery]);
  const sortedFrames = useMemo(() => sortHotFrames(visibleFrames, sortKey), [visibleFrames, sortKey]);
  const selected = selectedName ? hotFrames.find((frame) => frame.name === selectedName) : undefined;
  const fallbackSelected = sortedFrames[0];
  const highlightQuery = selected ? selected.symbol : "";
  const insight = selected ? describeHotFrame(selected) : undefined;

  if (hotFrames.length === 0) {
    return (
      <section className="profile-analysis" aria-label="CPU profile analysis">
        <h2>CPU profile</h2>
        <p className="flamegraph-empty">No application Java frames were found in this CPU profile. Use the flame graph to inspect runtime or native frames.</p>
        <div className="profile-flamegraph">
          <Flamegraph root={root} metadata={metadata} />
        </div>
      </section>
    );
  }

  return (
    <section className="profile-analysis" aria-label="CPU profile analysis">
      <div className="profile-toolbar">
        <div>
          <h2>CPU profile</h2>
          <p>Top table ranks Java symbols by self and total CPU samples. The flame graph shows sampled stack context, not source call order.</p>
        </div>
        <div className="profile-view-toggle" aria-label="Profile view mode">
          <button className={viewMode === "top-table" ? "active" : ""} onClick={() => setViewMode("top-table")}>Top Table</button>
          <button className={viewMode === "flame-graph" ? "active" : ""} onClick={() => setViewMode("flame-graph")}>Flame Graph</button>
          <button className={viewMode === "both" ? "active" : ""} onClick={() => setViewMode("both")}>Both</button>
        </div>
      </div>
      <div className={viewMode === "both" ? "profile-grid" : "profile-stack"}>
        {viewMode !== "flame-graph" && <TopTable frames={sortedFrames} selected={selected ?? fallbackSelected} explicitSelection={Boolean(selected)} sortKey={sortKey} onSort={setSortKey} onSelect={setSelectedName} />}
        {viewMode !== "top-table" && (
          <div className="profile-flamegraph">
            <Flamegraph
              root={root}
              metadata={metadata}
              highlightQuery={highlightQuery}
              insight={insight}
              searchQuery={searchQuery}
              onSearchQueryChange={setSearchQuery}
              onReset={() => setSelectedName(undefined)}
            />
          </div>
        )}
      </div>
    </section>
  );
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
    normalizedName.includes("adapter") ||
    normalizedName.includes("stubroutine") ||
    normalizedName.includes("vtablestub") ||
    normalizedName.includes("itable stub")
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

function TopTable({
  frames,
  selected,
  explicitSelection,
  sortKey,
  onSort,
  onSelect,
}: {
  frames: HotFrame[];
  selected?: HotFrame;
  explicitSelection?: boolean;
  sortKey: SortKey;
  onSort: (sortKey: SortKey) => void;
  onSelect: (name: string) => void;
}) {
  return (
    <div className="top-table-wrap" role="region" aria-label="Top table">
      <table className="top-table">
        <thead>
          <tr>
            <th><button className={sortKey === "symbol" ? "active" : ""} onClick={() => onSort("symbol")}>Symbol</button></th>
            <th><button className={sortKey === "self" ? "active" : ""} onClick={() => onSort("self")}>Self CPU</button></th>
            <th><button className={sortKey === "total" ? "active" : ""} onClick={() => onSort("total")}>Total CPU</button></th>
          </tr>
        </thead>
        <tbody>
          {frames.slice(0, 20).map((frame) => (
            <tr key={frame.name} className={explicitSelection && frame.name === selected?.name ? "active" : ""}>
              <td>
                <button onClick={() => onSelect(frame.name)}>
                  <span>{frame.symbol}</span>
                  <small>{frame.line ? `${frame.className}:${frame.line}` : frame.fullSymbol}</small>
                </button>
              </td>
              <td>{formatSamples(frame.self)} <small>{formatPercent(frame.selfPercent)}</small></td>
              <td>{formatSamples(frame.total)} <small>{formatPercent(frame.totalPercent)}</small></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatSamples(value: number) {
  return value.toLocaleString();
}

function formatPercent(value: number | string) {
  return typeof value === "string" ? value : `${value.toFixed(1)}%`;
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
    selfPercent: row.self_percent,
    totalPercent: row.total_percent,
  };
}

function describeHotFrame(frame: HotFrame) {
  if (frame.total <= 0) return undefined;
  if (frame.self > 0 && frame.self / frame.total >= 0.5) {
    return `High self CPU: inspect ${frame.symbol}'s own work first.`;
  }
  if (frame.self === 0 || frame.self / frame.total < 0.5) {
    return `High total, low self: start from ${frame.symbol}, then inspect highlighted callees and runtime/native frames.`;
  }
  return `Mixed CPU cost: inspect both ${frame.symbol} and its callees.`;
}
