import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, test, vi } from "vitest";
import { App } from "./app";

vi.mock("./routes/service-overview", () => ({
  ServiceOverview: ({ activeView }: { activeView: string }) => <div>active-view:{activeView}</div>,
}));

beforeEach(() => {
  vi.clearAllMocks();
});

test("left navigation changes the active diagnosis view", () => {
  render(<App />);

  expect(screen.getByText("active-view:cpu")).toBeInTheDocument();

  fireEvent.click(screen.getByLabelText("Service status"));
  expect(screen.getByText("active-view:status")).toBeInTheDocument();

  fireEvent.click(screen.getByLabelText("Allocation profiles"));
  expect(screen.getByText("active-view:memory")).toBeInTheDocument();
});
