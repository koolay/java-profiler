import { Flamegraph } from "../../visualization/flamegraph";
import { getFlamegraph } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";

export function MemoryView({ params }: { params: URLSearchParams }) {
	const service = params.get("service") ?? "service";
	const fallback: FlamegraphResponse = { root: { name: service, value: 0, children: [] }, metadata: { partial: false } };
	const { data, error, loading } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	return (
		<section className="view-grid">
			<div className="view-copy">
				<h2>Allocation sources</h2>
				<p>Allocation profiles identify code paths creating objects. Retained heap ownership requires heap dump analysis outside v1.</p>
				{loading && <p className="muted">Loading profile evidence.</p>}
				{error && <p className="warning">Backend unavailable: {error}</p>}
			</div>
			<Flamegraph root={data?.root ?? fallback.root} metadata={data?.metadata} />
		</section>
	);
}
