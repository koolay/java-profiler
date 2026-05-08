import { render, screen } from "@testing-library/react";
import { DeadlocksView } from "./deadlocks-view";

test("renders deadlock cycle details", () => {
  render(<DeadlocksView params={new URLSearchParams()} events={[{ event_id: "1", cycle_id: "cycle", involved_threads: ["a", "b"], blocking_frames: ["A.lock"] }]} />);
  expect(screen.getByText("cycle")).toBeInTheDocument();
  expect(screen.getByText("a -> b")).toBeInTheDocument();
});
