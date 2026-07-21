import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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

type Handler = ((event: MessageEvent<string>) => void) | null;

class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  onmessage: Handler = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
    queueMicrotask(() => {
      this.onmessage?.({ data: JSON.stringify(eventPage) } as MessageEvent<string>);
    });
  }

  close() {}
}

beforeEach(() => {
  MockEventSource.instances = [];
  vi.stubGlobal("EventSource", MockEventSource);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("App", () => {
  it("renders a labelled search field and stream results", async () => {
    render(<App />);

    expect(screen.getByRole("searchbox", { name: "Find an event" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Processing status" })).toBeInTheDocument();
    expect(await screen.findByText("purchase.failed")).toBeInTheDocument();
    expect(screen.getByText("crl_onramp_62de")).toBeInTheDocument();
    expect(screen.getByText(/Last updated/)).toBeInTheDocument();
  });

  it("reconnects the stream with the selected status filter", async () => {
    render(<App />);
    await screen.findByText("purchase.failed");

    fireEvent.change(screen.getByRole("combobox", { name: "Processing status" }), {
      target: { value: "failed" }
    });

    await waitFor(() => {
      const latest = MockEventSource.instances.at(-1);
      expect(latest?.url).toContain("status=failed");
    });
  });

  it("offers a retry when the stream fails", async () => {
    class ErrorOnlySource {
      url: string;
      onmessage: Handler = null;
      onerror: ((event: Event) => void) | null = null;

      constructor(url: string) {
        this.url = url;
        queueMicrotask(() => this.onerror?.(new Event("error")));
      }

      close() {}
    }

    vi.stubGlobal("EventSource", ErrorOnlySource);

    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Unable to connect to the events stream"
    );
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("shows an empty state when no events match", async () => {
    class EmptySource {
      url: string;
      onmessage: Handler = null;
      onerror: ((event: Event) => void) | null = null;

      constructor(url: string) {
        this.url = url;
        queueMicrotask(() => {
          this.onmessage?.({
            data: JSON.stringify({ events: [], total: 0 })
          } as MessageEvent<string>);
        });
      }

      close() {}
    }

    vi.stubGlobal("EventSource", EmptySource);

    render(<App />);

    expect(await screen.findByRole("heading", { name: "No matching events" })).toBeInTheDocument();
    expect(screen.getByText("0 events found")).toBeInTheDocument();
  });

  it("surfaces an error when the stream payload is invalid", async () => {
    class InvalidSource {
      url: string;
      onmessage: Handler = null;
      onerror: ((event: Event) => void) | null = null;

      constructor(url: string) {
        this.url = url;
        queueMicrotask(() => {
          this.onmessage?.({ data: JSON.stringify({ total: 1 }) } as MessageEvent<string>);
        });
      }

      close() {}
    }

    vi.stubGlobal("EventSource", InvalidSource);

    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Invalid events response: events must be an array"
    );
  });
});
