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
    root: { name: "root", value: 100, children: [{ name: "com/ebpfjava/examples/httpdemo/DemoHttpService.handleWork:93", value: 100 }] },
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

test("shows GC summary and ranks the largest pause first", async () => {
  render(<GCView params={new URLSearchParams("namespace=java-profiler-qa&service=jdk17-http-demo&start=2026-05-18T00:00:00.000Z&end=2026-05-18T00:01:00.000Z")} />);

  const summary = await screen.findByLabelText("GC summary");
  expect(within(summary).getByText("GC events")).toBeInTheDocument();
  expect(within(summary).getByText("2")).toBeInTheDocument();
  expect(within(summary).getByText("35.0 ms")).toBeInTheDocument();
  expect(within(summary).getByText("2.0 events/min")).toBeInTheDocument();
  expect(within(summary).getByText("0.06% of window")).toBeInTheDocument();
  expect(within(summary).getByText("Longest pause")).toBeInTheDocument();
  expect(within(summary).getAllByText("25.0 ms")).toHaveLength(2);
  expect(within(summary).getByText("P95 pause")).toBeInTheDocument();
  expect(within(summary).getByText("Top cause: allocation failure")).toBeInTheDocument();
  expect(screen.getByLabelText("GC interpretation")).toHaveTextContent("Low pause impact");
  expect(screen.getByRole("heading", { name: "Allocation correlation" })).toBeInTheDocument();
  const rows = screen.getAllByRole("article");
  expect(rows).toHaveLength(2);
  expect(rows[0]).toHaveTextContent("25.0 ms");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toBeInTheDocument();
  expect(screen.queryByText("Focus frame")).not.toBeInTheDocument();
  expect(screen.getByText("Copy frame")).toBeInTheDocument();
  expect(screen.getByText("Permalink")).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Selected flamegraph frame" })).not.toBeInTheDocument();
});

test("shows a GC empty state when no pause events are available", async () => {
  vi.mocked(getJVMEvents).mockResolvedValueOnce({ events: [], partial: false });
  vi.mocked(getFlamegraph).mockResolvedValueOnce({
    root: { name: "service", value: 0, children: [] },
    metadata: { partial: false },
  });

  render(<GCView params={new URLSearchParams("namespace=java-profiler-qa&service=jdk17-http-demo")} />);

  expect(await screen.findByText("No GC pause event evidence in this range.")).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Allocation correlation" })).toBeInTheDocument();
  expect(screen.getByText("No application Java frames were found in this profile. Use the flame graph to inspect runtime or native frames.")).toBeInTheDocument();
});
