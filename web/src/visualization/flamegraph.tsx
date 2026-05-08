import { useMemo, useState } from "react";
import type { FlamegraphNode, PartialMetadata } from "../api/types";

type Props = {
  root: FlamegraphNode;
  metadata?: PartialMetadata;
};

type Row = FlamegraphNode & { depth: number; path: string };

export function Flamegraph({ root, metadata }: Props) {
  const [query, setQuery] = useState("");
  const [zoomPath, setZoomPath] = useState("root");
  const rows = useMemo(() => flatten(root, zoomPath).filter((row) => row.name.toLowerCase().includes(query.toLowerCase())), [root, query, zoomPath]);
  const max = Math.max(1, ...rows.map((row) => row.value));
  return (
    <section className="flamegraph" aria-label="Flamegraph">
      <div className="flamegraph-tools">
        <input aria-label="Search flamegraph frames" placeholder="Search frame" value={query} onChange={(event) => setQuery(event.target.value)} />
        <button onClick={() => setZoomPath("root")}>Reset</button>
      </div>
      {metadata?.partial && <p className="warning">Partial result: {(metadata.reasons ?? ["query budget"]).join(", ")}.</p>}
      <div className="flamegraph-stack">
        {rows.map((row) => (
          <button
            key={row.path}
            className="flame-row"
            style={{ paddingLeft: 10 + row.depth * 14, width: `${Math.max(8, (row.value / max) * 100)}%` }}
            onClick={() => setZoomPath(row.path)}
            title={`${row.name}: ${row.value}`}
          >
            <span className="flame-frame">{row.name}</span>
            <b className="flame-value">{row.value.toLocaleString()}</b>
          </button>
        ))}
      </div>
    </section>
  );
}

function flatten(root: FlamegraphNode, zoomPath: string): Row[] {
  const rows: Row[] = [];
  const visit = (node: FlamegraphNode, depth: number, path: string) => {
    if (path.startsWith(zoomPath)) {
      rows.push({ ...node, depth, path });
      for (const child of node.children ?? []) {
        visit(child, depth + 1, `${path}/${child.name}`);
      }
      return;
    }
    for (const child of node.children ?? []) {
      visit(child, depth + 1, `${path}/${child.name}`);
    }
  };
  visit(root, 0, "root");
  return rows;
}
