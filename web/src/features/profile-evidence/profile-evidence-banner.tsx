import { getIngestionHealth, getTargetStatus } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphNode, IngestionHealth, TargetStatus } from "../../api/types";
import { classifyProfileEvidence, type ProfileEvidence } from "./profile-evidence";

const emptyStatuses: TargetStatus[] = [];
const emptyIngestion: IngestionHealth = {
  totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
  batches: [],
  partial: false,
};

export function useProfileEvidence(params: URLSearchParams, root?: FlamegraphNode) {
  const statusParams = new URLSearchParams(params);
  statusParams.delete("profile_type");
  const statuses = useAPI(() => getTargetStatus(statusParams), [statusParams.toString()], emptyStatuses);
  const ingestion = useAPI(() => getIngestionHealth(), [params.toString()], emptyIngestion);
  return classifyProfileEvidence({
    params,
    root,
    targetStatuses: statuses.data,
    targetStatusError: statuses.error,
    ingestionHealth: ingestion.data,
    ingestionError: ingestion.error,
  });
}

export function ProfileEvidenceBanner({ evidence }: { evidence: ProfileEvidence }) {
  if (evidence.state === "has_samples") return null;
  return (
    <div className="warning" role="status" aria-label="Profile evidence status">
      {evidence.message}
      {evidence.latestProfileBatchAt ? ` Latest profile ingestion: ${formatEvidenceTime(evidence.latestProfileBatchAt)}.` : ""}
      {evidence.latestStatusAt ? ` Latest target status: ${formatEvidenceTime(evidence.latestStatusAt)}${evidence.statusReason ? ` (${evidence.statusReason})` : ""}.` : ""}
    </div>
  );
}

function formatEvidenceTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toISOString().replace(".000Z", "Z");
}
