import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ServiceOverview } from "./service-overview";

vi.mock("../api/use-api", () => ({
  useAPI: () => ({
    data: { targets: [{ namespace: "ns-a", service: "svc-a", pod: "pod-a" }] },
    error: null,
    loading: false,
  }),
}));

vi.mock("../features/status/target-status-view", () => ({
  TargetStatusView: () => <div data-testid="target-status-view" />,
}));

afterEach(() => {
  vi.restoreAllMocks();
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

  expect(screen.getByLabelText("From")).toHaveAttribute("type", "datetime-local");
  expect(screen.getByLabelText("To")).toHaveAttribute("type", "datetime-local");
});
