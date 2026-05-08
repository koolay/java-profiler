import { render, screen } from "@testing-library/react";
import { Flamegraph } from "./flamegraph";

test("renders flamegraph frames and partial warning", () => {
  render(<Flamegraph root={{ name: "root", value: 10, children: [{ name: "Checkout.handle", value: 10 }] }} metadata={{ partial: true, reasons: ["node_limit"] }} />);
  expect(screen.getByText("Checkout.handle")).toBeInTheDocument();
  expect(screen.getByText(/Partial result/)).toBeInTheDocument();
});
