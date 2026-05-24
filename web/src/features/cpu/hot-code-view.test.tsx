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

test("keeps fallback classifier aligned with representative backend frame classes", () => {
  const frames = collectHotJavaFrames({
    name: "root",
    value: 70,
    children: [
      { name: "libasyncProfiler.so.StackWalker::walkVM", value: 10 },
      { name: "libc-2.17.so.__clock_gettime", value: 10 },
      { name: "pthread_cond_timedwait", value: 10 },
      { name: "java.lang.Thread.run:1583", value: 10 },
      { name: "jdk.internal.reflect.NativeMethodAccessorImpl.invoke0", value: 10 },
      { name: "I2C adapter", value: 10 },
      { name: "com/acme/orders/CheckoutService.priceCart:42", value: 20 },
      { name: "com/acme/payments/PaymentAdapter.apply:51", value: 9 },
    ],
  });

  expect(frames.map((frame) => frame.fullSymbol)).toEqual(["com.acme.orders.CheckoutService.priceCart", "com.acme.payments.PaymentAdapter.apply"]);
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
  expect(within(analysis).getByRole("heading", { name: "Single Pod CPU profile" })).toBeInTheDocument();
  expect(screen.getByLabelText("CPU profile units")).toHaveTextContent("CPU time");
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
  expect(screen.getByRole("status")).toHaveTextContent("DemoHttpService.handleWork");
  expect(screen.queryByLabelText("Selected flamegraph frame")).not.toBeInTheDocument();
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
  expect(handleRow).toHaveTextContent("0 ns 0.0%");
  expect(handleRow).toHaveTextContent("10 ns 100.0%");
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

test("selecting backend rows highlights by full package location", () => {
  render(
    <HotCodeView
      root={{
        name: "root",
        value: 10,
        children: [
          { name: "com/foo/CheckoutService.priceCart:10", value: 6 },
          { name: "org/acme/CheckoutService.priceCart:42", value: 4 },
        ],
      }}
      metadata={{ partial: false }}
      topRows={[
        {
          symbol: "CheckoutService.priceCart",
          location: "com.foo.CheckoutService.priceCart:10",
          profile_type: "java_cpu_nanoseconds",
          self: 6,
          total: 6,
          self_percent: "60.0%",
          total_percent: "60.0%",
        },
        {
          symbol: "CheckoutService.priceCart",
          location: "org.acme.CheckoutService.priceCart:42",
          profile_type: "java_cpu_nanoseconds",
          self: 4,
          total: 4,
          self_percent: "40.0%",
          total_percent: "40.0%",
        },
      ]}
    />,
  );

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /CheckoutService:10/ }));

  expect(screen.getByRole("button", { name: /CheckoutService\.priceCart:10/ })).toHaveClass("flame-row-match");
  expect(screen.getByRole("button", { name: /CheckoutService\.priceCart:42/ })).not.toHaveClass("flame-row-match");
});

test("uses flamegraph fallback top table when backend top rows are empty", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} topRows={[]} />);

  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.getByRole("row", { name: /DemoHttpService\.burnCpu/ })).toBeInTheDocument();
});

test("search filters top table rows and highlights matching flame graph frames", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "handlework" } });

  const topTable = screen.getByRole("region", { name: "Top table" });
  expect(within(topTable).getByRole("row", { name: /DemoHttpService\.handleWork/ })).toBeInTheDocument();
  expect(within(topTable).queryByRole("row", { name: /DemoHttpService\.burnCpu/ })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
});

test("shows a recoverable no-match state for top table search", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "not-a-frame" } });

  const topTable = screen.getByRole("region", { name: "Top table" });
  expect(within(topTable).getByText("No Java frames match the current search.")).toBeInTheDocument();

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "" } });

  expect(within(topTable).getByRole("row", { name: /DemoHttpService\.burnCpu/ })).toBeInTheDocument();
});

test("summarizes the top Java row and follows explicit selection", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const summary = screen.getByRole("region", { name: "Selected hot Java frame" });
  expect(summary).toHaveTextContent("Top Java frame");
  expect(summary).toHaveTextContent("DemoHttpService.burnCpu");
  expect(summary).toHaveTextContent("High self");

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /DemoHttpService\.handleWork/ }));

  expect(summary).toHaveTextContent("Selected Java frame");
  expect(summary).toHaveTextContent("DemoHttpService.handleWork");
  expect(summary).toHaveTextContent("inspect callees");
  expect(screen.getByLabelText("Search flamegraph frames")).toHaveValue("");
});

test("keeps selected top row highlighted while search is active", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /DemoHttpService\.handleWork/ }));
  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "burncpu" } });

  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).not.toHaveClass("flame-row-dimmed");
  for (const frame of screen.getAllByRole("button", { name: /DemoHttpService\.burnCpu:188/ })) {
    expect(frame).toHaveClass("flame-row-match");
  }
});

test("clicking a flamegraph frame selects the matching backend top row", () => {
  render(
    <HotCodeView
      root={{ name: "root", value: 10, children: [{ name: "com/foo/CheckoutService.priceCart:10", value: 10 }] }}
      metadata={{ partial: false }}
      topRows={[
        {
          symbol: "CheckoutService.priceCart",
          location: "com.foo.CheckoutService.priceCart:10",
          profile_type: "java_cpu_nanoseconds",
          self: 10,
          total: 10,
          self_percent: "100.0%",
          total_percent: "100.0%",
        },
      ]}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /com\/foo\/CheckoutService\.priceCart:10/ }));

  expect(screen.getByRole("region", { name: "Selected hot Java frame" })).toHaveTextContent("Selected Java frame");
  expect(screen.getByRole("row", { name: /CheckoutService\.priceCart/ })).toHaveClass("active");
});

test("reset clears search and selected top table row state", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  const topTable = screen.getByRole("region", { name: "Top table" });
  fireEvent.click(within(topTable).getByRole("button", { name: /DemoHttpService\.handleWork/ }));
  expect(screen.getByRole("status")).toHaveTextContent("DemoHttpService.handleWork");
  expect(within(topTable).getByRole("row", { name: /DemoHttpService\.handleWork/ })).toHaveClass("active");

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "handlework" } });
  fireEvent.click(screen.getByRole("button", { name: "Reset view" }));

  expect(screen.getByLabelText("Search flamegraph frames")).toHaveValue("");
  expect(within(topTable).getByRole("row", { name: /DemoHttpService\.handleWork/ })).not.toHaveClass("active");
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

  expect(screen.getByRole("button", { name: "Both" })).toHaveAttribute("aria-pressed", "true");
  fireEvent.click(screen.getByRole("button", { name: "Top Table" }));
  expect(screen.getByRole("region", { name: "Top table" })).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Flamegraph" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Top Table" })).toHaveAttribute("aria-pressed", "true");

  fireEvent.click(screen.getByRole("button", { name: "Flame Graph" }));
  expect(screen.queryByRole("region", { name: "Top table" })).not.toBeInTheDocument();
  expect(screen.getByRole("region", { name: "Flamegraph" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Flame Graph" })).toHaveAttribute("aria-pressed", "true");
});
