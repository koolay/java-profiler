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
    { name: "java.lang.Thread.run:1583", value: 4 },
    { name: "I2C.C2I adapters", value: 9 },
    { name: "libjvm.so.NativeFrame", value: 10 },
    { name: "pthread_cond_timedwait", value: 7 },
  ],
};

test("computes Pyroscope-style self and total metrics for actionable Java frames", () => {
  const frames = collectHotJavaFrames(root);

  expect(frames.map((frame) => frame.symbol)).toEqual(expect.arrayContaining(["DemoHttpService.burnCpu", "DemoHttpService.handleWork", "CheckoutService.priceCart"]));
  expect(frames.map((frame) => frame.symbol)).not.toContain("so.6");
  expect(frames.map((frame) => frame.symbol)).not.toContain("I2C.C2I adapters");
  expect(frames.map((frame) => frame.symbol)).not.toContain("Thread.run");
  expect(frames.find((frame) => frame.symbol === "DemoHttpService.burnCpu")).toMatchObject({ self: 26, total: 26 });
  expect(frames.find((frame) => frame.symbol === "DemoHttpService.handleWork")).toMatchObject({ self: 4, total: 16 });
  expect(frames.find((frame) => frame.symbol === "CheckoutService.priceCart")).toMatchObject({ self: 3, total: 3 });
});

test("keeps same class names from different packages as distinct hot code rows", () => {
  const frames = collectHotJavaFrames({
    name: "root",
    value: 10,
    children: [
      { name: "com/foo/CheckoutService.priceCart:10", value: 6 },
      { name: "org/acme/CheckoutService.priceCart:42", value: 4 },
    ],
  });

  expect(frames.filter((frame) => frame.symbol === "CheckoutService.priceCart")).toHaveLength(2);
  expect(frames.map((frame) => frame.fullSymbol)).toEqual(expect.arrayContaining(["com.foo.CheckoutService.priceCart", "org.acme.CheckoutService.priceCart"]));
});

test("keeps flame graph visible when CPU profile has no actionable Java frames", () => {
  render(
    <HotCodeView
      root={{
        name: "root",
        value: 14,
        children: [
          { name: "so.6", value: 8 },
          { name: "java.lang.Thread.run:1583", value: 4 },
          { name: "I2C.C2I adapters", value: 2 },
        ],
      }}
    />,
  );

  expect(screen.getByText(/No application Java frames were found/)).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Top table" })).not.toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Thread\.run/ })).toHaveClass("flame-row-runtime");
});

test("renders top table and flame graph in both mode", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const analysis = screen.getByLabelText("CPU profile analysis");
  expect(within(analysis).getByRole("heading", { name: "CPU profile" })).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Self CPU" })).toBeInTheDocument();
  expect(screen.getByRole("columnheader", { name: "Total CPU" })).toBeInTheDocument();
  expect(screen.getByRole("row", { name: /DemoHttpService\.burnCpu/ })).toBeInTheDocument();
  expect(screen.queryByRole("row", { name: /so\.6/ })).not.toBeInTheDocument();

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /DemoHttpService\.handleWork/ }));
  expect(screen.getByPlaceholderText("Search frame")).toHaveValue("");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
  expect(screen.getByRole("button", { name: /so\.6/ })).not.toHaveClass("flame-row-dimmed");
  expect(screen.getByText(/High total, low self: start from DemoHttpService\.handleWork/)).toBeInTheDocument();
});

test("renders backend top rows with self and total CPU values", () => {
  render(
    <HotCodeView
      root={root}
      metadata={{ partial: false }}
      topRows={[
        {
          symbol: "DemoHttpService.handleWork",
          location: "com/ebpfjava/examples/httpdemo/DemoHttpService.handleWork:93",
          profile_type: "java_cpu_nanoseconds",
          self: 0,
          total: 10,
          self_percent: "0.0%",
          total_percent: "100.0%",
        },
        {
          symbol: "DemoHttpService.burnCpu",
          location: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188",
          profile_type: "java_cpu_nanoseconds",
          self: 8,
          total: 8,
          self_percent: "80.0%",
          total_percent: "80.0%",
        },
      ]}
    />,
  );

  const handleRow = screen.getByRole("row", { name: /DemoHttpService\.handleWork/ });
  expect(handleRow).toHaveTextContent("0 0.0%");
  expect(handleRow).toHaveTextContent("10 100.0%");
});

test("selecting a backend top row highlights matching flame graph frames", () => {
  render(
    <HotCodeView
      root={root}
      metadata={{ partial: false }}
      topRows={[
        {
          symbol: "DemoHttpService.handleWork",
          location: "com/ebpfjava/examples/httpdemo/DemoHttpService.handleWork:93",
          profile_type: "java_cpu_nanoseconds",
          self: 0,
          total: 10,
          self_percent: "0.0%",
          total_percent: "100.0%",
        },
      ]}
    />,
  );

  fireEvent.click(within(screen.getByRole("region", { name: "Top table" })).getByRole("button", { name: /DemoHttpService\.handleWork/ }));

  expect(screen.getByPlaceholderText("Search frame")).toHaveValue("");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
});

test("uses flamegraph fallback top table when backend top rows are empty", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} topRows={[]} />);

  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.getByRole("row", { name: /DemoHttpService\.burnCpu/ })).toBeInTheDocument();
});

test("sorts the top table by total by default and supports self sorting", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const table = screen.getByRole("region", { name: "Top table" });
  const rows = within(table).getAllByRole("row");
  expect(rows[1]).toHaveTextContent("DemoHttpService.burnCpu");

  fireEvent.click(screen.getByRole("button", { name: "Self CPU" }));
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
