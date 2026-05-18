import { render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { getFlamegraph, getJVMEvents } from "../../api/client";
import { GCView } from "./gc-view";

vi.mock("../../api/client", () => ({
  getFlamegraph: vi.fn(),
  getJVMEvents: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({
    root: { name: "root", value: 100, children: [{ name: "AllocFrame.write", value: 100 }] },
    metadata: { partial: false },
  });
  vi.mocked(getJVMEvents).mockResolvedValue({
    events: [
      {
        event_id: "gc-1",
        event_type: "gc_pause",
        event_at: "2026-05-18T00:00:00.000Z",
        duration_ns: 10_000_000,
        action: "end of minor GC",
        cause: "allocation failure",
      },
      {
        event_id: "gc-2",
        event_type: "gc_pause",
        event_at: "2026-05-18T00:00:10.000Z",
        duration_ns: 25_000_000,
        action: "end of minor GC",
        cause: "allocation failure",
      },
    ],
    partial: false,
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

test("shows GC summary cards before the event list", async () => {
  render(<GCView params={new URLSearchParams("namespace=java-profiler-qa&service=jdk17-http-demo")} />);

  const summary = await screen.findByLabelText("GC summary");
  expect(within(summary).getByText("GC events")).toBeInTheDocument();
  expect(within(summary).getByText("2")).toBeInTheDocument();
  expect(within(summary).getByText("35.0 ms")).toBeInTheDocument();
  expect(within(summary).getByText("25.0 ms")).toBeInTheDocument();
  expect(within(summary).getByText("17.5 ms")).toBeInTheDocument();
});
