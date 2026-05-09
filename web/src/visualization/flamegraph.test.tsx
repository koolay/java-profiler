import { fireEvent, render, screen, within } from "@testing-library/react";
import { Flamegraph } from "./flamegraph";

test("renders flamegraph frames and partial warning", () => {
  render(<Flamegraph root={{ name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] }} metadata={{ partial: true, reasons: ["node_limit"] }} />);
  expect(screen.getByText("Checkout.handle")).toBeInTheDocument();
  expect(screen.getByText(/Partial result/)).toBeInTheDocument();
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
  expect(nativeFrame).toHaveAttribute("title", "libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis: 8");

  fireEvent.click(nativeFrame);
  const detail = screen.getByLabelText("Selected flamegraph frame");
  expect(within(detail).getByText("libjvm.so.VeryLongNativeFrameNameThatWillNeedEllipsis")).toBeInTheDocument();
  expect(within(detail).getByText("66.7%")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Zoom selected" }));
  expect(screen.getByText("BusyApp.lambda$main$0:14")).toBeInTheDocument();

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "busyapp" } });
  expect(screen.getByText("BusyApp.lambda$main$0:14").closest("button")).toHaveClass("flame-row-match");

  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(screen.getByText("java/lang/Thread.run")).toBeInTheDocument();
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
  expect(within(detail).getByText("10.6%")).toBeInTheDocument();
});
