import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { getFlamegraph, getIngestionHealth, getTargetStatus, getTopStacks } from "../../api/client";
import { CpuView } from "./cpu-view";

vi.mock("../../api/client", () => ({
  getFlamegraph: vi.fn(),
  getIngestionHealth: vi.fn(),
  getTargetStatus: vi.fn(),
  getTopStacks: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({
    root: { name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] },
    metadata: { partial: false },
  });
  vi.mocked(getTargetStatus).mockResolvedValue([]);
  vi.mocked(getIngestionHealth).mockResolvedValue({
    totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
    batches: [],
    partial: false,
  });
  vi.mocked(getTopStacks).mockResolvedValue([]);
});

afterEach(() => {
  vi.clearAllMocks();
});

test("shows a warning when the top-stacks endpoint fails", async () => {
  vi.mocked(getTopStacks).mockRejectedValueOnce(new Error("503 Service Unavailable"));

  render(<CpuView params={new URLSearchParams("namespace=java-profiler-qa&service=jdk17-http-demo")} />);

  expect(await screen.findByText("Top table unavailable: 503 Service Unavailable")).toBeInTheDocument();
});

test("shows evidence-aware empty state when temporary profiling expired", async () => {
  vi.mocked(getFlamegraph).mockResolvedValueOnce({ root: { name: "root", value: 0, children: [] }, metadata: { partial: false } });
  vi.mocked(getTargetStatus).mockResolvedValueOnce([
    {
      target: { namespace: "java-profiler-qa", service: "checkout", pod: "checkout-1" },
      desired_state: "disabled",
      reason: "temporary_expired",
      message: "temporary profiling expired",
      status_at: "2026-05-24T12:00:00Z",
    },
  ]);

  render(<CpuView params={new URLSearchParams("namespace=java-profiler-qa&service=checkout&pod=checkout-1")} />);

  expect(await screen.findByText(/temporary profiling has expired/)).toBeInTheDocument();
});
