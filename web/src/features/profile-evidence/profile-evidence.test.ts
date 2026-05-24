import { expect, test } from "vitest";
import type { IngestionHealth, TargetStatus } from "../../api/types";
import { classifyProfileEvidence } from "./profile-evidence";

const emptyHealth: IngestionHealth = {
  totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
  batches: [],
  partial: false,
};

function status(overrides: Partial<TargetStatus>): TargetStatus {
  return {
    target: { namespace: "prod", service: "checkout", pod: "checkout-1" },
    desired_state: "enabled",
    reason: "accepted",
    message: "",
    status_at: "2026-05-24T12:00:00Z",
    ...overrides,
  };
}

test("returns has_samples when the flamegraph has a non-zero root", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=checkout"),
    root: { name: "root", value: 1 },
    targetStatuses: [status({ desired_state: "disabled", reason: "disabled_by_metadata" })],
    ingestionHealth: emptyHealth,
  });

  expect(evidence.state).toBe("has_samples");
});

test("classifies disabled profiling from matching target status", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=checkout&pod=checkout-1"),
    root: { name: "root", value: 0 },
    targetStatuses: [status({ desired_state: "disabled", reason: "disabled_by_metadata", message: "profiling not enabled" })],
    ingestionHealth: emptyHealth,
  });

  expect(evidence).toMatchObject({ state: "profiling_disabled", statusReason: "disabled_by_metadata", message: "profiling not enabled" });
});

test("classifies expired temporary profiling before generic disabled state", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=checkout&pod=checkout-1"),
    root: { name: "root", value: 0 },
    targetStatuses: [status({ desired_state: "disabled", reason: "temporary_expired" })],
    ingestionHealth: emptyHealth,
  });

  expect(evidence.state).toBe("temporary_expired");
});

test("classifies no matching target for concrete scopes", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=missing"),
    root: { name: "root", value: 0 },
    targetStatuses: [status({})],
    ingestionHealth: emptyHealth,
  });

  expect(evidence.state).toBe("no_matching_target");
});

test("classifies aggregate ingestion gaps without claiming scope-specific loss", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=checkout"),
    root: { name: "root", value: 0 },
    targetStatuses: [],
    ingestionHealth: {
      totals: { accepted: 10, duplicate: 0, retryable: 1, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
      batches: [{ batch_type: "profile", status: "retryable", retryable: true, count: 1, latest_at: "2026-05-24T12:30:00Z" }],
      partial: false,
    },
  });

  expect(evidence.state).toBe("ingestion_gap");
  expect(evidence.message).toContain("not scoped to the selected service");
});

test("falls back to no samples when no stronger evidence exists", () => {
  const evidence = classifyProfileEvidence({
    params: new URLSearchParams("namespace=prod&service=checkout"),
    root: { name: "root", value: 0 },
    targetStatuses: [],
    ingestionHealth: emptyHealth,
  });

  expect(evidence.state).toBe("no_samples_in_range");
});
