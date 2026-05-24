import { Flamegraph } from "../../visualization/flamegraph";
import { getFlamegraph } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse } from "../../api/types";

export function LocksView({ params }: { params: URLSearchParams }) {
	const fallback: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
	const { data, error } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	return (
		<section className="profile-analysis profile-analysis-wide" aria-label="Lock profile analysis">
			<div className="profile-toolbar profile-toolbar-tight">
				<div>
					<h2>Lock contention</h2>
					<p>Lock profiles show where Java threads spend blocked or waiting time under contention.</p>
				</div>
			</div>
			{error && <p className="warning">Backend unavailable: {error}</p>}
			<Flamegraph root={data?.root ?? fallback.root} metadata={data?.metadata} profileType="java_lock_delay_nanoseconds" />
		</section>
	);
}
