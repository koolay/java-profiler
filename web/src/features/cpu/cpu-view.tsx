import { getFlamegraph, getTopStacks } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";
import { HotCodeView } from "./hot-code-view";

export function CpuView({ params }: { params: URLSearchParams }) {
	const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
	const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	const { data: topRows, error: topRowsError } = useAPI(() => getTopStacks(params), [params.toString()], []);
	const root = data?.root ?? fallback.root;
	const usableTopRows = !topRowsError && topRows && topRows.length > 0 ? topRows : undefined;
	const profileWindow = getProfileWindow(params);
	return (
		<section>
			{error && <p className="warning">Backend unavailable: {error}</p>}
			{topRowsError && <p className="warning">Top table unavailable: {topRowsError}</p>}
			<HotCodeView root={root} metadata={data?.metadata} topRows={usableTopRows} profileWindow={profileWindow} profileType="java_cpu_nanoseconds" title="Single Pod CPU profile" description="Top table ranks Java methods by CPU time. Values are rendered from nanoseconds into incident-readable time and average cores." valueLabel="CPU time" selfColumnLabel="Self CPU" totalColumnLabel="Total CPU" />
		</section>
	);
}

function getProfileWindow(params: URLSearchParams) {
	const start = Date.parse(params.get("start") ?? "");
	const end = Date.parse(params.get("end") ?? "");
	if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return undefined;
	return { start: new Date(start), end: new Date(end), durationMs: end - start };
}
