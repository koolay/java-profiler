import { fireEvent, render, screen, within } from "@testing-library/react";
import { Flamegraph } from "./flamegraph";

test("renders flamegraph frames and partial warning", () => {
  render(<Flamegraph root={{ name: "root", value: 14, children: [{ name: "Checkout.handle", value: 10 }, { name: "java.lang.Thread.run", value: 4 }] }} metadata={{ partial: true, reasons: ["node_limit"] }} />);
  expect(screen.getByText("Checkout.handle")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Checkout\.handle/ })).toHaveClass("flame-row-application");
  expect(screen.getByRole("button", { name: /Thread\.run/ })).toHaveClass("flame-row-runtime");
  expect(screen.getByText(/Partial result/)).toBeInTheDocument();
  const legend = screen.getByLabelText("Frame categories");
  expect(within(legend).getByText("Application Java")).toBeInTheDocument();
  expect(within(legend).getByText("JVM/runtime")).toBeInTheDocument();
  expect(within(legend).getByText("Native/system")).toBeInTheDocument();
});

test("inspects, searches, and zooms nested frames while keeping long labels readable", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 12,
        children: [
          {
            name: "libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis",
            value: 8,
            children: [{ name: "BusyApp.lambda$main$0:14", value: 8 }],
          },
          { name: "java/lang/Thread.run", value: 4 },
        ],
      }}
    />,
  );

  const nativeFrame = screen.getByRole("button", { name: /VeryLongNativeFrameName/ });
  expect(nativeFrame).toHaveClass("flame-row-native");
  fireEvent.mouseEnter(nativeFrame);
  const inspector = screen.getByRole("status");
  expect(within(inspector).getByText("Native/system")).toBeInTheDocument();
  expect(within(inspector).getByText("libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis")).toBeInTheDocument();
  expect(within(inspector).getByText("Total Samples")).toBeInTheDocument();
  expect(within(inspector).getByText("Self CPU")).toBeInTheDocument();

  fireEvent.click(nativeFrame);
  const detail = screen.getByLabelText("Selected flamegraph frame");
  expect(within(detail).getByText("libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis")).toBeInTheDocument();
  expect(within(detail).getByText("66.7%")).toBeInTheDocument();
  expect(within(detail).getByText("Self CPU")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Focus selected" }));
  expect(screen.getByText("BusyApp.lambda$main$0:14")).toBeInTheDocument();
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("Focused");
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("libjvm.so VeryLongNativeFrameNameThatWillNeedEllipsis");

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "busyapp" } });
  expect(screen.getByText("BusyApp.lambda$main$0:14").closest("button")).toHaveClass("flame-row-match");
  expect(screen.getByRole("button", { name: /VeryLongNativeFrameName/ })).toHaveClass("flame-row-dimmed");

  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(screen.getByRole("button", { name: /Thread\.run/ })).toBeInTheDocument();
});

test("focuses the selected frame even when hover has moved to a different frame", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 100,
        children: [
          { name: "SelectedFrame.method:10", value: 40, children: [{ name: "SelectedFrame.child:11", value: 40 }] },
          { name: "HoveredFrame.method:20", value: 60, children: [{ name: "HoveredFrame.child:21", value: 60 }] },
        ],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /SelectedFrame\.method:10/ }));
  fireEvent.mouseEnter(screen.getByRole("button", { name: /HoveredFrame\.method:20/ }));
  fireEvent.click(screen.getByRole("button", { name: "Focus selected" }));

  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("SelectedFrame.method:10");
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).not.toHaveTextContent("HoveredFrame.method:20");
});

test("returns to the previous zoom level with Back", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 8,
        children: [{ name: "DemoHttpService.handleWork:93", value: 8, children: [{ name: "DemoHttpService$$Lambda_.handle", value: 8 }] }],
      }}
    />,
  );

  expect(screen.getByRole("button", { name: "Back" })).toBeDisabled();

  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ }));
  fireEvent.click(screen.getByRole("button", { name: "Focus selected" }));
  expect(screen.getByRole("button", { name: "Back" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: "root 8" })).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByRole("button", { name: "root8" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Back" })).toBeDisabled();
});

