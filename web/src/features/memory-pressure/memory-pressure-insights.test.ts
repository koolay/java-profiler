import { describe, expect, test } from "vitest";
import type { AllocationSummary, JVMEventEvidence } from "../../api/types";
import { buildMemoryPressureInsights } from "./memory-pressure-insights";

function allocationSummary(overrides: Partial<AllocationSummary> = {}): AllocationSummary {
  return {
    schema_version: 1,
    requested_scope: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "", jvm: "", start: "", end: "" },
    effective_scope: { namespace: "prod", service: "checkout", pod: "checkout-1", container: "", jvm: "", start: "", end: "" },
    coverage: {
      has_data: true,
      profile_type: "java_allocation_bytes",
      total_value: 10 * 1024 * 1024,
      value_unit: "bytes",
      scanned_samples: 20,
      returned_paths: 1,
      returned_self_frames: 1,
      omitted_paths_lower_bound: 0,
      omitted_nodes_lower_bound: 0,
      partial: false,
      empty_state: "",
    },
    top_paths: [
      {
        rank: 1,
        leaf_frame: "com.example.BigAllocator.allocate",
        category: "application",
        total_value: 8 * 1024 * 1024,
        self_value: 1024,
        percent: 80,
        sample_count: 12,
        path: ["root", "com.example.BigAllocator.allocate"],
      },
    ],
    top_self_frames: [
      {
        rank: 1,
        frame: "com.example.BigAllocator.allocate",
        category: "application",
        self_value: 7 * 1024 * 1024,
        percent: 70,
      },
    ],
    insights: [],
    limitations: [],
    ...overrides,
  };
}

const emptyGC: JVMEventEvidence = { events: [], partial: false };

describe("buildMemoryPressureInsights", () => {
  test("identifies concentrated sampled allocation pressure without claiming retained heap ownership", () => {
    const insights = buildMemoryPressureInsights(allocationSummary(), emptyGC);

    expect(insights.map((insight) => insight.code)).toContain("high_total_path_allocation");
    expect(insights.map((insight) => insight.code)).toContain("high_self_frame_allocation");
    expect(insights.map((insight) => insight.code)).toContain("allocation_concentration");
    expect(insights.map((insight) => insight.message).join(" ")).toContain("sampled allocation");
    expect(insights.map((insight) => insight.message).join(" ")).not.toMatch(/retained heap owner/i);
  });

  test("correlates GC pauses with allocation evidence", () => {
    const insights = buildMemoryPressureInsights(allocationSummary(), {
      events: [{ event_id: "gc-1", event_type: "gc_pause", event_at: "2026-06-11T00:00:00Z", duration_ns: 200_000_000 }],
      partial: false,
    });

    expect(insights.map((insight) => insight.code)).toContain("gc_with_allocation_pressure");
  });

  test("flags GC evidence when allocation samples are missing", () => {
    const insights = buildMemoryPressureInsights(
      allocationSummary({
        coverage: {
          has_data: false,
          profile_type: "java_allocation_bytes",
          total_value: 0,
          value_unit: "bytes",
          scanned_samples: 0,
          returned_paths: 0,
          returned_self_frames: 0,
          omitted_paths_lower_bound: 0,
          omitted_nodes_lower_bound: 0,
          partial: false,
          empty_state: "no_samples",
        },
        top_paths: [],
        top_self_frames: [],
      }),
      { events: [{ event_id: "gc-1", event_type: "gc_pause", event_at: "2026-06-11T00:00:00Z", duration_ns: 50_000_000 }], partial: false },
    );

    expect(insights).toEqual([
      expect.objectContaining({
        code: "gc_without_allocation_evidence",
        severity: "warning",
      }),
    ]);
  });
});
