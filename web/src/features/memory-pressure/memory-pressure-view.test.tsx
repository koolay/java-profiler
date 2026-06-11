import { render, screen } from "@testing-library/react";
import { beforeEach, test, vi } from "vitest";
import { getAllocationSummary, getFlamegraph, getIngestionHealth, getJVMEvents, getTargetStatus } from "../../api/client";
import { MemoryPressureView } from "./memory-pressure-view";

vi.mock("../../api/client", () => ({
  getAllocationSummary: vi.fn(),
  getFlamegraph: vi.fn(),
  getIngestionHealth: vi.fn(),
  getJVMEvents: vi.fn(),
  getTargetStatus: vi.fn(),
}));

const params = new URLSearchParams("namespace=prod&service=checkout&pod=checkout-1&start=2026-06-11T00:00:00Z&end=2026-06-11T00:15:00Z");

beforeEach(() => {
  vi.mocked(getTargetStatus).mockResolvedValue([
    {
      target: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "app", process_id: 41 },
      status_at: "2026-06-11T00:14:30Z",
      desired_state: "unsupported",
      reason: "unsupported_jvm",
      message: "unsupported_jvm",
    },
    {
      target: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "app", process_id: 42 },
      status_at: "2026-06-11T00:14:30Z",
      desired_state: "enabled",
      reason: "accepted",
      message: "profiling accepted",
    },
  ]);
  vi.mocked(getIngestionHealth).mockResolvedValue({
    totals: { accepted: 5, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
    batches: [{ batch_type: "profile", status: "accepted", retryable: false, count: 5, latest_at: "2026-06-11T00:14:45Z" }],
    partial: false,
  });
  vi.mocked(getJVMEvents).mockResolvedValue({
    events: [{ event_id: "gc-1", event_type: "gc_pause", event_at: "2026-06-11T00:12:00Z", duration_ns: 150_000_000 }],
    partial: false,
  });
  vi.mocked(getAllocationSummary).mockResolvedValue({
    schema_version: 1,
    requested_scope: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "", jvm: "", start: "2026-06-11T00:00:00Z", end: "2026-06-11T00:15:00Z" },
    effective_scope: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "", jvm: "", start: "2026-06-11T00:00:00Z", end: "2026-06-11T00:15:00Z" },
    coverage: {
      has_data: true,
      profile_type: "java_allocation_bytes",
      total_value: 4 * 1024 * 1024,
      value_unit: "bytes",
      scanned_samples: 16,
      returned_paths: 1,
      returned_self_frames: 1,
      omitted_paths_lower_bound: 0,
      omitted_nodes_lower_bound: 0,
      partial: false,
    },
    top_paths: [{ rank: 1, leaf_frame: "com.example.CartBuilder.build", total_value: 4 * 1024 * 1024, self_value: 2048, percent: 70, category: "application", sample_count: 12, path: ["root", "com.example.CartBuilder.build"] }],
    top_self_frames: [{ rank: 1, frame: "com.example.CartBuilder.build", self_value: 2 * 1024 * 1024, percent: 50, category: "application" }],
    insights: [],
    limitations: [],
  });
  vi.mocked(getFlamegraph).mockResolvedValue({
    root: { name: "checkout", value: 4 * 1024 * 1024, children: [{ name: "com.example.CartBuilder.build", value: 4 * 1024 * 1024 }] },
    metadata: { partial: false },
  });
});

test("renders composed memory pressure evidence from existing query APIs", async () => {
  render(<MemoryPressureView params={params} />);

  expect(await screen.findByRole("region", { name: "Memory pressure investigation" })).toBeInTheDocument();
  expect(screen.getByText("Profiling accepted")).toBeInTheDocument();
  expect(screen.getByText("5 accepted")).toBeInTheDocument();
  expect(screen.getByText("1 GC pause")).toBeInTheDocument();
  expect(screen.getAllByText("4.0 MiB").length).toBeGreaterThan(0);
  expect(screen.queryByText("Loading memory pressure evidence.")).not.toBeInTheDocument();
  expect(screen.getByText(/CartBuilder.build accounts for 70% of sampled allocation pressure/)).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();

  expect(vi.mocked(getJVMEvents).mock.calls[0][0].toString()).toContain("event_type=gc_pause");
  expect(vi.mocked(getFlamegraph).mock.calls[0][0].toString()).toContain("profile_type=java_allocation_bytes");
});

test("keeps allocation evidence visible when ingestion health fails", async () => {
  vi.mocked(getIngestionHealth).mockRejectedValueOnce(new Error("503 Service Unavailable"));

  render(<MemoryPressureView params={params} />);

  expect(await screen.findByText("Ingestion evidence unavailable: 503 Service Unavailable")).toBeInTheDocument();
  expect(screen.getAllByText("4.0 MiB").length).toBeGreaterThan(0);
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
});
