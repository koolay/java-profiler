import { render, screen } from "@testing-library/react";
import { beforeEach, test, vi } from "vitest";
import { getFlamegraph, getIngestionHealth, getTargetStatus, getTopStacks } from "../../api/client";
import { IOView } from "./io-view";

vi.mock("../../api/client", () => ({
  getFlamegraph: vi.fn(),
  getIngestionHealth: vi.fn(),
  getTargetStatus: vi.fn(),
  getTopStacks: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({ root: { name: "root", value: 0, children: [] }, metadata: { partial: false } });
  vi.mocked(getTopStacks).mockResolvedValue([]);
  vi.mocked(getTargetStatus).mockResolvedValue([
    {
      target: { namespace: "prod", service: "checkout", pod: "checkout-1" },
      desired_state: "disabled",
      reason: "disabled_by_metadata",
      message: "profiling not enabled",
      status_at: "2026-05-24T12:00:00Z",
    },
  ]);
  vi.mocked(getIngestionHealth).mockResolvedValue({
    totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
    batches: [],
    partial: false,
  });
});

test("renders disabled target evidence for empty I/O profiles", async () => {
  render(<IOView params={new URLSearchParams("namespace=prod&service=checkout&pod=checkout-1&profile_type=java_io_wait_nanoseconds")} />);

  expect(await screen.findByText(/profiling not enabled/)).toBeInTheDocument();
});
