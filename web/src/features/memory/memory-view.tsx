import { Flamegraph } from "../../visualization/flamegraph";
import { getAllocationSummary, getFlamegraph } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { AllocationSummary, FlamegraphResponse } from "../../api/types";

export function MemoryView({ params }: { params: URLSearchParams }) {
	const service = params.get("service") ?? "service";
	const fallback: FlamegraphResponse = { root: { name: service, value: 0, children: [] }, metadata: { partial: false } };
	const { data, error, loading } = useAPI(() => getFlamegraph(params), [params.toString()], fallback);
	const summaryParams = new URLSearchParams(params);
	summaryParams.set("path_limit", "50");
	summaryParams.set("self_frame_limit", "50");
	const { data: summary, error: summaryError, loading: summaryLoading } = useAPI(() => getAllocationSummary(summaryParams), [summaryParams.toString()], emptyAllocationSummary(params));
	return (
		<section className="profile-analysis profile-analysis-wide" aria-label="Allocation profile analysis">
			<div className="profile-toolbar profile-toolbar-tight">
				<div>
					<h2>Allocation sources</h2>
					<p>Allocation profiles identify code paths creating objects. They show sampled allocation sources, not current retained heap ownership.</p>
				</div>
			</div>
			{loading && <p className="muted">Loading profile evidence.</p>}
			{error && <p className="warning">Backend unavailable: {error}</p>}
			{summaryLoading && <p className="muted">Loading allocation summary.</p>}
			{summaryError && <p className="warning">Allocation summary unavailable: {summaryError}. Showing flamegraph evidence only.</p>}
			{summary && !summaryError ? <AllocationSummaryPanel summary={summary} /> : null}
			<Flamegraph
				root={data?.root ?? fallback.root}
				metadata={data?.metadata}
				profileType="java_allocation_bytes"
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

function AllocationSummaryPanel({ summary }: { summary: AllocationSummary }) {
	const coverage = summary.coverage;
	const partialReasons = describePartialReasons(coverage.partial_reasons);
	const omittedEvidence = coverage.omitted_paths_lower_bound + coverage.omitted_nodes_lower_bound;
	return (
		<div className="allocation-summary" aria-label="Allocation summary">
			<div className="gc-summary-strip allocation-summary-strip">
				<div className="gc-summary-card gc-summary-card-primary">
					<span>Total allocated</span>
					<strong>{formatAllocationValue(coverage.total_value, coverage.value_unit)}</strong>
					<small>{coverage.has_data ? `${coverage.scanned_samples} samples` : emptyStateLabel(coverage.empty_state)}</small>
				</div>
				<div className="gc-summary-card">
					<span>Effective scope</span>
					<strong>{formatScope(summary.effective_scope)}</strong>
					<small>{formatRange(summary.effective_scope.start, summary.effective_scope.end)}</small>
				</div>
				<div className="gc-summary-card">
					<span>Returned evidence</span>
					<strong>{coverage.returned_paths} paths</strong>
					<small>{coverage.returned_self_frames} self frames</small>
				</div>
				<div className="gc-summary-card">
					<span>Data semantics</span>
					<strong>Sampled allocations</strong>
					<small>Not retained heap</small>
				</div>
			</div>
			{coverage.partial && (
				<p className="warning">
					Partial allocation summary: {partialReasons}. At least {omittedEvidence} paths, nodes, or self-frame rows may be missing from the returned evidence.
				</p>
			)}
			{!coverage.has_data && (
				<p className="warning">
					{emptyStateCopy(coverage.empty_state)}
				</p>
			)}
			{summary.insights.length > 0 && (
				<div className="allocation-insights" aria-label="Allocation insights">
					{summary.insights.map((insight) => (
						<p className="profile-insight" key={`${insight.category}-${insight.evidence_frame}`}>
							<strong>{categoryLabel(insight.category)}:</strong> {insightCopy(insight.message_code)} Evidence: {shortFrame(insight.evidence_frame)} allocated {formatAllocationValue(insight.evidence_value, coverage.value_unit)}.
						</p>
					))}
				</div>
			)}
			<div className="profile-grid allocation-grid">
				<AllocationTable
					ariaLabel="Top allocating paths"
					rows={summary.top_paths.map((path) => ({
						key: path.path.join(">"),
						name: shortFrame(path.leaf_frame),
						detail: path.path.join(" > "),
						category: path.category,
						value: path.total_value,
						percent: path.percent,
					}))}
					title="Top allocating paths"
					valueUnit={coverage.value_unit}
				/>
				<AllocationTable
					ariaLabel="Top self allocating frames"
					rows={summary.top_self_frames.map((frame) => ({
						key: frame.frame,
						name: shortFrame(frame.frame),
						detail: frame.frame,
						category: frame.category,
						value: frame.self_value,
						percent: frame.percent,
					}))}
					title="Top self allocating frames"
					valueUnit={coverage.value_unit}
				/>
			</div>
		</div>
	);
}

type AllocationTableRow = {
	key: string;
	name: string;
	detail: string;
	category: string;
	value: number;
	percent: number;
};

function AllocationTable({ ariaLabel, rows, title, valueUnit }: { ariaLabel: string; rows: AllocationTableRow[]; title: string; valueUnit: string }) {
	return (
		<div className="top-table-wrap" role="region" aria-label={ariaLabel}>
			<table className="top-table allocation-table">
				<thead>
					<tr>
						<th>{title}</th>
						<th>Allocated</th>
						<th>Share</th>
					</tr>
				</thead>
				<tbody>
					{rows.length === 0 ? (
						<tr>
							<td className="top-table-empty" colSpan={3}>No allocation rows returned for this scope.</td>
						</tr>
					) : (
						rows.map((row) => (
							<tr key={row.key}>
								<td>
									<span>{row.name}</span>
									<small title={row.detail}>{categoryLabel(row.category)} · {row.detail}</small>
								</td>
								<td>{formatAllocationValue(row.value, valueUnit)}</td>
								<td>{formatPercent(row.percent)}</td>
							</tr>
						))
					)}
				</tbody>
			</table>
		</div>
	);
}

function emptyAllocationSummary(params: URLSearchParams): AllocationSummary {
	const start = params.get("start") ?? "";
	const end = params.get("end") ?? "";
	return {
		schema_version: 1,
		requested_scope: { namespace: params.get("namespace") ?? "", service: params.get("service") ?? "", pod: params.get("pod") ?? "", container: "", jvm: "", start, end },
		effective_scope: { namespace: params.get("namespace") ?? "", service: params.get("service") ?? "", pod: params.get("pod") ?? "", container: "", jvm: "", start, end },
		coverage: {
			has_data: false,
			empty_state: "no_samples_in_range",
			profile_type: "java_allocation_bytes",
			total_value: 0,
			value_unit: "bytes",
			scanned_samples: 0,
			returned_paths: 0,
			returned_self_frames: 0,
			omitted_paths_lower_bound: 0,
			omitted_nodes_lower_bound: 0,
			partial: false,
		},
		top_paths: [],
		top_self_frames: [],
		insights: [],
		limitations: [],
	};
}

function formatAllocationValue(value: number, unit: string) {
	if (unit === "objects" || unit === "events") return new Intl.NumberFormat().format(value);
	if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MiB`;
	if (value >= 1024) return `${(value / 1024).toFixed(1)} KiB`;
	return `${new Intl.NumberFormat().format(value)} B`;
}

function formatPercent(value: number) {
	return `${value.toFixed(value >= 10 ? 1 : 2)}%`;
}

function formatScope(scope: AllocationSummary["effective_scope"]) {
	const service = scope.service || "all services";
	const pod = scope.pod || "all pods";
	return `${scope.namespace || "all namespaces"} / ${service} / ${pod}`;
}

function formatRange(start: string, end: string) {
	if (!start || !end) return "selected range";
	const durationMs = Date.parse(end) - Date.parse(start);
	if (!Number.isFinite(durationMs) || durationMs <= 0) return "selected range";
	const minutes = Math.round(durationMs / 60000);
	return `${minutes}m window`;
}

function shortFrame(frame: string) {
	return frame.split(/[/.]/).slice(-2).join(".").replace(/:\d+$/, (line) => line);
}

function categoryLabel(category: string) {
	return category.split("_").map((part) => part[0]?.toUpperCase() + part.slice(1)).join(" ");
}

function insightCopy(code: string) {
	if (code.includes("string_construction")) return "String construction is a dominant allocation source.";
	if (code.includes("thread_local_cleanup")) return "Thread-local cleanup is allocating in this window.";
	if (code.includes("database_query_building")) return "Database query construction is contributing allocation pressure.";
	if (code.includes("array_copy")) return "Array copying or growth is a visible allocation source.";
	return "This category is visible in the returned allocation samples.";
}

function emptyStateLabel(state?: string) {
	if (!state) return "no data";
	return state.replaceAll("_", " ");
}

function emptyStateCopy(state?: string) {
	if (state === "profiling_disabled") return "This target is visible, but allocation profiling is not enabled for it.";
	if (state === "no_matching_target") return "No Java profiling target matched this namespace, service, Pod, and time range.";
	if (state === "ingestion_gap") return "Collector or backend ingestion reported dropped or retryable profile data in this window.";
	if (state === "unsupported_runtime") return "The selected runtime cannot provide this allocation profile.";
	if (state === "query_error") return "The backend could not query allocation evidence for this scope.";
	if (state === "no_stack_frames") return "Allocation samples exist, but their stack frames were missing or unusable for this scope.";
	return "No allocation samples matched this scope and time range. Try a wider range or a more specific Java target.";
}

function describePartialReasons(reasons?: string[]) {
	if (!reasons || reasons.length === 0) return "result limits were reached";
	const labels: Record<string, string> = {
		path_limit: "top path limit reached",
		node_limit: "flamegraph node budget reached",
		self_frame_limit: "top self-frame limit reached; smaller self frames may be missing",
		no_stack_frames: "some sampled allocations had no usable stack frames",
	};
	return reasons.map((reason) => labels[reason] || reason.replaceAll("_", " ")).join(", ");
}