test("shows real Java demo frame names in the detail panel", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 47,
        children: [
          {
            name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188",
            value: 5,
          },
        ],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.burnCpu/ }));

  const detail = screen.getByLabelText("Selected flamegraph frame");
  expect(within(detail).getByText("com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188")).toBeInTheDocument();
  expect(within(detail).getAllByText("10.6%")).toHaveLength(2);
});

test("highlights selected Java frames without replacing flamegraph context", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 21,
        children: [
          { name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188", value: 7 },
          { name: "libjvm.so.NativeFrame", value: 7, children: [{ name: "com/ebpfjava/examples/httpdemo/DemoHttpService.burnCpu:188", value: 7 }] },
          { name: "com/ebpfjava/examples/httpdemo/DemoHttpService.handleWork:93", value: 7 },
        ],
      }}
      highlightQuery="DemoHttpService"
    />,
  );

  const burnCpuFrames = screen.getAllByRole("button", { name: /DemoHttpService\.burnCpu:188/ });
  expect(burnCpuFrames).toHaveLength(2);
  expect(screen.getByRole("button", { name: /^root/ })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /NativeFrame/ })).not.toHaveClass("flame-row-dimmed");
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
  expect(screen.getByText(/Full sampled stack context/)).toBeInTheDocument();

  fireEvent.click(burnCpuFrames[0]);
  expect(within(screen.getByLabelText("Selected flamegraph frame")).getAllByText("7")).toHaveLength(2);

  fireEvent.click(screen.getByRole("button", { name: "Focus selected" }));
  expect(screen.getByText(/Focused stack context/)).toBeInTheDocument();
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("DemoHttpService.burnCpu:188");
});

test("shows self and total metrics for leaf frames in the inspector", () => {
  render(<Flamegraph root={{ name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] }} />);

  fireEvent.mouseEnter(screen.getByRole("button", { name: /Checkout\.handle/ }));

  const inspector = screen.getByRole("status");
  expect(within(inspector).getByText("Application Java")).toBeInTheDocument();
  expect(within(inspector).getByText("Checkout.handle")).toBeInTheDocument();
  expect(within(inspector).getAllByText(/10/).length).toBeGreaterThanOrEqual(2);
  expect(within(inspector).getAllByText("100.0%")).toHaveLength(2);
});

test("keeps child width proportional to parent total when the parent has self CPU", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 100,
        children: [{ name: "Checkout.handle", value: 100, children: [{ name: "Checkout.parse", value: 10 }] }],
      }}
    />,
  );

  const child = screen.getByRole("button", { name: /Checkout\.parse/ });
  expect(child).toHaveStyle({ width: "10%" });
});

test("keeps flamegraph context when no frames match the search", () => {
  render(<Flamegraph root={{ name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] }} />);

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "DemoHttpService" } });

  expect(screen.getByRole("button", { name: /Checkout\.handle/ })).toHaveClass("flame-row-dimmed");
  expect(screen.getByText(/Search highlights matching frames/)).toBeInTheDocument();
});

test("hides native and runtime frames without removing application frames", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 30,
        children: [
          { name: "libc.so.6", value: 10 },
          { name: "java.lang.Thread.run", value: 10 },
          { name: "com.acme.Checkout.handle:42", value: 10 },
        ],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: "Hide Native" }));

  expect(screen.queryByRole("button", { name: /libc\.so/ })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /Thread\.run/ })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Checkout\.handle/ })).toBeInTheDocument();
});

test("shows a custom empty state when there are no samples", () => {
  render(<Flamegraph root={{ name: "service", value: 0, children: [] }} emptyMessage="No allocation samples returned." />);

  expect(screen.getByText("No allocation samples returned.")).toBeInTheDocument();
});
