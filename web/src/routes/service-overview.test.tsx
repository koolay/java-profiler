import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ServiceOverview } from "./service-overview";

let useAPICall = 0;
let currentTargets = [{ namespace: "ns-a", service: "svc-a", pod: "pod-a" }];
let historicalTargets = [{ namespace: "ns-a", service: "svc-a", pod: "pod-a" }];

vi.mock("../api/use-api", () => ({
  useAPI: () => ({
    data: { targets: useAPICall++ % 2 === 0 ? currentTargets : historicalTargets },
    error: null,
    loading: false,
  }),
}));

vi.mock("../features/status/target-status-view", () => ({
  TargetStatusView: () => <div data-testid="target-status-view" />,
}));

vi.mock("../features/memory-pressure/memory-pressure-view", () => ({
  MemoryPressureView: () => <section aria-label="Memory pressure investigation" />,
}));

afterEach(() => {
  vi.restoreAllMocks();
  useAPICall = 0;
  currentTargets = [{ namespace: "ns-a", service: "svc-a", pod: "pod-a" }];
  historicalTargets = [{ namespace: "ns-a", service: "svc-a", pod: "pod-a" }];
});

test("starts with empty selectors and compact time presets", () => {
  render(<ServiceOverview activeView="status" onViewChange={vi.fn()} />);

  expect(screen.getByRole("combobox", { name: "Namespace" })).toHaveValue("");
  expect(screen.getByRole("combobox", { name: "Service" })).toHaveValue("");
  expect(screen.getByRole("combobox", { name: "Pod" })).toHaveValue("");
  expect(screen.getByRole("button", { name: "5m" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "15m" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "30m" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "1h" })).toBeInTheDocument();
  expect(screen.queryByLabelText("From")).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Custom" }));

  expect(screen.getByText(/\d{4}\/\d{2}\/\d{2}, \d{2}:\d{2}:\d{2} → \d{4}\/\d{2}\/\d{2}, \d{2}:\d{2}:\d{2}/)).toBeInTheDocument();
  expect(screen.getByLabelText("From")).toHaveAttribute("type", "datetime-local");
  expect(screen.getByLabelText("To")).toHaveAttribute("type", "datetime-local");
});

test("shows historical selector guidance when current range has no suggestions", () => {
  currentTargets = [];
  historicalTargets = [{ namespace: "historical-ns", service: "historical-svc", pod: "historical-pod" }];

  render(<ServiceOverview activeView="status" onViewChange={vi.fn()} />);

  expect(screen.getByText("No selector suggestions have samples in this time range. Showing historical targets; adjust the range or type a value directly.")).toBeInTheDocument();
});

test("routes memory pressure investigation from the workbench navigation", () => {
  const onViewChange = vi.fn();

  render(<ServiceOverview activeView={"memory-pressure" as never} onViewChange={onViewChange} />);

  expect(screen.getByRole("region", { name: "Memory pressure investigation" })).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: /OOM investigation/i }));

  expect(onViewChange).toHaveBeenCalledWith("memory-pressure");
});
