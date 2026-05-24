import type { FlamegraphNode, IngestionHealth, TargetStatus } from "../../api/types";

export type ProfileEvidenceState =
  | "has_samples"
  | "profiling_disabled"
  | "temporary_expired"
  | "no_matching_target"
  | "ingestion_gap"
  | "query_error"
  | "no_samples_in_range";

export type ProfileEvidence = {
  state: ProfileEvidenceState;
  message: string;
  latestProfileBatchAt?: string;
  latestStatusAt?: string;
  statusReason?: string;
};

export type ProfileEvidenceInput = {
  params: URLSearchParams;
  root?: FlamegraphNode;
  targetStatuses?: TargetStatus[] | null;
  targetStatusError?: string | null;
  ingestionHealth?: IngestionHealth | null;
  ingestionError?: string | null;
};

export function classifyProfileEvidence({ params, root, targetStatuses, targetStatusError, ingestionHealth, ingestionError }: ProfileEvidenceInput): ProfileEvidence {
  if ((root?.value ?? 0) > 0) {
    return { state: "has_samples", message: "Profile samples are available for this selection." };
  }
  if (targetStatusError || ingestionError) {
    return { state: "query_error", message: "The backend could not load all status evidence for this profile window." };
  }

  const matchedStatuses = (targetStatuses ?? []).filter((status) => matchesScope(params, status));
  const latestStatus = newestStatus(matchedStatuses);
  if (latestStatus?.reason === "temporary_expired") {
    return {
      state: "temporary_expired",
      message: "No profile samples were found because temporary profiling has expired for this target.",
      latestStatusAt: latestStatus.status_at,
      statusReason: latestStatus.reason,
    };
  }
  if (latestStatus?.reason === "disabled_by_metadata" || latestStatus?.desired_state === "disabled") {
    return {
      state: "profiling_disabled",
      message: latestStatus.message || "No profile samples were found because profiling is disabled for this target.",
      latestStatusAt: latestStatus.status_at,
      statusReason: latestStatus.reason,
    };
  }
  if ((targetStatuses ?? []).length > 0 && matchedStatuses.length === 0 && hasConcreteScope(params)) {
    return {
      state: "no_matching_target",
      message: "No Java profiling target matched this namespace, service, Pod, and time range.",
    };
  }

  const profileBatch = latestProfileBatch(ingestionHealth);
  if (profileBatch && hasIngestionProblem(ingestionHealth)) {
    return {
      state: "ingestion_gap",
      message: "Recent aggregate ingestion health shows rejected, retryable, dropped, or truncated profile evidence. This may explain missing samples, but it is not scoped to the selected service.",
      latestProfileBatchAt: profileBatch.latest_at,
    };
  }

  return {
    state: "no_samples_in_range",
    message: "No profile samples matched this scope and time range.",
    latestProfileBatchAt: profileBatch?.latest_at,
    latestStatusAt: latestStatus?.status_at,
    statusReason: latestStatus?.reason,
  };
}

function matchesScope(params: URLSearchParams, status: TargetStatus) {
  const target = status.target ?? {};
  return matchesParam(params, "namespace", target.namespace) && matchesParam(params, "service", target.service) && matchesParam(params, "pod", target.pod);
}

function matchesParam(params: URLSearchParams, key: string, targetValue?: string) {
  const selected = normalizeScope(params.get(key));
  if (!selected) return true;
  return selected === normalizeScope(targetValue ?? "");
}

function normalizeScope(value: string | null) {
  const normalized = (value ?? "").trim();
  return normalized === "all" ? "" : normalized;
}

function hasConcreteScope(params: URLSearchParams) {
  return Boolean(normalizeScope(params.get("namespace")) || normalizeScope(params.get("service")) || normalizeScope(params.get("pod")));
}

function newestStatus(statuses: TargetStatus[]) {
  return [...statuses].sort((a, b) => Date.parse(b.status_at ?? "") - Date.parse(a.status_at ?? ""))[0];
}

function latestProfileBatch(health?: IngestionHealth | null) {
  return health?.batches
    .filter((batch) => batch.batch_type === "profile")
    .sort((a, b) => Date.parse(b.latest_at) - Date.parse(a.latest_at))[0];
}

function hasIngestionProblem(health?: IngestionHealth | null) {
  if (!health) return false;
  return health.totals.retryable > 0 || health.totals.rejected > 0 || health.totals.dropped_samples > 0 || health.totals.dropped_stacks > 0 || health.totals.truncated_batches > 0;
}
