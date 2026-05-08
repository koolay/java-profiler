import { expect, test } from "@playwright/test";

declare const process: { env: Record<string, string | undefined> };

const enabled = process.env.REAL_ACCEPTANCE === "1";
const baseURL = process.env.REAL_ACCEPTANCE_BASE_URL ?? "http://127.0.0.1:18081";
const namespace = process.env.REAL_ACCEPTANCE_NAMESPACE ?? "java-profiler-qa";
const service = process.env.REAL_ACCEPTANCE_SERVICE ?? "checkout-java";
const artifactDir = process.env.REAL_ACCEPTANCE_ARTIFACT_DIR ?? "/tmp/java-profiler-real-acceptance-ui";

test.skip(!enabled, "Set REAL_ACCEPTANCE=1 to run against a real deployed cluster UI.");
test.use({ video: "on", screenshot: "only-on-failure" });

test("real cluster service diagnosis flow exposes status, profile, deadlock, and ingestion surfaces", async ({ page }) => {
  const consoleMessages: string[] = [];
  page.on("console", (message) => consoleMessages.push(`[${message.type()}] ${message.text()}`));
  page.on("pageerror", (error) => consoleMessages.push(`[pageerror] ${error.message}`));

  await page.goto(baseURL, { waitUntil: "networkidle" });
  await expect(page.getByRole("heading", { name: "Service diagnosis" })).toBeVisible();

  await page.getByRole("textbox", { name: "Namespace", exact: true }).fill(namespace);
  await page.getByRole("textbox", { name: "Service", exact: true }).fill(service);
  await page.getByLabel("Range").selectOption("60");

  await page.getByRole("button", { name: "status", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Target status" })).toBeVisible();
  await expect(page.getByRole("cell", { name: /accepted|unsupported_jvm|temporary_expired|disabled_by_metadata/ }).first()).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-01-status.png`, fullPage: true });

  await page.getByRole("button", { name: "cpu", exact: true }).click();
  await expect(page.getByPlaceholder("Search frame")).toBeVisible();
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-02-cpu.png`, fullPage: true });

  await page.getByRole("button", { name: "deadlocks", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Deadlock cycles" })).toBeVisible();
  await expect(page.getByText("No deadlock cycles returned for this service and time range.")).toBeHidden();
  await expect(page.getByText(/-cycle$/).first()).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-03-deadlocks.png`, fullPage: true });

  await page.getByRole("button", { name: "ingestion", exact: true }).click();
  const ingestion = page.getByRole("region", { name: "Ingestion health" });
  await expect(ingestion).toBeVisible();
  await expect(ingestion.getByText("Loading ingestion evidence.")).toBeHidden();
  await expect(ingestion.getByText(/accepted x [1-9]\d*/i).first()).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-04-ingestion.png`, fullPage: true });

  await test.info().attach("browser-console", {
    body: consoleMessages.join("\n") || "no browser console messages",
    contentType: "text/plain",
  });
});
