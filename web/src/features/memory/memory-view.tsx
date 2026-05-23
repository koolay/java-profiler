import { Flamegraph } from "../../visualization/flamegraph";
import { getFlamegraph } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";

export function MemoryView({ params }: { params: URLSearchParams }) {
	const service = params.get("service") ?? "service";
	const fallback: FlamegraphResponse = { root: { name: service, value: 0, children: [] }, metadata: { partial: false } };
	const { data, error, loading } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	return (
		<section className="profile-analysis profile-analysis-wide" aria-label="Allocation profile analysis">
			<div className="profile-toolbar profile-toolbar-tight">
				<div>
					<h2>Allocation sources</h2>
					<p>Allocation profiles identify code paths creating objects. This view only has data when allocation profiling is enabled for the collector.</p>
				</div>
			</div>
			{loading && <p className="muted">Loading profile evidence.</p>}
			{error && <p className="warning">Backend unavailable: {error}</p>}
			<Flamegraph
				root={data?.root ?? fallback.root}
				metadata={data?.metadata}
				emptyMessage="No allocation samples returned. Allocation profiling is disabled by default in this environment because CPU-only profiling is the validated safe mode."
				showInspector
				showSelectedDetail={false}
				valueLabel="Allocated bytes"
				inspectorTotalLabel="Total Allocated"
				inspectorSelfLabel="Self Allocated"
				detailTotalPercentLabel="Total Allocated %"
				detailSelfPercentLabel="Self Allocated %"
			/>
		</section>
	);
}
