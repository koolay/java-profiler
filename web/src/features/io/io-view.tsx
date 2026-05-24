import { getFlamegraph, getTopStacks } from "../../api/client";
import type { FlamegraphResponse } from "../../api/types";
import { useAPI } from "../../api/use-api";
import { HotCodeView } from "../cpu/hot-code-view";
import { ProfileEvidenceBanner, useProfileEvidence } from "../profile-evidence/profile-evidence-banner";

export function IOView({ params }: { params: URLSearchParams }) {
  const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
  const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
  const { data: topRows, error: topRowsError } = useAPI(() => getTopStacks(params), [params.toString()], []);
  const root = data?.root ?? fallback.root;
  const evidence = useProfileEvidence(params, root);
  return (
    <section>
      {error && <p className="warning">Backend unavailable: {error}</p>}
      {topRowsError && <p className="warning">Top table unavailable: {topRowsError}</p>}
      <ProfileEvidenceBanner evidence={evidence} />
      <HotCodeView
        root={root}
        metadata={data?.metadata}
        topRows={!topRowsError && topRows && topRows.length > 0 ? topRows : undefined}
        profileWindow={profileWindow(params)}
        profileType="java_io_wait_nanoseconds"
        analysisLabel="I/O wait profile analysis"
        title="Single Pod I/O wait profile"
        description="I/O wait evidence highlights socket or file blocking paths. If the flamegraph is empty, the top table may still show correlated Java methods."
        valueLabel="I/O wait"
        selfColumnLabel="Self I/O"
        totalColumnLabel="Total I/O"
        flamegraphEmptyMessage="No I/O flamegraph samples returned for this service and time range. Check Target status to confirm the target is enabled before inspecting the flamegraph."
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
