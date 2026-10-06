import type { Page } from "@playwright/test";

export const E2E_API_KEY =
  process.env.E2E_API_KEY ?? "co_local_dev_key_not_for_production";
export const E2E_API_BASE = process.env.E2E_API_BASE ?? "http://localhost:8080";

export async function seedConsoleAuth(page: Page, options?: { apiKey?: string }) {
  const key = options?.apiKey ?? E2E_API_KEY;
  await page.addInitScript(
    ({ key, base }) => {
      if (key) {
        localStorage.setItem("contract_ops_api_key", key);
      }
      localStorage.setItem("contract_ops_api_base", base);
    },
    { key, base: E2E_API_BASE }
  );
}

export async function waitForApiReady() {
  const url = `${E2E_API_BASE}/healthz`;
  const deadline = Date.now() + 90_000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      /* retry */
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`API not ready at ${url}`);
}

export async function waitForCosts(minRows = 1) {
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    const response = await fetch(`${E2E_API_BASE}/v1/costs?limit=5`, {
      headers: { Authorization: `Bearer ${E2E_API_KEY}` }
    });
    if (response.ok) {
      const body = (await response.json()) as { costs?: unknown[] };
      if ((body.costs?.length ?? 0) >= minRows) {
        return;
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
  throw new Error("timed out waiting for cost rows");
}

export async function queueBackfillForFirstConnection() {
  const list = await fetch(`${E2E_API_BASE}/v1/connections`, {
    headers: { Authorization: `Bearer ${E2E_API_KEY}` }
  });
  if (!list.ok) {
    throw new Error(`list connections: ${list.status}`);
  }
  const payload = (await list.json()) as { connections: { id: string }[] };
  const id = payload.connections[0]?.id;
  if (!id) {
    throw new Error("no connections to backfill");
  }
  const end = new Date();
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
  const backfill = await fetch(`${E2E_API_BASE}/v1/connections/${id}/backfill`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${E2E_API_KEY}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ start: start.toISOString(), end: end.toISOString() })
  });
  if (!backfill.ok && backfill.status !== 202) {
    throw new Error(`backfill: ${backfill.status}`);
  }
}
