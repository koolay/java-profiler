import { test, expect } from "@playwright/test";

test("service diagnosis surface loads", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Service diagnosis" })).toBeVisible();
  await expect(page.getByRole("button", { name: "memory" })).toBeVisible();
});
