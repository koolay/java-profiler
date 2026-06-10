import type { AllocationSummary, JVMEventEvidence } from "../../api/types";

export type MemoryPressureInsight = {
  code:
    | "high_total_path_allocation"
    | "high_self_frame_allocation"
    | "allocation_concentration"
    | "gc_with_allocation_pressure"
    | "gc_without_allocation_evidence"
    | "allocation_without_gc_evidence";
  severity: "info" | "warning";
  message: string;
  frame?: string;
};

const concentrationPercent = 60;

export function buildMemoryPressureInsights(summary: AllocationSummary, gcEvidence: JVMEventEvidence): MemoryPressureInsight[] {
  const insights: MemoryPressureInsight[] = [];
  const hasAllocationData = Boolean(summary.coverage?.has_data && summary.coverage.total_value > 0);
  const gcCount = gcEvidence.events?.length ?? 0;
  const topPath = summary.top_paths?.[0];
  const topSelfFrame = summary.top_self_frames?.[0];

  if (!hasAllocationData) {
    if (gcCount > 0) {
      insights.push({
        code: "gc_without_allocation_evidence",
        severity: "warning",
        message: "GC pause evidence exists, but sampled allocation evidence is missing or stale for this window.",
      });
    }
    return insights;
  }

  if (topPath) {
    insights.push({
      code: "high_total_path_allocation",
      severity: "info",
      frame: topPath.leaf_frame,
      message: `${topPath.leaf_frame} is the highest sampled allocation path in this window.`,
    });
  }

  if (topSelfFrame) {
    insights.push({
      code: "high_self_frame_allocation",
      severity: "info",
      frame: topSelfFrame.frame,
      message: `${topSelfFrame.frame} has the highest sampled allocation self cost.`,
    });
  }

  if (topPath && topPath.percent >= concentrationPercent) {
    insights.push({
      code: "allocation_concentration",
      severity: "warning",
      frame: topPath.leaf_frame,
      message: `${topPath.leaf_frame} accounts for ${Math.round(topPath.percent)}% of sampled allocation pressure; inspect this path for object churn.`,
    });
  }

  if (gcCount > 0) {
    insights.push({
      code: "gc_with_allocation_pressure",
      severity: "warning",
      message: `${gcCount} GC pause event${gcCount === 1 ? "" : "s"} occurred while sampled allocation pressure was present.`,
    });
  } else {
    insights.push({
      code: "allocation_without_gc_evidence",
      severity: "info",
      message: "Sampled allocation pressure is present, but no GC pause evidence was returned for this window.",
    });
  }

  return insights;
}
