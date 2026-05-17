import { getFlamegraph, getTopStacks } from "../../api/client";
import type { FlamegraphResponse } from "../../api/types";
import { useAPI } from "../../api/use-api";
import { HotCodeView } from "../cpu/hot-code-view";

export function IOView({ params }: { params: URLSearchParams }) {
  const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
  const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
  const { data: topRows, error: topRowsError } = useAPI(() => getTopStacks(params), [params.toString()], []);
  return (
    <section>
      {error && <p className="warning">Backend unavailable: {error}</p>}
      {topRowsError && <p className="warning">Top table unavailable: {topRowsError}</p>}
      <HotCodeView
        root={data?.root ?? fallback.root}
        metadata={data?.metadata}
        topRows={!topRowsError && topRows && topRows.length > 0 ? topRows : undefined}
        profileWindow={profileWindow(params)}
        profileType="java_io_wait_nanoseconds"
        title="Single Pod I/O wait profile"
        description="I/O wait evidence highlights Java socket or file blocking paths when backend evidence can preserve stack ownership."
        valueLabel="I/O wait"
        selfColumnLabel="Self I/O"
        totalColumnLabel="Total I/O"
      />
    </section>
  );
}

function profileWindow(params: URLSearchParams) {
  const start = Date.parse(params.get("start") ?? "");
  const end = Date.parse(params.get("end") ?? "");
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return undefined;
  return { start: new Date(start), end: new Date(end), durationMs: end - start };
}
