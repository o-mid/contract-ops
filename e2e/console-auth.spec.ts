import { test, expect } from "@playwright/test";
import { E2E_API_KEY, queueBackfillForFirstConnection, seedConsoleAuth, waitForCosts } from "./helpers";

test.describe("authenticated console", () => {
  test.beforeEach(async ({ page }) => {
    await seedConsoleAuth(page);
  });

  test("settings stores API key and base URL fields", async ({ page }) => {
    await page.goto("/console?view=settings");
    await expect(page.getByRole("heading", { name: "Settings" })).toBeVisible();
    await expect(page.locator("#api-key")).toHaveValue(E2E_API_KEY);
  });

  test("connections lists seeded demo vendors", async ({ page }) => {
    await page.goto("/console?view=connections");
    await expect(page.getByText("Acme Billing (demo)")).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText("Staging vendor (demo)")).toBeVisible();
  });

  test("costs page shows rows after sync", async ({ page }) => {
    await queueBackfillForFirstConnection();
    await waitForCosts(1);

    await page.goto("/console?view=costs");
    await expect(page.getByRole("heading", { name: "Costs" })).toBeVisible();
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByRole("cell", { name: /fakevendor/i }).first()).toBeVisible();
  });
});
