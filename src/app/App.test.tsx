import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

const eventPage = {
  events: [
    {
      id: "evt_01HV1B03",
      source: "onramp-webhook",
      type: "purchase.failed",
      status: "failed",
      occurredAt: "2026-07-21T07:57:00Z",
      correlationId: "crl_onramp_62de"
    }
  ],
  total: 1
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("App", () => {
  it("renders a labelled search field and API results", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(eventPage), { status: 200 })
      )
    );

    render(<App />);

    expect(screen.getByRole("searchbox", { name: "Find an event" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Processing status" })).toBeInTheDocument();
    expect(await screen.findByText("purchase.failed")).toBeInTheDocument();
    expect(screen.getByText("crl_onramp_62de")).toBeInTheDocument();
  });

  it("requests the selected status filter", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(eventPage), { status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    await screen.findByText("purchase.failed");

    fireEvent.change(screen.getByRole("combobox", { name: "Processing status" }), {
      target: { value: "failed" }
    });

    await waitFor(() => {
      expect(fetchMock).toHaveBeenLastCalledWith(
        expect.stringContaining("status=failed"),
        expect.anything()
      );
    });
  });

  it("offers a retry when the API fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("", { status: 503 })));

    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load events (503)");
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("shows an empty state when no events match", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ events: [], total: 0 }), { status: 200 })
      )
    );

    render(<App />);

    expect(await screen.findByRole("heading", { name: "No matching events" })).toBeInTheDocument();
    expect(screen.getByText("0 events found")).toBeInTheDocument();
  });
});
