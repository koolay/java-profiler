import { useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../api/types";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
};

type Frame = FlamegraphNode & {
  depth: number;
  path: string;
  left: number;
  width: number;
  matched: boolean;
};

export function Flamegraph({ root, metadata }: Props) {
  const [query, setQuery] = useState("");
  const [zoomPath, setZoomPath] = useState("root");
  const frames = useMemo(() => layout(root, zoomPath, query), [root, query, zoomPath]);
  const depth = Math.max(0, ...frames.map((frame) => frame.depth));
  return (
    <section className="flamegraph" aria-label="Flamegraph">
      <div className="flamegraph-tools">
        <input aria-label="Search flamegraph frames" placeholder="Search frame" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button onClick={() => setZoomPath("root")}>Reset</button>
      </div>
      {metadata?.partial && <p className="warning">Partial result: {(metadata.reasons ?? ["query budget"]).join(", ")}.</p>}
      <div className="flamegraph-stack" style={{ height: Math.max(1, depth + 1) * 28 }}>
        {frames.map((frame) => (
          <button
            key={frame.path}
            className={`flame-row${frame.matched ? " flame-row-match" : ""}${frame.width < 7 ? " flame-row-tiny" : ""}`}
            style={{
              left: `${frame.left}%`,
              top: frame.depth * 28,
              width: `${frame.width}%`,
            }}
            onClick={() => setZoomPath(frame.path)}
            title={`${frame.name}: ${frame.value}`}
          >
            <span className="flame-frame">{frame.name}</span>
            {frame.width >= 7 && <b className="flame-value">{frame.value.toLocaleString()}</b>}
          </button>
        ))}
      </div>
    </section>
  );
}

function layout(root: FlamegraphNode, zoomPath: string, query: string): Frame[] {
  const zoomRoot = findByPath(root, zoomPath) ?? root;
  const normalizedQuery = query.trim().toLowerCase();
  const frames: Frame[] = [];
  const visit = (node: FlamegraphNode, depth: number, path: string, left: number, width: number) => {
    frames.push({
      ...node,
      depth,
      path,
      left,
      width,
      matched: normalizedQuery.length > 0 && node.name.toLowerCase().includes(normalizedQuery),
    });
    const children = node.children ?? [];
    const total = children.reduce((sum, child) => sum + Math.max(0, child.value), 0) || Math.max(1, node.value);
    let offset = left;
    for (const [index, child] of children.entries()) {
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
