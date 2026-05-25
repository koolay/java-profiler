import { mkdirSync, readdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import playwright from "../web/node_modules/playwright/index.js";
import playwrightTest from "../web/node_modules/@playwright/test/index.js";

const { chromium } = playwright;
const { expect } = playwrightTest;

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, "..");
const outDir = process.env.DOC_SCREENSHOT_DIR ?? path.join(repoRoot, "docs/assets/screenshots");
const baseURL = process.env.REAL_ACCEPTANCE_BASE_URL ?? "http://127.0.0.1:18081";
const namespace = process.env.REAL_ACCEPTANCE_NAMESPACE ?? "java-profiler-qa";
const service = process.env.REAL_ACCEPTANCE_SERVICE ?? "jdk17-http-demo";
const pod = process.env.REAL_ACCEPTANCE_POD ?? "";

mkdirSync(outDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({
  viewport: { width: 1440, height: 1000 },
  deviceScaleFactor: 1,
});

const consoleMessages = [];
page.on("console", (message) => consoleMessages.push(`[${message.type()}] ${message.text()}`));
page.on("pageerror", (error) => consoleMessages.push(`[pageerror] ${error.message}`));

async function clickNav(label) {
  await page.locator("button").filter({ hasText: label }).first().click();
}

async function expectProfileEvidenceOrTopRow(viewName) {
  const topRow = page.getByRole("region", { name: "Top table" }).locator("tbody button").first();
  const rowVisible = await topRow
    .waitFor({ state: "visible", timeout: 5_000 })
    .then(() => true)
    .catch(() => false);
  if (rowVisible) {
    return topRow;
  }
  await expect(page.getByRole("status", { name: "Profile evidence status" })).toBeVisible();
  throw new Error(`${viewName} has no non-empty profile rows; screenshot capture requires real profile evidence.`);
}

try {
  await page.goto(baseURL, { waitUntil: "domcontentloaded" });
  await expect(page.getByText("Incident workbench")).toBeVisible();
  await page.getByPlaceholder("All namespaces").fill(namespace);
  await page.getByPlaceholder("All services").fill(service);
  if (pod) {
    await page.getByPlaceholder("single Java Pod").fill(pod);
  }
  await page.getByRole("button", { name: "1h", exact: true }).click();

  await clickNav("Targets");
  await expect(page.getByRole("heading", { name: "Target status" })).toBeVisible();
  await page.getByLabel("Java targets only").uncheck();
  await expect(page.getByRole("cell", { name: /accepted|unsupported_jvm|temporary_expired|disabled_by_metadata/ }).first()).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-target-status.png"), fullPage: true });

  await clickNav("CPU");
  await expect(page.getByRole("heading", { name: "Single Pod CPU profile" })).toBeVisible();
  const cpuFrame = await expectProfileEvidenceOrTopRow("CPU");
  await cpuFrame.click();
  await expect(page.getByRole("region", { name: /Selected (hot Java|flamegraph) frame/ })).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-cpu-analysis.png"), fullPage: true });

  await clickNav("Alloc");
  await expect(page.getByRole("heading", { name: "Allocation sources" })).toBeVisible();
  await expect(page.locator('[aria-label="Allocation summary"]')).toBeVisible();
  await expect(page.getByRole("region", { name: "Top allocating paths" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Top self allocating frames" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Flamegraph" })).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-allocation-analysis.png"), fullPage: true });

  await clickNav("Latency");
  await expect(page.getByRole("heading", { name: "Single Pod Wall Clock profile" })).toBeVisible();
  await expectProfileEvidenceOrTopRow("Wall Clock");
  await page.screenshot({ path: path.join(outDir, "real-wall-clock.png"), fullPage: true });

  await clickNav("I/O");
  await expect(page.getByRole("heading", { name: "Single Pod I/O wait profile" })).toBeVisible();
  await expectProfileEvidenceOrTopRow("I/O wait");
  await page.screenshot({ path: path.join(outDir, "real-io-wait.png"), fullPage: true });

  await clickNav("GC");
  await expect(page.getByRole("heading", { name: "GC pauses" })).toBeVisible();
  await expect(page.locator('[aria-label="GC summary"]')).toBeVisible();
  const gcEventRow = page.locator(".gc-event-row").first();
  const gcEmptyState = page.getByText("No GC pause event evidence in this range.");
  const gcEventVisible = await gcEventRow
    .waitFor({ state: "visible", timeout: 3_000 })
    .then(() => true)
    .catch(() => false);
  if (!gcEventVisible) {
    await expect(gcEmptyState).toBeVisible();
  }
  await page.screenshot({ path: path.join(outDir, "real-gc-pauses.png"), fullPage: true });

  await clickNav("Deadlocks");
  await expect(page.getByRole("heading", { name: "Deadlock cycles" })).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-deadlocks.png"), fullPage: true });

  await clickNav("Batches");
  const ingestion = page.getByRole("region", { name: "Ingestion health" });
  await expect(ingestion).toBeVisible();
  await expect(ingestion.getByText(/accepted x [1-9]\d*/i).first()).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-ingestion-health.png"), fullPage: true });
} finally {
  await browser.close();
}

console.log(JSON.stringify({
  baseURL,
  namespace,
  service,
  pod,
  outDir,
  screenshots: readdirSync(outDir).filter((name) => name.endsWith(".png")).sort(),
  consoleMessages,
}, null, 2));
