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

test("real cluster Java profiling workbench exposes status, CPU, Wall Clock, I/O, GC, deadlock, and ingestion surfaces", async ({ page }) => {
  const consoleMessages: string[] = [];
  page.on("console", (message) => consoleMessages.push(`[${message.type()}] ${message.text()}`));
  page.on("pageerror", (error) => consoleMessages.push(`[pageerror] ${error.message}`));

  await page.goto(baseURL, { waitUntil: "networkidle" });
  await expect(page.getByText("Java Profiler")).toBeVisible();
  await expect(page.getByRole("button", { name: "CPU profiles", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "GC pauses", exact: true })).toBeVisible();

  await page.getByRole("textbox", { name: "Namespace", exact: true }).fill(namespace);
  await page.getByRole("textbox", { name: "Service", exact: true }).fill(service);
  await page.getByRole("combobox", { name: "Range" }).selectOption("60");

  await page.getByRole("button", { name: "Service status", exact: true }).click();
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

  await page.getByRole("button", { name: "CPU profiles", exact: true }).click();
  const analysis = page.getByRole("region", { name: "CPU profile analysis" });
  await expect(analysis.getByRole("heading", { name: "Single Pod CPU profile" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Symbol", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Self CPU" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Total CPU" })).toBeVisible();
  const topTable = page.getByRole("region", { name: "Top table" });
  const firstDataRow = topTable.locator("tbody tr").first();
  const topFrameButton = firstDataRow.getByRole("button").first();
  await expect(topFrameButton).toBeVisible();
  await expect(firstDataRow).not.toContainText(/(^|\b)(so\.6|libjvm|pthread|clock_gettime|\[vdso\])(\b|$)/i);
  await topFrameButton.click();
  await expect(page.getByPlaceholder("Search frame")).toBeVisible();
  await expect(page.getByPlaceholder("Search frame")).toHaveValue("");
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  await expect(page.getByText(/Full sampled stack context/)).toBeVisible();
  const legend = page.getByLabel("Frame categories");
  await expect(legend.getByText("Application Java")).toBeVisible();
  await expect(legend.getByText("JVM/runtime")).toBeVisible();
  await expect(legend.getByText("Native/system")).toBeVisible();
  const nativeFrame = page.getByRole("button", { name: /so\.6|libjvm|pthread|\[vdso\]/i }).first();
  const hasNativeFrame = await nativeFrame
    .waitFor({ state: "visible", timeout: 2_000 })
    .then(() => true)
    .catch(() => false);
  if (hasNativeFrame) {
    await expect(nativeFrame).not.toHaveClass(/flame-row-dimmed/);
  }
  await topFrameButton.click();
  const inspector = page.getByRole("status");
  await expect(inspector).toContainText(/Application Java|Native\/system|JVM\/runtime/);
  await expect(inspector).toContainText(/Total CPU/);
  await expect(inspector).toContainText(/Self CPU/);
  await expect(inspector.getByRole("button", { name: "Focus frame" })).toBeVisible();
  await expect(inspector.getByRole("button", { name: "Copy frame" })).toBeVisible();
  await expect(inspector.getByRole("button", { name: "Permalink" })).toBeVisible();
  await page.getByPlaceholder("Search frame").fill("burnCpu");
  await expect(page.getByText(/Search highlights matching frames/)).toBeVisible();
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  if (hasNativeFrame) {
    await expect(nativeFrame).toHaveClass(/flame-row-dimmed/);
  }
  const flamegraph = page.getByRole("region", { name: "Flamegraph", exact: true });
  const focusState = page.getByRole("region", { name: "Focused flamegraph state" });
  const focusPath = page.getByRole("navigation", { name: "Focused flamegraph path" });
  const focusedRows = flamegraph.locator(".flame-row");
  const rowCount = await focusedRows.count();
  expect(rowCount).toBeGreaterThan(1);
  await focusedRows.nth(1).click();
  await page.getByRole("button", { name: "Focus frame" }).click();
  await expect(page.getByText(/Focused stack context/)).toBeVisible();
  await expect(focusState).toContainText("Focused:");
  await expect(focusPath).toContainText("Focused");
  await expect(focusState.getByRole("button", { name: "Back" })).toBeEnabled();
  await expect(focusState.getByRole("button", { name: "Reset" })).toBeVisible();
  const initialFocusName = await focusPath.locator("code").last().textContent();
  const nextFocusIndex = rowCount > 2 ? 2 : 1;
  await focusedRows.nth(nextFocusIndex).click();
  await page.getByRole("button", { name: "Focus frame" }).click();
  await expect(focusPath.locator("code").last()).not.toHaveText(initialFocusName ?? "");
  await focusState.getByRole("button", { name: "Back" }).click();
  await expect(focusPath.locator("code").last()).toHaveText(initialFocusName ?? "");
  await focusState.getByRole("button", { name: "Reset" }).click();
  await expect(focusState).toBeHidden();
  await expect(focusPath).toBeHidden();
  await page.getByRole("button", { name: "Reset view" }).click();
  await page.getByRole("button", { name: "Top Table" }).click();
  await expect(page.getByRole("region", { name: "Top table" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Flamegraph", exact: true })).toBeHidden();
  await page.getByRole("button", { name: "Both" }).click();
  await expect(page.getByRole("region", { name: "Top table" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Flamegraph", exact: true })).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-02-cpu.png`, fullPage: true });

  await page.getByRole("button", { name: "Allocation profiles", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Allocation sources" })).toBeVisible();
  await expect(page.getByText("Loading profile evidence.")).toBeHidden();
  await expect(page.getByRole("button", { name: /^root\s+\d/ })).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-03-memory.png`, fullPage: true });

  await page.getByRole("button", { name: "Wall Clock profiles", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Single Pod Wall Clock profile" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Top table" })).toContainText(/BusyApp|DemoHttpService/);
  await page.screenshot({ path: `${artifactDir}/ui-04-wall-clock.png`, fullPage: true });

  await page.getByRole("button", { name: "I/O wait profiles", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Single Pod I/O wait profile" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Top table" })).toContainText(/BusyApp|DemoHttpService/);
  await page.screenshot({ path: `${artifactDir}/ui-05-io.png`, fullPage: true });

  await page.getByRole("button", { name: "GC pauses", exact: true }).click();
  await expect(page.getByRole("heading", { name: "GC pauses" })).toBeVisible();
  await expect(page.getByText(/JVM GC|gc_pause|Allocation correlation/).first()).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-06-gc.png`, fullPage: false });

  await page.getByRole("button", { name: "Deadlock diagnosis", exact: true }).click();
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
  await page.screenshot({ path: `${artifactDir}/ui-07-deadlocks.png`, fullPage: true });

  await page.getByRole("button", { name: "Ingestion health", exact: true }).click();
  const ingestion = page.getByRole("region", { name: "Ingestion health" });
  await expect(ingestion).toBeVisible();
  await expect(ingestion.getByText("Loading ingestion evidence.")).toBeHidden();
  await expect(ingestion.getByText(/accepted x [1-9]\d*/i).first()).toBeVisible();
  await page.screenshot({ path: `${artifactDir}/ui-08-ingestion.png`, fullPage: true });

  await test.info().attach("browser-console", {
    body: consoleMessages.join("\n") || "no browser console messages",
    contentType: "text/plain",
  });
});
