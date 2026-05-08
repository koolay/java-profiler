import { Flamegraph } from "../../visualization/flamegraph";
import { getFlamegraph } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";

export function LocksView({ params }: { params: URLSearchParams }) {
	const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
	const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	return (
		<section>
			{error && <p className="warning">Backend unavailable: {error}</p>}
			<Flamegraph root={data?.root ?? fallback.root} metadata={data?.metadata} />
		</section>
	);
}
