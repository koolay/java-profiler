import { getFlamegraph, getTopStacks } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";
import { HotCodeView } from "./hot-code-view";

export function CpuView({ params }: { params: URLSearchParams }) {
	const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
	const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	const { data: topRows, error: topRowsError } = useAPI(() => getTopStacks(params), [params.toString()], []);
	const root = data?.root ?? fallback.root;
	return (
		<section>
			{error && <p className="warning">Backend unavailable: {error}</p>}
			{topRowsError && <p className="warning">Top table unavailable: {topRowsError}</p>}
			<HotCodeView root={root} metadata={data?.metadata} topRows={topRowsError ? undefined : topRows} />
		</section>
	);
}
