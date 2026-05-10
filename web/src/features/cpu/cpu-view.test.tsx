import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, test, vi } from "vitest";
import { getFlamegraph, getTopStacks } from "../../api/client";
import { CpuView } from "./cpu-view";

vi.mock("../../api/client", () => ({
  getFlamegraph: vi.fn(),
  getTopStacks: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getFlamegraph).mockResolvedValue({
    root: { name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] },
    metadata: { partial: false },
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
