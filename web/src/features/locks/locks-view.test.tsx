import { render, screen } from "@testing-library/react";
import { beforeEach, test, vi } from "vitest";
import { getFlamegraph, getIngestionHealth, getTargetStatus } from "../../api/client";
import { LocksView } from "./locks-view";

vi.mock("../../api/client", () => ({
  getFlamegraph: vi.fn(),
  getIngestionHealth: vi.fn(),
  getTargetStatus: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({ root: { name: "root", value: 0, children: [] }, metadata: { partial: false } });
  vi.mocked(getTargetStatus).mockResolvedValue([]);
  vi.mocked(getIngestionHealth).mockResolvedValue({
    totals: { accepted: 10, duplicate: 0, retryable: 1, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
    batches: [{ batch_type: "profile", status: "retryable", retryable: true, count: 1, latest_at: "2026-05-24T12:30:00Z" }],
    partial: false,
  });
});

test("renders aggregate ingestion evidence for empty lock profiles", async () => {
  render(<LocksView params={new URLSearchParams("namespace=prod&service=checkout&profile_type=java_lock_delay_nanoseconds")} />);

  expect(await screen.findByText(/Recent aggregate ingestion health/)).toBeInTheDocument();
  expect(screen.getByText(/not scoped to the selected service/)).toBeInTheDocument();
});
