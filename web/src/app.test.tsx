import { render, screen } from "@testing-library/react";
import { beforeEach, test, vi } from "vitest";
import { App } from "./app";

vi.mock("./routes/service-overview", () => ({
  ServiceOverview: ({ activeView }: { activeView: string }) => <div>active-view:{activeView}</div>,
}));

beforeEach(() => {
  vi.clearAllMocks();
});

test("renders the Java profiler workbench with CPU as the initial view", () => {
  render(<App />);

  expect(screen.getByText("active-view:cpu")).toBeInTheDocument();
});
