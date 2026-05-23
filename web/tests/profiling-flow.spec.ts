import { test, expect } from "@playwright/test";

test("service diagnosis surface loads", async ({ page }) => {
  await page.route("**/api/ui/v1/flamegraph?**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        root: {
          name: "root",
          value: 100,
          children: [
            {
              name: "com/example/DemoHttpService.handleWork:93",
              value: 80,
              children: [{ name: "com/example/DemoHttpService.burnCpu:188", value: 60 }],
            },
          ],
        },
        metadata: { partial: false },
      }),
    });
  });
  await page.route("**/api/ui/v1/top-stacks?**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          symbol: "DemoHttpService.handleWork",
          location: "DemoHttpService:93",
          profile_type: "java_cpu_nanoseconds",
          self: 20,
          total: 80,
          self_display: "20 ns",
          total_display: "80 ns",
          self_percent: "20.0%",
          total_percent: "80.0%",
        },
      ]),
    });
  });

  await page.goto("/");
  await expect(page.getByText("Java Profiler")).toBeVisible();
  await expect(page.getByRole("button", { name: "CPU profiles", exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "CPU profile analysis" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Single Pod CPU profile" })).toBeVisible();
  await expect(page.getByPlaceholder("Search frame")).toBeVisible();
  const flameRows = page.locator(".flame-row");
  await flameRows.filter({ hasText: /DemoHttpService\.handleWork:93/ }).first().click();
  const focusState = page.getByRole("region", { name: "Focused flamegraph state" });
  await expect(focusState).toContainText("Focused:");
  await expect(focusState).toContainText(/DemoHttpService\.handleWork/);
  await flameRows.filter({ hasText: /DemoHttpService\.burnCpu:188/ }).first().click();
  await expect(focusState).toContainText(/DemoHttpService\.burnCpu/);
  await focusState.getByRole("button", { name: "Back" }).click();
  await expect(focusState).toContainText(/DemoHttpService\.handleWork/);
  await focusState.getByRole("button", { name: "Reset" }).click();
  await expect(focusState).toBeHidden();
});
