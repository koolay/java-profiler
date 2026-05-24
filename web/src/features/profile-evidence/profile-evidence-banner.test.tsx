import { render, screen } from "@testing-library/react";
import { test } from "vitest";
import { ProfileEvidenceBanner } from "./profile-evidence-banner";

test("hides when samples are available", () => {
  render(<ProfileEvidenceBanner evidence={{ state: "has_samples", message: "ok" }} />);

  expect(screen.queryByRole("status", { name: "Profile evidence status" })).not.toBeInTheDocument();
});

test("renders status and freshness details", () => {
  render(
    <ProfileEvidenceBanner
      evidence={{
        state: "temporary_expired",
        message: "No profile samples were found because temporary profiling has expired for this target.",
        latestProfileBatchAt: "2026-05-24T12:30:00Z",
        latestStatusAt: "2026-05-24T12:00:00Z",
        statusReason: "temporary_expired",
      }}
    />,
  );

  expect(screen.getByText(/temporary profiling has expired/)).toBeInTheDocument();
  expect(screen.getByText(/Latest profile ingestion: 2026-05-24T12:30:00Z/)).toBeInTheDocument();
  expect(screen.getByText(/Latest target status: 2026-05-24T12:00:00Z/)).toBeInTheDocument();
});
