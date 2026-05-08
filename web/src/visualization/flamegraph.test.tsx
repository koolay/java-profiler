import { fireEvent, render, screen } from "@testing-library/react";
import { Flamegraph } from "./flamegraph";

test("renders flamegraph frames and partial warning", () => {
  render(<Flamegraph root={{ name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] }} metadata={{ partial: true, reasons: ["node_limit"] }} />);
  expect(screen.getByText("Checkout.handle")).toBeInTheDocument();
  expect(screen.getByText(/Partial result/)).toBeInTheDocument();
});

test("zooms nested frames and keeps long labels available", () => {
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
  expect(screen.getByText("BusyApp.lambda$main$0:14")).toBeInTheDocument();

  fireEvent.change(screen.getByLabelText("Search flamegraph frames"), { target: { value: "busyapp" } });
  expect(screen.getByText("BusyApp.lambda$main$0:14").closest("button")).toHaveClass("flame-row-match");

  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  expect(screen.getByText("java/lang/Thread.run")).toBeInTheDocument();
});
