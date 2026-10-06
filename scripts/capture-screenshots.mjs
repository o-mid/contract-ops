import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const baseUrl = process.env.SCREENSHOT_BASE_URL ?? "http://localhost:3000";
const apiKey =
  process.env.SCREENSHOT_API_KEY ?? "co_local_dev_key_not_for_production";
const apiBase = process.env.SCREENSHOT_API_BASE ?? "http://localhost:8080";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = join(root, "docs", "screenshots");

const shots = [
  { name: "01-landing.png", path: "/", wait: 900 },
  { name: "02-architecture.png", path: "/#architecture", wait: 1200, scroll: true },
  { name: "03-activity-feed.png", path: "/console", wait: 1200, needsKey: false },
  {
    name: "04-status-filter.png",
    path: "/console?status=processed",
    wait: 800,
    needsKey: false
  },
  {
    name: "05-connections.png",
    path: "/console?view=connections",
    wait: 800,
    needsKey: true,
    waitFor: "Acme Billing (demo)"
  },
  { name: "06-settings.png", path: "/console?view=settings", wait: 800, needsKey: false },
  {
    name: "07-empty-state.png",
    path: "/console?q=zzznomatch",
    wait: 800,
    needsKey: false
  },
  {
    name: "08-costs.png",
    path: "/console?view=costs",
    wait: 800,
    needsKey: true,
    waitFor: "Billing rows"
  }
];

await mkdir(outDir, { recursive: true });

const browser = await chromium.launch();
const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 2
});

await context.addInitScript(
  ({ key, base }) => {
    localStorage.setItem("contract_ops_api_key", key);
    localStorage.setItem("contract_ops_api_base", base);
  },
  { key: apiKey, base: apiBase }
);

const page = await context.newPage();

for (const shot of shots) {
  await page.goto(`${baseUrl}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector("h1", { timeout: 30_000 });
  if (shot.waitFor) {
    const locator = shot.waitFor === "Billing rows"
      ? page.getByRole("heading", { name: shot.waitFor })
      : page.getByRole("listitem").filter({ hasText: shot.waitFor });
    await locator.waitFor({
      timeout: 30_000
    });
    await page.waitForTimeout(400);
  }
  await page.waitForTimeout(shot.wait);
  if (shot.scroll) {
    await page.evaluate(() => window.scrollTo({ top: 700, behavior: "instant" }));
    await page.waitForTimeout(400);
  }
  await page.screenshot({ path: join(outDir, shot.name), fullPage: false });
  console.log("wrote", shot.name);
}

await browser.close();
