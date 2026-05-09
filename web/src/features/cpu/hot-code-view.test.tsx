import { fireEvent, render, screen, within } from "@testing-library/react";
import { collectHotJavaFrames, HotCodeView } from "./hot-code-view";

const root = {
  name: "root",
  value: 110,
  children: [
    { name: "so.6", value: 61 },
    {
      name: "com/ebpfjava/examples/httpdemo/DemoHttpService.handleWork:93",
      value: 16,
      children: [{ name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188", value: 12 }],
    },
    { name: "com/ebpfjava/examples/httpdemo/DemoHttpService$$Lambda_.handle", value: 20 },
    { name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188", value: 12 },
    { name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:189", value: 2 },
    { name: "org/acme/orders/CheckoutService.priceCart:42", value: 3 },
    { name: "java/lang/Thread.run:1583", value: 5 },
    { name: "libjvm.so.NativeFrame", value: 10 },
    { name: "pthread_cond_timedwait", value: 7 },
  ],
};

test("computes Pyroscope-style self and total metrics for actionable Java frames", () => {
  const frames = collectHotJavaFrames(root);

  expect(frames.map((frame) => frame.symbol)).toEqual(expect.arrayContaining(["DemoHttpService.burnCpu", "DemoHttpService.handleWork", "CheckoutService.priceCart"]));
  expect(frames.map((frame) => frame.symbol)).not.toContain("so.6");
  expect(frames.find((frame) => frame.symbol === "DemoHttpService.burnCpu")).toMatchObject({ self: 26, total: 26 });
  expect(frames.find((frame) => frame.symbol === "DemoHttpService.handleWork")).toMatchObject({ self: 4, total: 16 });
  expect(frames.find((frame) => frame.symbol === "CheckoutService.priceCart")).toMatchObject({ self: 3, total: 3 });
});

test("renders top table and flame graph in both mode", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const analysis = screen.getByLabelText("CPU profile analysis");
  expect(within(analysis).getByRole("heading", { name: "CPU profile" })).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Self" })).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Total" })).toBeInTheDocument();
  expect(screen.getByRole("row", { name: /DemoHttpService\.burnCpu/ })).toBeInTheDocument();
  expect(screen.queryByRole("row", { name: /so\.6/ })).not.toBeInTheDocument();

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /DemoHttpService\.handleWork/ }));
  expect(screen.getByPlaceholderText("Search frame")).toHaveValue("");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
});

test("sorts the top table by total by default and supports self sorting", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const table = screen.getByRole("region", { name: "Top table" });
  const rows = within(table).getAllByRole("row");
  expect(rows[1]).toHaveTextContent("DemoHttpService.burnCpu");

  fireEvent.click(screen.getByRole("button", { name: "Self" }));
  const selfSortedRows = within(table).getAllByRole("row");
  expect(selfSortedRows[1]).toHaveTextContent("DemoHttpService.burnCpu");
});

test("switches between top table and flame graph views", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  fireEvent.click(screen.getByRole("button", { name: "Top Table" }));
  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Flamegraph" })).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Flame Graph" }));
  expect(screen.queryByRole("region", { name: "Top table" })).not.toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
});
