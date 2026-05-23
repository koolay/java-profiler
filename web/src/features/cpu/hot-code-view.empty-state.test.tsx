import { render, screen } from "@testing-library/react";
import { HotCodeView } from "./hot-code-view";

test("uses the custom flamegraph empty message when no frames are available", () => {
  render(
    <HotCodeView
      root={{ name: "service", value: 0, children: [] } as any}
      metadata={{ partial: false }}
      flamegraphEmptyMessage="No wall-clock flamegraph samples returned for this service and time range. The top table may still surface correlated Java methods."
    />,
  );

  expect(
    screen.getByText("No wall-clock flamegraph samples returned for this service and time range. The top table may still surface correlated Java methods."),
  ).toBeInTheDocument();
  expect(screen.queryByText("No application Java frames were found in this profile. Use the flame graph to inspect runtime or native frames.")).not.toBeInTheDocument();
});

