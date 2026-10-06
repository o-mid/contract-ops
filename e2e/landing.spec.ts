import { test, expect } from "@playwright/test";

test.describe("marketing site", () => {
  test("landing page and architecture section", async ({ page }) => {
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: /integrations you can operate/i })
    ).toBeVisible();
    await expect(page.getByRole("link", { name: "Open console" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Launch console" })).toBeVisible();

    await page.goto("/#architecture");
    await expect(page.locator("#architecture")).toBeVisible();
  });
});
