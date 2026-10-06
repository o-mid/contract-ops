import { errorCatalog } from "../domain/errorCatalog.gen";

export type ProblemDetails = {
  type: string;
  title: string;
  status: number;
  detail?: string;
  code?: string;
  retryable?: boolean;
  action?: string;
};

export class ApiError extends Error {
  readonly status: number;
  readonly problem?: ProblemDetails;

  constructor(message: string, status: number, problem?: ProblemDetails) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.problem = problem;
  }

  userMessage(): string {
    if (this.problem?.code) {
      const entry = errorCatalog[this.problem.code as keyof typeof errorCatalog];
      if (entry?.message) {
        return entry.message;
      }
    }
    return this.problem?.detail ?? this.problem?.title ?? this.message;
  }
}

type RequestOptions = {
  apiBaseUrl: string;
  apiKey?: string;
  signal?: AbortSignal;
};

export async function apiRequest<T>(
  path: string,
  options: RequestOptions & {
    method?: string;
    body?: unknown;
    headers?: Record<string, string>;
  }
): Promise<T> {
  const { apiBaseUrl, apiKey, signal, method = "GET", body, headers = {} } = options;
  const url = `${apiBaseUrl.replace(/\/$/, "")}${path}`;

  const requestHeaders: Record<string, string> = {
    Accept: "application/json",
    ...headers
  };

  if (apiKey?.trim()) {
    requestHeaders.Authorization = `Bearer ${apiKey.trim()}`;
  }

  if (body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  const response = await fetch(url, {
    method,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal
  });

  const contentType = response.headers.get("content-type") ?? "";

  if (!response.ok) {
    if (contentType.includes("application/problem+json")) {
      const problem = (await response.json()) as ProblemDetails;
      throw new ApiError(problem.title ?? "Request failed", response.status, problem);
    }
    if (contentType.includes("application/json")) {
      const payload = (await response.json()) as { error?: string };
      throw new ApiError(payload.error ?? `Request failed (${response.status})`, response.status);
    }
    throw new ApiError(`Request failed (${response.status})`, response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
