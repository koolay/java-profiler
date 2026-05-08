import { render, screen } from "@testing-library/react";
import { TargetStatusView } from "./target-status-view";

test("renders target status reasons", () => {
  render(<TargetStatusView params={new URLSearchParams()} statuses={[{ desired_state: "disabled", reason: "unsupported_jvm", message: "OpenJ9 skipped" }]} />);
  expect(screen.getByText("unsupported_jvm")).toBeInTheDocument();
  expect(screen.getByText("OpenJ9 skipped")).toBeInTheDocument();
});
