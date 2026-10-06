import { test, expect } from "@playwright/test";
import { E2E_API_BASE, seedConsoleAuth } from "./helpers";

test.describe("console activity (public feed)", () => {
  test.beforeEach(async ({ page }) => {
    await seedConsoleAuth(page);
  });

  test("loads feed and supports search filters in the URL", async ({ page }) => {
    await page.goto("/console");
    await expect(page.getByRole("heading", { name: "Activity feed" })).toBeVisible();

    await page.goto("/console?status=processed");
    await expect(page).toHaveURL(/status=processed/);
    await expect(page.getByRole("heading", { name: "Activity feed" })).toBeVisible();

    await page.goto("/console?q=zzznomatch");
    await expect(page.getByRole("heading", { name: "No matching events" })).toBeVisible({
      timeout: 30_000
    });
  });
});
