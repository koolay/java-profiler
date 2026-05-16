import { expect, test } from "@playwright/test";

declare const process: { env: Record<string, string | undefined> };

const enabled = process.env.REAL_ACCEPTANCE === "1";
const baseURL = process.env.REAL_ACCEPTANCE_BASE_URL ?? "http://127.0.0.1:18081";
const namespace = process.env.REAL_ACCEPTANCE_NAMESPACE ?? "java-profiler-qa";
const service = process.env.REAL_ACCEPTANCE_SERVICE ?? "checkout-java";
const artifactDir = process.env.REAL_ACCEPTANCE_ARTIFACT_DIR ?? "/tmp/java-profiler-real-acceptance-ui";
const requireDeadlockEvidence = process.env.REAL_ACCEPTANCE_REQUIRE_DEADLOCK === "1";

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
  const statusCell = page.getByRole("cell", { name: /accepted|unsupported_jvm|temporary_expired|disabled_by_metadata/ }).first();
  const hasFilteredJavaStatus = await statusCell
    .waitFor({ state: "visible", timeout: 2_000 })
    .then(() => true)
    .catch(() => false);
  if (!hasFilteredJavaStatus) {
    await expect(page.getByText(/No matching targets/)).toBeVisible();
    const javaOnlyFilter = page.getByRole("checkbox", { name: "Java targets only" });
    if (await javaOnlyFilter.isChecked()) {
      await javaOnlyFilter.uncheck();
    }
    const anyStatusCell = page.getByRole("cell", { name: /accepted|unsupported_jvm|temporary_expired|disabled_by_metadata/ }).first();
    const hasAnyStatus = await anyStatusCell
      .waitFor({ state: "visible", timeout: 2_000 })
      .then(() => true)
      .catch(() => false);
    if (!hasAnyStatus) {
      await expect(page.getByText(/No matching targets/)).toBeVisible();
    }
  }
  await page.screenshot({ path: `${artifactDir}/ui-01-status.png`, fullPage: true });

  await page.getByRole("button", { name: "cpu", exact: true }).click();
  const analysis = page.getByRole("region", { name: "CPU profile analysis" });
  await expect(analysis.getByRole("heading", { name: "CPU profile" })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "Symbol" })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "Self CPU" })).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "Total CPU" })).toBeVisible();
  const topTable = page.getByRole("region", { name: "Top table" });
  await expect(topTable.getByRole("button", { name: /DemoHttpService\.handleWork/ }).first()).toBeVisible();
  const firstDataRow = topTable.locator("tbody tr").first();
  await expect(firstDataRow).not.toContainText(/(^|\b)(so\.6|libjvm|pthread|clock_gettime|\[vdso\])(\b|$)/i);
  await topTable.getByRole("button", { name: /DemoHttpService\.burnCpu/ }).first().click();
  await expect(page.getByPlaceholder("Search frame")).toBeVisible();
  await expect(page.getByPlaceholder("Search frame")).toHaveValue("");
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  await expect(page.getByText(/Full sampled stack context/)).toBeVisible();
  await expect(page.getByText(/start from DemoHttpService|Start by inspecting this method|inspect both DemoHttpService/)).toBeVisible();
  const legend = page.getByLabel("Frame categories");
  await expect(legend.getByText("Application Java")).toBeVisible();
  await expect(legend.getByText("JVM/runtime")).toBeVisible();
  await expect(legend.getByText("Native/system")).toBeVisible();
  await expect(page.getByRole("button", { name: /so\.6/ }).first()).not.toHaveClass(/flame-row-dimmed/);
  const demoFrame = page.getByRole("button", { name: /DemoHttpService\.(burnCpu|handleWork)/ }).first();
  await expect(demoFrame).toBeVisible();
  await demoFrame.click();
  const selectedFrame = page.getByRole("region", { name: "Selected flamegraph frame" });
  await expect(selectedFrame).toContainText(/DemoHttpService\.(burnCpu|handleWork)/);
  await expect(selectedFrame).toContainText(/Samples/);
  await expect(selectedFrame).toContainText(/Total CPU/);
  await expect(selectedFrame).toContainText(/Self CPU/);
  const inspector = page.getByRole("status");
  await expect(inspector).toContainText(/Application Java|Native\/system|JVM\/runtime/);
  await expect(inspector).toContainText(/Total CPU/);
  await expect(inspector).toContainText(/Self CPU/);
  await page.getByPlaceholder("Search frame").fill("burnCpu");
  await expect(page.getByText(/Search highlights matching frames/)).toBeVisible();
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /so\.6/ }).first()).toHaveClass(/flame-row-dimmed/);
  await page.getByRole("button", { name: "Focus selected" }).click();
  await expect(page.getByText(/Focused stack context/)).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Focused flamegraph path" })).toContainText("Focused");
  await expect(page.getByRole("button", { name: "Back" })).toBeEnabled();
  await page.getByRole("button", { name: "Back" }).click();
  await expect(page.getByText(/Search highlights matching frames/)).toBeVisible();
  await page.getByRole("button", { name: "Reset", exact: true }).click();
  await page.getByRole("button", { name: "Top Table" }).click();
  await expect(page.getByRole("region", { name: "Top table" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Flamegraph", exact: true })).toBeHidden();
  await page.getByRole("button", { name: "Both" }).click();
  await expect(page.getByRole("region", { name: "Top table" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Flamegraph", exact: true })).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-02-cpu.png`, fullPage: true });

  await page.getByRole("button", { name: "deadlocks", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Deadlock cycles" })).toBeVisible();
  if (requireDeadlockEvidence) {
    await expect(page.getByText("No deadlock cycles returned for this service and time range.")).toBeHidden();
    await expect(page.getByText(/-cycle$/).first()).toBeVisible();
  } else {
    const emptyDeadlockState = page.getByText("No deadlock cycles returned for this service and time range.");
    const deadlockCycle = page.getByText(/-cycle$/).first();
    const hasDeadlockCycle = await deadlockCycle
      .waitFor({ state: "visible", timeout: 2_000 })
      .then(() => true)
      .catch(() => false);
    if (!hasDeadlockCycle) {
      await expect(emptyDeadlockState).toBeVisible();
    }
  }
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
