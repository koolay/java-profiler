import { render, screen } from "@testing-library/react";
import { IngestionHealthView } from "./ingestion-health-view";

test("shows ingestion loss totals and latest retry or rejection message", () => {
  render(
    <IngestionHealthView
      health={{
        totals: {
          accepted: 2,
          duplicate: 0,
          retryable: 1,
          rejected: 1,
          dropped_samples: 5,
          dropped_stacks: 3,
          truncated_batches: 1,
        },
        batches: [
          {
            batch_type: "profile",
            status: "retryable",
            retryable: true,
            count: 1,
            latest_at: "2026-05-10T00:00:00Z",
            last_message: "storage unavailable",
          },
        ],
        partial: false,
      }}
    />,
  );

  expect(screen.getByText("accepted batches")).toBeInTheDocument();
  expect(screen.getByText("retryable batches")).toBeInTheDocument();
  expect(screen.getByText("rejected batches")).toBeInTheDocument();
  expect(screen.getByText("dropped samples")).toBeInTheDocument();
  expect(screen.getByText("dropped stacks")).toBeInTheDocument();
  expect(screen.getByText("truncated batches")).toBeInTheDocument();
  expect(screen.getAllByText("storage unavailable").length).toBeGreaterThan(0);
});
