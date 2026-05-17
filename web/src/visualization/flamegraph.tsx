import { useEffect, useMemo, useRef, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../api/types";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
  emptyMessage?: string;
  highlightQuery?: string;
  insight?: string;
  searchQuery?: string;
  onSearchQueryChange?: (query: string) => void;
  onReset?: () => void;
  formatValue?: (value: number) => string;
  valueLabel?: string;
};

type Frame = FlamegraphNode & {
  depth: number;
  path: string;
  left: number;
  width: number;
  self: number;
  totalPercent: number;
  selfPercent: number;
  matched: boolean;
  dimmed: boolean;
  category: FrameCategory;
};

type FrameCategory = "application" | "runtime" | "native";

type FocusPoint = {
  path: string;
  signature?: string;
};

type FocusState = FocusPoint & {
  history: FocusPoint[];
};

const rootFocus: FocusState = { path: "root", history: [] };

export function Flamegraph({ root, metadata, emptyMessage = "No profile samples returned for this service and time range.", highlightQuery = "", insight, searchQuery, onSearchQueryChange, onReset, formatValue = defaultFormatValue, valueLabel = "Samples" }: Props) {
  const [internalQuery, setInternalQuery] = useState("");
  const [focus, setFocus] = useState<FocusState>(rootFocus);
  const [selectedPath, setSelectedPath] = useState("root");
  const [hoveredPath, setHoveredPath] = useState<string | undefined>();
  const [hideSystemFrames, setHideSystemFrames] = useState(false);
  const hoverClearTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const query = searchQuery ?? internalQuery;
  const setQuery = (nextQuery: string) => {
    if (searchQuery === undefined) {
      setInternalQuery(nextQuery);
    }
    onSearchQueryChange?.(nextQuery);
  };
  const focusedSignature = focus.path === "root" ? undefined : framePathSignature(root, focus.path);
  const focusValid = focus.path === "root" || (focusedSignature !== undefined && focusedSignature === focus.signature);
  const zoomPath = focusValid ? focus.path : "root";
  const frames = useMemo(() => layout(root, zoomPath, query, highlightQuery), [root, highlightQuery, query, zoomPath]);
  const visibleFrames = useMemo(() => (hideSystemFrames ? frames.filter((frame) => frame.path === zoomPath || frame.category === "application") : frames), [frames, hideSystemFrames, zoomPath]);
  const depth = Math.max(0, ...visibleFrames.map((frame) => frame.depth));
  const rowHeight = 32;
  const queryActive = query.trim().length > 0;
  const highlightActive = highlightQuery.trim().length > 0;
  const zoomed = zoomPath !== "root";
  const selectedFrame = (highlightActive && selectedPath === "root" ? visibleFrames.find((frame) => frame.matched) : visibleFrames.find((frame) => frame.path === selectedPath)) ?? visibleFrames[0];
  const inspectedFrame = visibleFrames.find((frame) => frame.path === hoveredPath) ?? selectedFrame;
  const focusedFrame = zoomed ? visibleFrames.find((frame) => frame.path === zoomPath) : undefined;
  const focusedPercent = focusedFrame ? (Math.max(0, focusedFrame.value) / profileTotalValue(root)) * 100 : 0;
  const zoomTrail = zoomed ? ["root", ...focus.history.slice(1).map((entry) => entry.path), zoomPath].map((path) => findByPath(root, path)?.name ?? "root") : [];
  useEffect(() => {
    if (focusValid) return;
    setFocus(rootFocus);
    setSelectedPath("root");
    setHoveredPath(undefined);
  }, [focusValid]);
  useEffect(() => () => {
    if (hoverClearTimer.current) clearTimeout(hoverClearTimer.current);
  }, []);
  const cancelHoverClear = () => {
    if (hoverClearTimer.current) clearTimeout(hoverClearTimer.current);
    hoverClearTimer.current = undefined;
  };
  const inspectPath = (path: string) => {
    cancelHoverClear();
    setHoveredPath(path);
  };
  const clearInspectedPathSoon = () => {
    cancelHoverClear();
    hoverClearTimer.current = setTimeout(() => setHoveredPath(undefined), 120);
  };
  const resetZoom = () => {
    cancelHoverClear();
    setQuery("");
    setFocus(rootFocus);
    setSelectedPath("root");
    setHoveredPath(undefined);
    onReset?.();
  };
  const zoomToPath = (path: string) => {
    if (path === zoomPath) return;
    const signature = framePathSignature(root, path);
    if (!signature) return;
    const current: FocusPoint = { path: zoomPath, signature: zoomPath === "root" ? undefined : framePathSignature(root, zoomPath) };
    setFocus((state) => ({ path, signature, history: [...state.history, current] }));
    setSelectedPath(path);
    cancelHoverClear();
    setHoveredPath(undefined);
  };
  const backZoom = () => {
    const previous = focus.history.at(-1);
    if (!previous) return;
    const previousValid = previous.path === "root" || framePathSignature(root, previous.path) === previous.signature;
    const target = previousValid ? previous : { path: "root", signature: undefined };
    setFocus({ ...target, history: focus.history.slice(0, -1) });
    setSelectedPath(target.path);
    cancelHoverClear();
    setHoveredPath(undefined);
  };
  const hasSamples = (root.value > 0 || (root.children?.length ?? 0) > 0) && visibleFrames.length > 0;
  return (
    <section className="flamegraph" aria-label="Flamegraph">
      <div className="flamegraph-tools">
        <input aria-label="Search flamegraph frames" placeholder="Search frame" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button onClick={resetZoom}>Reset view</button>
        <button className={hideSystemFrames ? "active" : ""} aria-pressed={hideSystemFrames} onClick={() => setHideSystemFrames((value) => !value)}>Hide Native</button>
      </div>
      <p className="flamegraph-mode">
        {zoomed
            ? "Focused stack context. Width is relative to the focused block."
          : queryActive
            ? "Search highlights matching frames and keeps the sampled stack context visible."
            : "Full sampled stack context. Width shows total resource share under the current root; vertical position is stack hierarchy."}
      </p>
      {metadata?.partial && <p className="warning">Partial result: {(metadata.reasons ?? ["query budget"]).join(", ")}.</p>}
      {insight && <p className="profile-insight">{insight}</p>}
      {zoomed && focusedFrame && (
        <div className="focus-status" role="region" aria-label="Focused flamegraph state">
          <div>
            <span>Focused:</span>
            <code>{focusedFrame.name}</code>
            <strong>{displayFrameValue(focusedFrame, formatValue)} · {focusedPercent.toFixed(1)}% of profile</strong>
          </div>
          <div className="focus-status-actions">
            <button className="focus-status-back" onClick={backZoom} disabled={focus.history.length === 0} title="Return to the previous focused frame">Back</button>
            <button className="focus-status-reset" onClick={resetZoom} title="Return to the full root profile and clear search">Reset</button>
          </div>
        </div>
      )}
      <div className="flamegraph-legend" aria-label="Frame categories">
        <span><i className="legend-application" />Application Java</span>
        <span><i className="legend-runtime" />JVM/runtime</span>
        <span><i className="legend-native" />Native/system</span>
      </div>
      {zoomed && (
        <nav className="focus-breadcrumb" aria-label="Focused flamegraph path">
          <span>Focused</span>
          {zoomTrail.map((label, index) => (
            <code key={`${label}-${index}`}>{formatFrameLabel(label)}</code>
          ))}
        </nav>
      )}
      {hasSamples && inspectedFrame && selectedFrame && (
        <FrameInspector
          frame={inspectedFrame}
          onFocus={() => zoomToPath(inspectedFrame.path)}
          onMouseEnter={() => inspectPath(inspectedFrame.path)}
          onMouseLeave={clearInspectedPathSoon}
          formatValue={formatValue}
          valueLabel={valueLabel}
        />
      )}
      {hasSamples ? (
        <div className="flamegraph-stack" style={{ height: Math.max(1, depth + 1) * rowHeight }}>
          {visibleFrames.map((frame) => (
            <button
              key={frame.path}
              className={`flame-row flame-row-${frame.category}${frame.matched ? " flame-row-match" : ""}${frame.dimmed ? " flame-row-dimmed" : ""}${frame.path === selectedFrame?.path ? " flame-row-selected" : ""}${frame.width < 7 ? " flame-row-tiny" : ""}`}
              style={{
                left: `${frame.left}%`,
                top: frame.depth * rowHeight,
                width: `${frame.width}%`,
              }}
              onClick={() => {
                setSelectedPath(frame.path);
                inspectPath(frame.path);
              }}
              onFocus={() => inspectPath(frame.path)}
              onBlur={clearInspectedPathSoon}
              onMouseEnter={() => inspectPath(frame.path)}
              onMouseLeave={clearInspectedPathSoon}
              aria-describedby="flamegraph-frame-inspector"
              aria-label={`${formatFrameLabel(frame.name)} ${displayFrameValue(frame, formatValue)}`}
            >
              <span className="flame-frame">{formatFrameLabel(frame.name)}</span>
              {frame.width >= 7 && <b className="flame-value">{displayFrameValue(frame, formatValue)}</b>}
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
              <dt>{valueLabel}</dt>
              <dd>{displayFrameValue(selectedFrame, formatValue)}</dd>
            </div>
            <div>
              <dt>Self</dt>
              <dd>{formatValue(selectedFrame.self)}</dd>
            </div>
            <div>
              <dt>Total CPU</dt>
              <dd>{selectedFrame.totalPercent.toFixed(1)}%</dd>
            </div>
            <div>
              <dt>Self CPU</dt>
              <dd>{selectedFrame.selfPercent.toFixed(1)}%</dd>
            </div>
            <div>
              <dt>Depth</dt>
              <dd>{selectedFrame.depth}</dd>
            </div>
          </dl>
        </div>
      )}
    </section>
  );
}

function layout(root: FlamegraphNode, zoomPath: string, query: string, highlightQuery: string): Frame[] {
  const zoomRoot = findByPath(root, zoomPath) ?? root;
  const currentRootValue = profileTotalValue(zoomRoot);
  const normalizedQuery = normalizeFrameSearch(query);
  const normalizedHighlight = normalizeFrameSearch(highlightQuery);
  const activeMatch = normalizedQuery || normalizedHighlight;
  const matches = (node: FlamegraphNode) => activeMatch.length > 0 && normalizeFrameSearch(node.name).includes(activeMatch);
  const frames: Frame[] = [];
  const visit = (node: FlamegraphNode, depth: number, path: string, left: number, width: number) => {
    const matched = matches(node);
    const childTotal = (node.children ?? []).reduce((sum, child) => sum + Math.max(0, child.value), 0);
    const self = Math.max(0, Math.max(0, node.value) - childTotal);
    frames.push({
      ...node,
      depth,
      path,
      left,
      width,
      self,
      totalPercent: (Math.max(0, node.value) / currentRootValue) * 100,
      selfPercent: (self / currentRootValue) * 100,
      matched,
      dimmed: normalizedQuery.length > 0 && !matched,
      category: classifyFrame(node.name),
    });
    const children = (node.children ?? []).map((child, index) => ({ child, index }));
    const total = Math.max(Math.max(0, node.value), childTotal, 1);
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

function profileTotalValue(root: FlamegraphNode) {
  return Math.max(1, Math.max(0, root.value), (root.children ?? []).reduce((sum, child) => sum + Math.max(0, child.value), 0));
}

function normalizeFrameSearch(value: string) {
  return value.trim().replaceAll("/", ".").toLowerCase();
}

function defaultFormatValue(value: number) {
  return value.toLocaleString();
}

function displayFrameValue(frame: Frame, formatValue: (value: number) => string) {
  return frame.display_value ?? formatValue(frame.value);
}

function FrameInspector({ frame, onFocus, onMouseEnter, onMouseLeave, formatValue, valueLabel }: { frame: Frame; onFocus: () => void; onMouseEnter: () => void; onMouseLeave: () => void; formatValue: (value: number) => string; valueLabel: string }) {
  return (
    <div className={`flamegraph-tooltip flamegraph-tooltip-${frame.category}`} id="flamegraph-frame-inspector" role="status" onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
      <div>
        <span>{labelForCategory(frame.category)}</span>
        <strong>{frame.name}</strong>
      </div>
      <dl>
        <div>
          <dt>Total {valueLabel}</dt>
          <dd>{displayFrameValue(frame, formatValue)} <span>{frame.totalPercent.toFixed(1)}%</span></dd>
        </div>
        <div>
          <dt>Self CPU</dt>
          <dd>{formatValue(frame.self)} <span>{frame.selfPercent.toFixed(1)}%</span></dd>
        </div>
        <div>
          <dt>Depth</dt>
          <dd>{frame.depth}</dd>
        </div>
      </dl>
      <button type="button" onClick={onFocus}>Focus frame</button>
    </div>
  );
}

function labelForCategory(category: FrameCategory) {
  if (category === "native") return "Native/system";
  if (category === "runtime") return "JVM/runtime";
  return "Application Java";
}

function classifyFrame(name: string): FrameCategory {
  const normalized = name.replaceAll("/", ".").toLowerCase();
  if (
    /^so(\.|$)/.test(normalized) ||
    normalized.includes(".so") ||
    normalized.includes("[vdso]") ||
    normalized.includes("pthread") ||
    normalized.includes("clock_") ||
    normalized.includes("__") ||
    /(^|[./])(read|write|open|close|poll|select|epoll|recv|send|accept|connect)([:.]|$)/.test(normalized)
  ) {
    return "native";
  }
  if (
    normalized.startsWith("java.") ||
    normalized.startsWith("javax.") ||
    normalized.startsWith("jdk.") ||
    normalized.startsWith("sun.") ||
    normalized.startsWith("com.sun.") ||
    normalized.includes("c2i adapter") ||
    normalized.includes("i2c adapter") ||
    normalized.includes("stubroutine") ||
    normalized.includes("vtablestub") ||
    normalized.includes("thread.") ||
    normalized.includes("serverimpl") ||
    normalized.includes("filter$chain")
  ) {
    return "runtime";
  }
  return "application";
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

function framePathSignature(root: FlamegraphNode, path: string): string | undefined {
  const names: string[] = [];
  let current: FlamegraphNode | undefined = root;
  names.push(current.name);
  if (path !== "root") {
    for (const part of path.split("/").slice(1)) {
      const index = Number(part);
      if (!Number.isInteger(index)) return undefined;
      current = current?.children?.[index];
      if (!current) return undefined;
      names.push(current.name);
    }
  }
  return names.join("\u0000");
}

function formatFrameLabel(name: string) {
  const normalized = name.replaceAll("/", ".");
  if (normalized.includes(".so")) {
    return normalized.replace(/^(?:.*\/)?([^/.]+\.so(?:\.\d+)?)[.:]?(.*)$/, (_, library: string, symbol: string) => {
      const trimmed = String(symbol).replace(/^\./, "");
      return trimmed ? `${library} ${trimmed}` : library;
    });
  }
  const parts = normalized.split(".");
  if (parts.length <= 2) return normalized;
  return parts.slice(-2).join(".");
}
