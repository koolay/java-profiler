import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { TargetStatusView } from "./target-status-view";

test("explains why Java targets are hidden when the default filter removes every row", async () => {
  render(
    <TargetStatusView
      params={new URLSearchParams()}
      statuses={[
        { desired_state: "disabled", reason: "disabled_by_metadata", message: "profiling not enabled", target: { pod: "zookeeper-0", process_id: 90929 } },
        { desired_state: "disabled", reason: "unsupported_jvm", message: "OpenJ9 skipped", target: { pod: "legacy-0", process_id: 42 } },
      ]}
    />,
  );

  await waitFor(() =>
    expect(
      screen.getByText("No enabled Java targets are visible in this scope. 1 disabled_by_metadata, 1 unsupported_jvm. Uncheck \"Java targets only\" to inspect the blocked targets."),
    ).toBeInTheDocument(),
  );
  expect(screen.queryByText("zookeeper-0")).not.toBeInTheDocument();

  fireEvent.click(screen.getByLabelText("Java targets only"));

  await waitFor(() => expect(screen.getByText("zookeeper-0")).toBeInTheDocument());
  await waitFor(() => expect(screen.getByText("legacy-0")).toBeInTheDocument());
});
