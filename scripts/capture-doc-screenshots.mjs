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

mkdirSync(outDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({
  viewport: { width: 1440, height: 1000 },
  deviceScaleFactor: 1,
});

const consoleMessages = [];
page.on("console", (message) => consoleMessages.push(`[${message.type()}] ${message.text()}`));
page.on("pageerror", (error) => consoleMessages.push(`[pageerror] ${error.message}`));

try {
  await page.goto(baseURL, { waitUntil: "networkidle" });
  await expect(page.getByRole("heading", { name: "Service diagnosis" })).toBeVisible();
  await page.getByRole("textbox", { name: "Namespace", exact: true }).fill(namespace);
  await page.getByRole("textbox", { name: "Service", exact: true }).fill(service);
  await page.getByLabel("Range").selectOption("360");

  await page.getByRole("button", { name: "status", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Target status" })).toBeVisible();
  await expect(page.getByRole("cell", { name: /accepted|unsupported_jvm|temporary_expired|disabled_by_metadata/ }).first()).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-target-status.png"), fullPage: true });

  await page.getByRole("button", { name: "cpu", exact: true }).click();
  await expect(page.getByRole("region", { name: "CPU profile analysis" }).getByRole("heading", { name: "CPU profile" })).toBeVisible();
  const topTable = page.getByRole("region", { name: "Top table" });
  const demoFrame = topTable.getByRole("button", { name: /DemoHttpService\.(burnCpu|handleWork|allocateObjects)/ }).first();
  await expect(demoFrame).toBeVisible();
  await demoFrame.click();
  await expect(page.getByRole("region", { name: "Selected flamegraph frame" })).toBeVisible();
  await page.getByRole("button", { name: "Both" }).click();
  await page.screenshot({ path: path.join(outDir, "real-cpu-analysis.png"), fullPage: true });

  await page.getByRole("button", { name: "deadlocks", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Deadlock cycles" })).toBeVisible();
  await page.screenshot({ path: path.join(outDir, "real-deadlocks.png"), fullPage: true });

  await page.getByRole("button", { name: "ingestion", exact: true }).click();
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
  outDir,
  screenshots: readdirSync(outDir).filter((name) => name.endsWith(".png")).sort(),
  consoleMessages,
}, null, 2));
