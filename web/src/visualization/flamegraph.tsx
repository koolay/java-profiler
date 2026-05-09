import { useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../api/types";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
  emptyMessage?: string;
  highlightQuery?: string;
};

type Frame = FlamegraphNode & {
  depth: number;
  path: string;
  left: number;
  width: number;
  matched: boolean;
  dimmed: boolean;
};

export function Flamegraph({ root, metadata, emptyMessage = "No profile samples returned for this service and time range.", highlightQuery = "" }: Props) {
  const [query, setQuery] = useState("");
  const [zoomPath, setZoomPath] = useState("root");
  const [selectedPath, setSelectedPath] = useState("root");
  const [zoomHistory, setZoomHistory] = useState<string[]>([]);
  const frames = useMemo(() => layout(root, zoomPath, query, highlightQuery), [root, highlightQuery, query, zoomPath]);
  const depth = Math.max(0, ...frames.map((frame) => frame.depth));
  const rowHeight = 32;
  const queryActive = query.trim().length > 0;
  const highlightActive = highlightQuery.trim().length > 0;
  const zoomed = zoomPath !== "root";
  const selectedFrame = ((queryActive || highlightActive) && selectedPath === "root" ? frames.find((frame) => frame.matched) : frames.find((frame) => frame.path === selectedPath)) ?? frames[0];
  const currentRootValue = Math.max(1, frames[0]?.value ?? 1);
  const selectedPercent = selectedFrame ? (selectedFrame.value / currentRootValue) * 100 : 0;
  const resetZoom = () => {
    setQuery("");
    setZoomPath("root");
    setSelectedPath("root");
    setZoomHistory([]);
  };
  const zoomToPath = (path: string) => {
    if (path === zoomPath) return;
    setZoomHistory((history) => [...history, zoomPath]);
    setZoomPath(path);
    setSelectedPath(path);
  };
  const zoomToSelected = () => {
    if (selectedFrame) {
      zoomToPath(selectedFrame.path);
    }
  };
  const backZoom = () => {
    setZoomHistory((history) => {
      const previous = history.at(-1);
      if (!previous) return history;
      setZoomPath(previous);
      setSelectedPath(previous);
      return history.slice(0, -1);
    });
  };
  const hasSamples = (root.value > 0 || (root.children?.length ?? 0) > 0) && frames.length > 0;
  return (
    <section className="flamegraph" aria-label="Flamegraph">
      <div className="flamegraph-tools">
        <input aria-label="Search flamegraph frames" placeholder="Search frame" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button onClick={backZoom} disabled={zoomHistory.length === 0}>Back</button>
        <button onClick={resetZoom}>Reset</button>
      </div>
      <p className="flamegraph-mode">
        {zoomed
            ? "Focused stack context. Width is relative to the focused block."
          : queryActive
            ? "Search highlights matching frames and keeps the sampled stack context visible."
            : "Full sampled stack context. Width shows total resource share under the current root; vertical position is stack hierarchy."}
      </p>
      {metadata?.partial && <p className="warning">Partial result: {(metadata.reasons ?? ["query budget"]).join(", ")}.</p>}
      {hasSamples ? (
        <div className="flamegraph-stack" style={{ height: Math.max(1, depth + 1) * rowHeight }}>
          {frames.map((frame) => (
            <button
              key={frame.path}
              className={`flame-row${frame.matched ? " flame-row-match" : ""}${frame.dimmed ? " flame-row-dimmed" : ""}${frame.path === selectedFrame?.path ? " flame-row-selected" : ""}${frame.width < 7 ? " flame-row-tiny" : ""}`}
              style={{
                left: `${frame.left}%`,
                top: frame.depth * rowHeight,
                width: `${frame.width}%`,
              }}
              onClick={() => setSelectedPath(frame.path)}
              title={`${frame.name}: ${frame.value}`}
            >
              <span className="flame-frame">{formatFrameLabel(frame.name)}</span>
              {frame.width >= 7 && <b className="flame-value">{frame.value.toLocaleString()}</b>}
            </button>
          ))}
        </div>
      ) : (
        <p className="flamegraph-empty">{query.trim() ? `No frames match "${query.trim()}".` : emptyMessage}</p>
      )}
      {hasSamples && selectedFrame && (
        <div className="flamegraph-detail" role="region" aria-label="Selected flamegraph frame">
          <div>
            <span>Selected frame</span>
            <code>{selectedFrame.name}</code>
          </div>
          <dl>
            <div>
              <dt>Samples</dt>
              <dd>{selectedFrame.value.toLocaleString()}</dd>
            </div>
            <div>
              <dt>Current root</dt>
              <dd>{selectedPercent.toFixed(1)}%</dd>
            </div>
            <div>
              <dt>Depth</dt>
              <dd>{selectedFrame.depth}</dd>
            </div>
          </dl>
          <button onClick={zoomToSelected}>Focus selected</button>
        </div>
      )}
    </section>
  );
}

function layout(root: FlamegraphNode, zoomPath: string, query: string, highlightQuery: string): Frame[] {
  const zoomRoot = findByPath(root, zoomPath) ?? root;
  const normalizedQuery = query.trim().toLowerCase();
  const normalizedHighlight = highlightQuery.trim().toLowerCase();
  const activeMatch = normalizedQuery || normalizedHighlight;
  const matches = (node: FlamegraphNode) => activeMatch.length > 0 && node.name.toLowerCase().includes(activeMatch);
  const frames: Frame[] = [];
  const visit = (node: FlamegraphNode, depth: number, path: string, left: number, width: number) => {
    const matched = matches(node);
    frames.push({
      ...node,
      depth,
      path,
      left,
      width,
      matched,
      dimmed: activeMatch.length > 0 && !matched,
    });
    const children = (node.children ?? []).map((child, index) => ({ child, index }));
    const total = children.reduce((sum, { child }) => sum + Math.max(0, child.value), 0) || Math.max(1, node.value);
    let offset = left;
    for (const { child, index } of children) {
      const childWidth = width * (Math.max(0, child.value) / total);
      visit(child, depth + 1, `${path}/${index}`, offset, childWidth);
      offset += childWidth;
    }
  };
  visit(zoomRoot, 0, zoomPath, 0, 100);
  return frames;
}

function findByPath(root: FlamegraphNode, path: string): FlamegraphNode | undefined {
  if (path === "root") return root;
  const parts = path.split("/").slice(1);
  let current: FlamegraphNode | undefined = root;
  for (const part of parts) {
    const index = Number(part);
    if (!Number.isInteger(index)) return undefined;
    current = current?.children?.[index];
  }
  return current;
}

function formatFrameLabel(name: string) {
  const normalized = name.replaceAll("/", ".");
  const parts = normalized.split(".");
  if (parts.length <= 2) return normalized;
  return parts.slice(-2).join(".");
}
