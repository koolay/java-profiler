import { fireEvent, render, screen } from "@testing-library/react";
import { TargetStatusView } from "./target-status-view";

test("filters to accepted Java targets by default and can show all statuses", () => {
  render(
    <TargetStatusView
      params={new URLSearchParams()}
      statuses={[
        { desired_state: "disabled", reason: "unsupported_jvm", message: "OpenJ9 skipped", target: { pod: "not-java" } },
        { desired_state: "temporary", reason: "accepted", message: "HotSpot-compatible JVM profiling session active", target: { pod: "java-demo" } },
      ]}
    />,
  );

  expect(screen.getByText("java-demo")).toBeInTheDocument();
  expect(screen.queryByText("not-java")).not.toBeInTheDocument();

  fireEvent.click(screen.getByLabelText("Java targets only"));
  expect(screen.getByText("not-java")).toBeInTheDocument();
});

test("keeps long pod names and user actions inspectable", () => {
  render(
    <TargetStatusView
      params={new URLSearchParams()}
      statuses={[
        {
          desired_state: "temporary",
          reason: "accepted",
          message: "HotSpot-compatible JVM profiling session active",
          target: { pod: "async-profiler-lab-845cc49cc-plpf6", process_id: 3749150 },
        },
      ]}
    />,
  );

  expect(screen.getByText("async-profiler-lab-845cc49cc-plpf6")).toHaveAttribute("title", "async-profiler-lab-845cc49cc-plpf6");
  expect(screen.getByText("Open CPU, memory, locks, or thread views for the same target and time range.")).toBeInTheDocument();
});
