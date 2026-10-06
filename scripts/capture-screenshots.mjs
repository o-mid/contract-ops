import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const baseUrl = process.env.SCREENSHOT_BASE_URL ?? "http://localhost:5173";
const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = join(root, "docs", "screenshots");

const shots = [
  { name: "01-activity-feed.png", path: "/" },
  { name: "02-status-filter.png", path: "/?status=processed" },
  { name: "03-search.png", path: "/?q=fireblocks" },
  { name: "04-empty-state.png", path: "/?q=zzznomatch" },
  { name: "05-connections.png", path: "/?view=connections" },
  { name: "06-settings.png", path: "/?view=settings" }
];

await mkdir(outDir, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });

for (const shot of shots) {
  await page.goto(`${baseUrl}${shot.path}`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector("h1", { timeout: 15_000 });
  await page.waitForTimeout(600);
  await page.screenshot({ path: join(outDir, shot.name), fullPage: false });
  console.log("wrote", shot.name);
}

await browser.close();
