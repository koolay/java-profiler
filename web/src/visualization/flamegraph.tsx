import { useEffect, useMemo, useRef, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../api/types";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
  emptyMessage?: string;
  profileType?: string;
  highlightQuery?: string;
  insight?: string;
  searchQuery?: string;
  onSearchQueryChange?: (query: string) => void;
  onReset?: () => void;
  formatValue?: (value: number) => string;
  valueLabel?: string;
  showInspector?: boolean;
  showSelectedDetail?: boolean;
  inspectorTotalLabel?: string;
  inspectorSelfLabel?: string;
  detailTotalPercentLabel?: string;
  detailSelfPercentLabel?: string;
  onFrameSelect?: (frameName: string) => void;
};

type Frame = FlamegraphNode & {
  depth: number;
  path: string;
  left: number;
  width: number;
  self: number;
  totalPercent: number;
  selfPercent: number;
  queryMatched: boolean;
  highlightMatched: boolean;
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

type ZoomTrailEntry = {
  path: string;
  label: string;
};

const rootFocus: FocusState = { path: "root", history: [] };

export function Flamegraph({ root, metadata, emptyMessage = "No profile samples returned for this service and time range.", profileType, highlightQuery = "", insight, searchQuery, onSearchQueryChange, onReset, formatValue = defaultFormatValue, valueLabel = "Samples", showInspector = true, showSelectedDetail = true, inspectorTotalLabel = "Total CPU", inspectorSelfLabel = "Self CPU", detailTotalPercentLabel = "Total CPU", detailSelfPercentLabel = "Self CPU", onFrameSelect }: Props) {
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
  const selectedFrame = (highlightActive && selectedPath === "root" ? visibleFrames.find((frame) => frame.highlightMatched) : visibleFrames.find((frame) => frame.path === selectedPath)) ?? visibleFrames[0];
  const inspectedFrame = visibleFrames.find((frame) => frame.path === hoveredPath) ?? selectedFrame;
  const focusedFrame = zoomed ? visibleFrames.find((frame) => frame.path === zoomPath) : undefined;
  const focusedPercent = focusedFrame ? (Math.max(0, focusedFrame.value) / profileTotalValue(root)) * 100 : 0;
  const zoomTrail: ZoomTrailEntry[] = zoomed
    ? [...focus.history.map((entry) => entry.path), zoomPath].map((path) => ({
        path,
        label: findByPath(root, path)?.name ?? "root",
      }))
    : [];
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
    setFocus((state) => {
      const existingIndex = state.history.findIndex((entry) => entry.path === path);
      const history = existingIndex >= 0 ? state.history.slice(0, existingIndex) : [...state.history, current];
      return { path, signature, history };
    });
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
      <section className={`flamegraph ${profileThemeClass(profileType)}`} aria-label="Flamegraph">
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
      {metadata?.partial && <p className="warning">{partialResultMessage(metadata?.reasons)}</p>}
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
          <span>Focused path</span>
          {zoomTrail.map((entry, index) => (
            <button
              key={`${entry.path}-${index}`}
              type="button"
              className={`focus-breadcrumb-chip${entry.path === zoomPath ? " focus-breadcrumb-current" : ""}`}
              onClick={() => zoomToPath(entry.path)}
              disabled={entry.path === zoomPath}
              aria-current={entry.path === zoomPath ? "page" : undefined}
              title={entry.path === zoomPath ? "Current focus" : `Zoom to ${formatFrameLabel(entry.label)}`}
            >
              {formatFrameLabel(entry.label)}
            </button>
          ))}
        </nav>
      )}
      {showInspector && hasSamples && inspectedFrame && selectedFrame && (
        <FrameInspector
          frame={inspectedFrame}
          onMouseEnter={() => inspectPath(inspectedFrame.path)}
          onMouseLeave={clearInspectedPathSoon}
          formatValue={formatValue}
          totalLabel={inspectorTotalLabel}
          selfLabel={inspectorSelfLabel}
        />
      )}
      {hasSamples ? (
        <div className="flamegraph-stack" style={{ height: Math.max(1, depth + 1) * rowHeight }}>
          {visibleFrames.map((frame) => {
            const frameDetail = formatFrameDetail(frame, formatValue, inspectorTotalLabel, inspectorSelfLabel);
            return (
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
                  zoomToPath(frame.path);
                  inspectPath(frame.path);
                  onFrameSelect?.(frame.name);
                }}
                onFocus={() => inspectPath(frame.path)}
                onBlur={clearInspectedPathSoon}
                onMouseEnter={() => inspectPath(frame.path)}
                onMouseLeave={clearInspectedPathSoon}
                title={frameDetail.title}
                aria-describedby={showInspector ? "flamegraph-frame-inspector" : undefined}
                aria-label={frameDetail.ariaLabel}
              >
                <span className="flame-frame">{formatFrameLabel(frame.name)}</span>
                {frame.width >= 7 && <b className="flame-value">{frameDetail.totalValue}</b>}
              </button>
            );
          })}
        </div>
      ) : (
        <p className="flamegraph-empty">{query.trim() ? `No frames match "${query.trim()}".` : emptyMessage}</p>
      )}
      {showSelectedDetail && hasSamples && selectedFrame && (
        <div className="flamegraph-detail" role="region" aria-label="Selected flamegraph frame">
          {(() => {
            const frameDetail = formatFrameDetail(selectedFrame, formatValue, valueLabel, detailSelfPercentLabel);
            return (
              <>
          <div>
            <span>Selected frame</span>
            <code>{selectedFrame.name}</code>
          </div>
          <p className="frame-diagnosis">{frameDetail.diagnosis}</p>
          <dl>
            <div>
              <dt>{valueLabel}</dt>
              <dd>{frameDetail.totalValue}</dd>
            </div>
            <div>
              <dt>{detailTotalPercentLabel}</dt>
              <dd>{frameDetail.totalPercent}</dd>
            </div>
            <div>
              <dt>{detailSelfPercentLabel}</dt>
              <dd>{frameDetail.selfPercent}</dd>
            </div>
            <div>
              <dt>Depth</dt>
              <dd>{selectedFrame.depth}</dd>
            </div>
          </dl>
              </>
            );
          })()}
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
  const queryMatches = (node: FlamegraphNode) => normalizedQuery.length > 0 && normalizeFrameSearch(node.name).includes(normalizedQuery);
  const highlightMatches = (node: FlamegraphNode) => normalizedHighlight.length > 0 && normalizeFrameSearch(node.name).includes(normalizedHighlight);
  const frames: Frame[] = [];
  const visit = (node: FlamegraphNode, depth: number, path: string, left: number, width: number) => {
    const queryMatched = queryMatches(node);
    const highlightMatched = highlightMatches(node);
    const matched = queryMatched || highlightMatched;
    const childTotal = (node.children ?? []).reduce((sum, child) => sum + Math.max(0, child.value), 0);
    const self = Math.max(0, Math.max(0, node.value) - childTotal);
    const childLayoutTotal = depth === 0 ? Math.max(childTotal, 1) : Math.max(Math.max(0, node.value), childTotal, 1);
    frames.push({
      ...node,
      depth,
      path,
      left,
      width,
      self,
      totalPercent: (Math.max(0, node.value) / currentRootValue) * 100,
      selfPercent: (self / currentRootValue) * 100,
      queryMatched,
      highlightMatched,
      matched,
      dimmed: normalizedQuery.length > 0 && !queryMatched && !highlightMatched,
      category: classifyFrame(node.name),
    });
    const children = (node.children ?? []).map((child, index) => ({ child, index }));
    let offset = left;
    for (const { child, index } of children) {
      const childWidth = width * (Math.max(0, child.value) / childLayoutTotal);
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

function formatFrameDetail(frame: Frame, formatValue: (value: number) => string, totalLabel: string, selfLabel: string) {
  const category = labelForCategory(frame.category);
  const totalValue = displayFrameValue(frame, formatValue);
  const selfValue = formatValue(frame.self);
  const totalPercent = `${frame.totalPercent.toFixed(1)}%`;
  const selfPercent = `${frame.selfPercent.toFixed(1)}%`;
  const diagnosis = frameDiagnosis(frame);
  const title = [
    frame.name,
    `Category: ${category}`,
    `${totalLabel}: ${totalValue} (${totalPercent})`,
    `${selfLabel}: ${selfValue} (${selfPercent})`,
    `Depth: ${frame.depth}`,
  ].join("\n");
  return {
    ariaLabel: `${frame.name}. ${category}. ${totalLabel} ${totalValue}, ${totalPercent}. ${selfLabel} ${selfValue}, ${selfPercent}. Depth ${frame.depth}.`,
    title,
    category,
    totalValue,
    selfValue,
    totalPercent,
    selfPercent,
    diagnosis,
  };
}

function frameDiagnosis(frame: Frame) {
  if (frame.category !== "application") {
    return "Find the owning Java caller before treating this frame as the optimization target.";
  }
  if (frame.totalPercent > 0 && frame.selfPercent >= frame.totalPercent * 0.5) {
    return "High self means this frame's own work is the first optimization target.";
  }
  if (frame.totalPercent >= 1 && frame.selfPercent < frame.totalPercent * 0.25) {
    return "High total with low self means inspect callees under this frame.";
  }
  return "Inspect callers and callees before choosing the optimization target.";
}

function partialResultMessage(reasons?: string[]) {
  const reasonList = reasons ?? ["query budget"];
  if (reasonList.includes("node_limit")) {
    return "Showing a partial flamegraph because the frame budget was reached.";
  }
  if (reasonList.includes("query budget")) {
    return "Showing a partial flamegraph because the query budget was reached.";
  }
  return `Showing a partial flamegraph because ${reasonList.join(", ")}.`;
}

function FrameInspector({ frame, onMouseEnter, onMouseLeave, formatValue, totalLabel, selfLabel }: { frame: Frame; onMouseEnter: () => void; onMouseLeave: () => void; formatValue: (value: number) => string; totalLabel: string; selfLabel: string }) {
  const copyText = frame.name;
  const frameDetail = formatFrameDetail(frame, formatValue, totalLabel, selfLabel);
  const copyFrame = () => {
    if (navigator.clipboard) {
      void navigator.clipboard.writeText(copyText);
    }
  };
  const copyPermalink = () => {
    if (navigator.clipboard) {
      const url = new URL(window.location.href);
      url.searchParams.set("frame", copyText);
      void navigator.clipboard.writeText(url.toString());
    }
  };
  return (
    <div className={`flamegraph-tooltip flamegraph-tooltip-${frame.category}`} id="flamegraph-frame-inspector" role="status" onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
      <div>
        <span>{frameDetail.category}</span>
        <strong>{frame.name}</strong>
        <p className="frame-diagnosis">{frameDetail.diagnosis}</p>
      </div>
      <dl>
        <div>
          <dt>{totalLabel}</dt>
          <dd>{frameDetail.totalValue} <span>{frameDetail.totalPercent}</span></dd>
        </div>
        <div>
          <dt>{selfLabel}</dt>
          <dd>{frameDetail.selfValue} <span>{frameDetail.selfPercent}</span></dd>
        </div>
        <div>
          <dt>Depth</dt>
          <dd>{frame.depth}</dd>
        </div>
      </dl>
      <div className="flamegraph-tooltip-actions">
        <button type="button" onClick={copyFrame}>Copy frame</button>
        <button type="button" onClick={copyPermalink}>Permalink</button>
      </div>
    </div>
  );
}

function labelForCategory(category: FrameCategory) {
  if (category === "native") return "Native/system";
  if (category === "runtime") return "JVM/runtime";
  return "Application Java";
}

function profileThemeClass(profileType?: string) {
  if (profileType === "java_allocation_bytes") return "flamegraph-profile-allocation";
  if (profileType === "java_wall_clock_nanoseconds") return "flamegraph-profile-wall";
  if (profileType === "java_io_wait_nanoseconds") return "flamegraph-profile-io";
  if (profileType === "java_lock_delay_nanoseconds") return "flamegraph-profile-lock";
  return "flamegraph-profile-cpu";
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
