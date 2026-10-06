import { test, expect } from "@playwright/test";
import { E2E_API_BASE, E2E_API_KEY, queueBackfillForFirstConnection, waitForCosts } from "./helpers";

test.describe("HTTP API", () => {
  test("health and readiness", async ({ request }) => {
    const health = await request.get(`${E2E_API_BASE}/healthz`);
    expect(health.ok()).toBeTruthy();
    const ready = await request.get(`${E2E_API_BASE}/readyz`);
    expect(ready.ok()).toBeTruthy();
  });

  test("costs endpoint requires auth", async ({ request }) => {
    const denied = await request.get(`${E2E_API_BASE}/v1/costs`);
    expect(denied.status()).toBe(401);
  });

  test("backfill produces cost rows", async ({ request }) => {
    await queueBackfillForFirstConnection();
    await waitForCosts(1);
    const response = await request.get(`${E2E_API_BASE}/v1/costs?limit=10`, {
      headers: { Authorization: `Bearer ${E2E_API_KEY}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.total).toBeGreaterThan(0);
    expect(body.costs?.length).toBeGreaterThan(0);
  });
});
