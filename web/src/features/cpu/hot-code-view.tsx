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
  className: string;
  line?: number;
  self: number;
  total: number;
  selfPercent: number;
  totalPercent: number;
};

type ViewMode = "top-table" | "flame-graph" | "both";

export function HotCodeView({ root, metadata }: Props) {
  const hotFrames = useMemo(() => collectHotJavaFrames(root), [root]);
  const [selectedName, setSelectedName] = useState<string | undefined>();
  const [viewMode, setViewMode] = useState<ViewMode>("both");
  const selected = hotFrames.find((frame) => frame.name === selectedName) ?? hotFrames[0];
  const stackQuery = selected?.symbol ?? "DemoHttpService";

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
        {viewMode !== "flame-graph" && <TopTable frames={hotFrames} selected={selected} onSelect={setSelectedName} />}
        {viewMode !== "top-table" && (
          <div className="profile-flamegraph">
            <Flamegraph key={stackQuery} root={root} metadata={metadata} initialQuery={stackQuery} />
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
      self,
      total,
      selfPercent: (self / totalSamples) * 100,
      totalPercent: (total / totalSamples) * 100,
    }))
    .sort((left, right) => right.self - left.self || right.total - left.total || left.name.localeCompare(right.name));
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

function TopTable({ frames, selected, onSelect }: { frames: HotFrame[]; selected?: HotFrame; onSelect: (name: string) => void }) {
  return (
    <div className="top-table-wrap" role="region" aria-label="Top table">
      <table className="top-table">
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Self</th>
            <th>Total</th>
          </tr>
        </thead>
        <tbody>
          {frames.slice(0, 20).map((frame) => (
            <tr key={frame.name} className={frame.name === selected?.name ? "active" : ""}>
              <td>
                <button onClick={() => onSelect(frame.name)}>
                  <span>{frame.symbol}</span>
                  <small>{frame.line ? `${frame.className}:${frame.line}` : frame.className}</small>
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
