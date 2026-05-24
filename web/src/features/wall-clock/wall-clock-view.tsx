import { getFlamegraph, getTopStacks } from "../../api/client";
import type { FlamegraphResponse } from "../../api/types";
import { useAPI } from "../../api/use-api";
import { HotCodeView } from "../cpu/hot-code-view";
import { ProfileEvidenceBanner, useProfileEvidence } from "../profile-evidence/profile-evidence-banner";

export function WallClockView({ params }: { params: URLSearchParams }) {
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
        profileType="java_wall_clock_nanoseconds"
        analysisLabel="Wall Clock profile analysis"
        title="Single Pod Wall Clock profile"
        description="Wall Clock shows runnable, blocked, waiting, or sleeping time. Totals accumulate across sampled stacks, so they can exceed the selected window."
        valueLabel="Wall time"
        selfColumnLabel="Self Wall"
        totalColumnLabel="Total Wall"
        flamegraphEmptyMessage="No wall-clock flamegraph samples returned for this service and time range. The top table may still surface correlated Java methods."
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
