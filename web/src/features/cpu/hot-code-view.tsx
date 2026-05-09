import { useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../../api/types";
import { Flamegraph } from "../../visualization/flamegraph";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
};

type HotFrame = {
  name: string;
  symbol: string;
  fullSymbol: string;
  className: string;
  line?: number;
  self: number;
  total: number;
  selfPercent: number;
  totalPercent: number;
};

type ViewMode = "top-table" | "flame-graph" | "both";
type SortKey = "total" | "self" | "symbol";

export function HotCodeView({ root, metadata }: Props) {
  const hotFrames = useMemo(() => collectHotJavaFrames(root), [root]);
  const [selectedName, setSelectedName] = useState<string | undefined>();
  const [viewMode, setViewMode] = useState<ViewMode>("both");
  const [sortKey, setSortKey] = useState<SortKey>("total");
  const sortedFrames = useMemo(() => sortHotFrames(hotFrames, sortKey), [hotFrames, sortKey]);
  const selected = hotFrames.find((frame) => frame.name === selectedName) ?? sortedFrames[0];
  const highlightQuery = selected?.symbol ?? "";
  const insight = selected ? describeHotFrame(selected) : undefined;

  if (hotFrames.length === 0) {
    return (
      <section className="profile-analysis" aria-label="CPU profile analysis">
        <h2>CPU profile</h2>
        <p className="flamegraph-empty">No application Java frames were found in this CPU profile. Use the flame graph to inspect runtime or native frames.</p>
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
        {viewMode !== "flame-graph" && <TopTable frames={sortedFrames} selected={selected} sortKey={sortKey} onSort={setSortKey} onSelect={setSelectedName} />}
        {viewMode !== "top-table" && (
          <div className="profile-flamegraph">
            <Flamegraph root={root} metadata={metadata} highlightQuery={highlightQuery} insight={insight} />
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
      const key = `${parsed.className}.${parsed.method}`;
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
  if (
    normalizedName.includes(".so") ||
    normalizedName.includes("[vdso]") ||
    normalizedName.includes("pthread") ||
    normalizedName.includes("clock_gettime")
  ) {
    return false;
  }
  if (/^(read|write|open|close|poll|select|epoll|recv|send|accept|connect|syscall|nanosleep)$/.test(frame.method.toLowerCase())) return false;
  if (/^\d/.test(frame.method) || /^\d/.test(frame.className)) return false;
  if (!/^[A-Z_$]/.test(frame.className)) return false;
  const normalizedClass = frame.fullClassName.toLowerCase();
  const excludedPrefixes = [
    "java.",
    "javax.",
    "jdk.",
    "sun.",
    "com.sun.",
    "org.graalvm.",
    "lib",
  ];
  return !excludedPrefixes.some((prefix) => normalizedClass.startsWith(prefix));
}

function sortHotFrames(frames: HotFrame[], sortKey: SortKey) {
  return [...frames].sort((left, right) => {
    if (sortKey === "symbol") return left.symbol.localeCompare(right.symbol) || right.total - left.total;
    if (sortKey === "self") return right.self - left.self || right.total - left.total || left.symbol.localeCompare(right.symbol);
    return right.total - left.total || right.self - left.self || left.symbol.localeCompare(right.symbol);
  });
}

function TopTable({
  frames,
  selected,
  sortKey,
  onSort,
  onSelect,
}: {
  frames: HotFrame[];
  selected?: HotFrame;
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
            <tr key={frame.name} className={frame.name === selected?.name ? "active" : ""}>
              <td>
                <button onClick={() => onSelect(frame.name)}>
                  <span>{frame.symbol}</span>
                  <small>{frame.line ? `${frame.className}:${frame.line}` : frame.fullSymbol}</small>
                </button>
              </td>
              <td>{formatSamples(frame.self)} <small>{frame.selfPercent.toFixed(1)}%</small></td>
              <td>{formatSamples(frame.total)} <small>{frame.totalPercent.toFixed(1)}%</small></td>
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

function describeHotFrame(frame: HotFrame) {
  if (frame.total <= 0) return undefined;
  if (frame.self > 0 && frame.self / frame.total >= 0.5) {
    return `High self CPU: ${frame.symbol} directly consumes a meaningful share of samples. Start by inspecting this method's own work.`;
  }
  if (frame.self === 0 || frame.self / frame.total < 0.5) {
    return `High total, low Java self: start from ${frame.symbol}, then inspect the highlighted frames in the full stack. Runtime/native frames show where samples landed; this Java row is the nearest actionable owner.`;
  }
  return `Mixed CPU cost: inspect both ${frame.symbol} and its callees in the flame graph.`;
}
