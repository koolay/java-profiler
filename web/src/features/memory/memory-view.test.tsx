import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { getAllocationSummary, getFlamegraph } from "../../api/client";
import { MemoryView } from "./memory-view";

vi.mock("../../api/client", () => ({
  getAllocationSummary: vi.fn(),
  getFlamegraph: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({
    root: { name: "root", value: 1024, children: [{ name: "Checkout.allocate", value: 1024 }] },
    metadata: { partial: false },
  });
  vi.mocked(getAllocationSummary).mockResolvedValue({
    schema_version: 1,
    requested_scope: { namespace: "prod", service: "all", pod: "all", container: "all", jvm: "all", start: "2026-05-24T12:00:00Z", end: "2026-05-24T12:30:00Z" },
    effective_scope: { namespace: "prod", service: "", pod: "", container: "", jvm: "", start: "2026-05-24T12:00:00Z", end: "2026-05-24T12:30:00Z" },
    coverage: {
      has_data: true,
      profile_type: "java_allocation_bytes",
      total_value: 2 * 1024 * 1024,
      value_unit: "bytes",
      scanned_samples: 12,
      returned_paths: 1,
      returned_self_frames: 1,
      omitted_paths_lower_bound: 0,
      omitted_nodes_lower_bound: 0,
      partial: false,
    },
    top_paths: [{ rank: 1, leaf_frame: "java/lang/StringBuilder.append:136", total_value: 2 * 1024 * 1024, self_value: 1024, percent: 80, category: "string_construction", sample_count: 12, path: ["root", "java/lang/StringBuilder.append:136"] }],
    top_self_frames: [{ rank: 1, frame: "java/lang/StringBuilder.append:136", self_value: 1024, percent: 10, category: "string_construction" }],
    insights: [{ severity: "info", category: "string_construction", message_code: "allocation.string_construction.dominant", evidence_frame: "java/lang/StringBuilder.append:136", evidence_value: 2 * 1024 * 1024 }],
    limitations: [],
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

test("renders allocation summary evidence and retained-heap semantics", async () => {
  render(<MemoryView params={new URLSearchParams("namespace=prod&profile_type=java_allocation_bytes&start=2026-05-24T12:00:00Z&end=2026-05-24T12:30:00Z")} />);

  expect(await screen.findAllByText("2.0 MiB")).toHaveLength(2);
  expect(screen.getByText("Sampled allocations")).toBeInTheDocument();
  expect(screen.getByText(/String construction is a dominant allocation source/)).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Top allocating paths" })).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Top self allocating frames" })).toBeInTheDocument();
});

test("shows classified empty state", async () => {
  vi.mocked(getAllocationSummary).mockResolvedValueOnce({
    schema_version: 1,
    requested_scope: { namespace: "prod", service: "", pod: "", container: "", jvm: "", start: "2026-05-24T12:00:00Z", end: "2026-05-24T12:30:00Z" },
    effective_scope: { namespace: "prod", service: "", pod: "", container: "", jvm: "", start: "2026-05-24T12:00:00Z", end: "2026-05-24T12:30:00Z" },
    coverage: {
      has_data: false,
      empty_state: "profiling_disabled",
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
  });

  render(<MemoryView params={new URLSearchParams("namespace=prod&profile_type=java_allocation_bytes&start=2026-05-24T12:00:00Z&end=2026-05-24T12:30:00Z")} />);

  expect(await screen.findByText("This target is visible, but allocation profiling is not enabled for it.")).toBeInTheDocument();
});

test("shows partial warning and falls back when summary endpoint fails", async () => {
  vi.mocked(getAllocationSummary).mockRejectedValueOnce(new Error("404 Not Found"));

  render(<MemoryView params={new URLSearchParams("namespace=prod&profile_type=java_allocation_bytes&start=2026-05-24T12:00:00Z&end=2026-05-24T12:30:00Z")} />);

  expect(await screen.findByText("Allocation summary unavailable: 404 Not Found. Showing flamegraph evidence only.")).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
});
