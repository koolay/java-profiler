import { fireEvent, render, screen, within } from "@testing-library/react";
import { act } from "react";
import { afterEach, vi } from "vitest";
import { Flamegraph } from "./flamegraph";

afterEach(() => {
  vi.useRealTimers();
});

test("renders flamegraph frames and partial warning", () => {
  render(<Flamegraph root={{ name: "root", value: 14, children: [{ name: "Checkout.handle", value: 10 }, { name: "java.lang.Thread.run", value: 4 }] }} metadata={{ partial: true, reasons: ["node_limit"] }} />);
  expect(screen.getByText("Checkout.handle")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Checkout\.handle/ })).toHaveClass("flame-row-application");
  expect(screen.getByRole("button", { name: /Thread\.run/ })).toHaveClass("flame-row-runtime");
  expect(screen.getByText(/partial flamegraph/i)).toBeInTheDocument();
  const legend = screen.getByLabelText("Frame categories");
  expect(within(legend).getByText("Application Java")).toBeInTheDocument();
  expect(within(legend).getByText("JVM/runtime")).toBeInTheDocument();
  expect(within(legend).getByText("Native/system")).toBeInTheDocument();
});

test("inspects, searches, and zooms nested frames while keeping long labels readable", () => {
  const { container } = render(
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
  expect(within(inspector).getByText("Total CPU")).toBeInTheDocument();
  expect(within(inspector).getByText("Self CPU")).toBeInTheDocument();

  fireEvent.click(nativeFrame);
  const detail = screen.getByLabelText("Selected flamegraph frame");
  expect(within(detail).getByText("libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis")).toBeInTheDocument();
  expect(detail).toHaveTextContent("100.0%");
  expect(within(detail).getAllByText("Self CPU")).toHaveLength(1);

  const busyFrame = screen.getByRole("button", { name: /BusyApp\.lambda\$main\$0:14/ });
  fireEvent.click(busyFrame);
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("BusyApp.lambda$main$0:14");
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("Focused:");
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("66.7% of profile");
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toContainElement(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toContainElement(screen.getByRole("button", { name: "Reset" }));
  const focusPath = screen.getByRole("navigation", { name: "Focused flamegraph path" });
  expect(within(focusPath).getByRole("button", { name: "root" })).toBeEnabled();
  expect(within(focusPath).getByRole("button", { name: /libjvm\.so VeryLongNativeFrameNameThatWillNeedEllipsis/ })).toBeEnabled();
  expect(within(focusPath).getByRole("button", { name: /BusyApp\.lambda\$main\$0:14/ })).toBeDisabled();

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "busyapp" } });
  expect(screen.getByLabelText("Search flamegraph frames")).toHaveValue("busyapp");
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("BusyApp.lambda$main$0:14");
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("BusyApp.lambda$main$0:14");
  fireEvent.click(within(screen.getByRole("navigation", { name: "Focused flamegraph path" })).getByRole("button", { name: /libjvm\.so VeryLongNativeFrameNameThatWillNeedEllipsis/ }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis");
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).not.toHaveTextContent("BusyApp.lambda$main$0:14");

  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(screen.getByRole("button", { name: /Thread\.run/ })).toBeInTheDocument();
});

test("focuses the frame you click", () => {
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

  const hoveredFrame = screen.getByRole("button", { name: /HoveredFrame\.method:20/ });
  fireEvent.click(hoveredFrame);

  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("HoveredFrame.method:20");
  expect(screen.getByRole("navigation", { name: "Focused flamegraph path" })).toHaveTextContent("HoveredFrame.method:20");
});

test("returns to the selected frame after hover inspection expires", () => {
  vi.useFakeTimers();
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

  const hoveredFrame = screen.getByText("HoveredFrame.method:20").closest("button");
  expect(hoveredFrame).not.toBeNull();
  fireEvent.mouseEnter(hoveredFrame!);
  fireEvent.mouseLeave(hoveredFrame!);
  act(() => vi.advanceTimersByTime(130));
  expect(screen.getByLabelText("Selected flamegraph frame")).toHaveTextContent("root");
  expect(screen.getByLabelText("Selected flamegraph frame")).not.toHaveTextContent("HoveredFrame.method:20");
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

  expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ }));
  expect(screen.getByRole("button", { name: "Back" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: "root 8" })).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByRole("button", { name: "root 8" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
});

test("walks back through consecutive focused frames and reset returns to root", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 100,
        children: [
          {
            name: "DemoHttpService.handleWork:93",
            value: 80,
            children: [{ name: "DemoHttpService.burnCpu:188", value: 60, children: [{ name: "Math.sqrt", value: 40 }] }],
          },
        ],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("DemoHttpService.handleWork:93");

  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.burnCpu:188/ }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("DemoHttpService.burnCpu:188");

  fireEvent.click(screen.getByRole("button", { name: "Back" }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("DemoHttpService.handleWork:93");

  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(screen.queryByRole("region", { name: "Focused flamegraph state" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /^root 100/ })).toBeInTheDocument();
});

test("clears focus state when a refreshed profile no longer contains the focused path", () => {
  const { rerender } = render(
    <Flamegraph
      root={{
        name: "root",
        value: 10,
        children: [{ name: "Checkout.handle", value: 10, children: [{ name: "Checkout.parse", value: 10 }] }],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /Checkout\.handle/ }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("Checkout.handle");

  rerender(<Flamegraph root={{ name: "root", value: 6, children: [{ name: "Checkout.other", value: 6 }] }} />);

  expect(screen.queryByRole("region", { name: "Focused flamegraph state" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /^root 6/ })).toBeInTheDocument();
});

test("clears focus when a refreshed profile reuses the same path and leaf name under another ancestor", () => {
  const { rerender } = render(
    <Flamegraph
      root={{
        name: "root",
        value: 10,
        children: [{ name: "Checkout.alpha", value: 10, children: [{ name: "SharedFrame.run", value: 10 }] }],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /SharedFrame\.run/ }));
  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("SharedFrame.run");

  rerender(
    <Flamegraph
      root={{
        name: "root",
        value: 10,
        children: [{ name: "Checkout.beta", value: 10, children: [{ name: "SharedFrame.run", value: 10 }] }],
      }}
    />,
  );

  expect(screen.queryByRole("region", { name: "Focused flamegraph state" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /^root 10/ })).toBeInTheDocument();
});

test("uses child samples as profile basis when the root has no own value", () => {
  render(
    <Flamegraph
      root={{
        name: "root",
        value: 0,
        children: [
          { name: "Checkout.handle", value: 30 },
          { name: "Checkout.parse", value: 70 },
        ],
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: /Checkout\.handle/ }));

  expect(screen.getByRole("region", { name: "Focused flamegraph state" })).toHaveTextContent("30.0% of profile");
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
  expect(detail).toHaveTextContent("100.0%");
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
  expect(within(screen.getByLabelText("Selected flamegraph frame")).getAllByText("7")).toHaveLength(1);

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
