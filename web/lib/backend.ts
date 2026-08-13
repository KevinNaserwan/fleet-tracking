import type { NextRequest } from "next/server";

const REQUEST_ID_PATTERN = /^[A-Za-z0-9._:/-]{1,128}$/;
const BACKEND_TIMEOUT_MS = 8_000;

export function requestIdFor(request: NextRequest): string {
  const supplied = request.headers.get("x-request-id");
  return supplied && REQUEST_ID_PATTERN.test(supplied) ? supplied : crypto.randomUUID();
}

export async function backendFetch(
  path: string,
  requestId: string,
  init: Omit<RequestInit, "cache" | "redirect" | "signal"> = {}
): Promise<Response> {
  const configuredUrl = process.env.BACKEND_URL;
  if (!configuredUrl) {
    throw new Error("BACKEND_URL is not configured");
  }

  const baseUrl = new URL(configuredUrl.endsWith("/") ? configuredUrl : `${configuredUrl}/`);
  const endpoint = new URL(path.replace(/^\//, ""), baseUrl);
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  headers.set("X-Request-ID", requestId);

  return fetch(endpoint, {
    ...init,
    headers,
    cache: "no-store",
    redirect: "error",
    signal: AbortSignal.timeout(BACKEND_TIMEOUT_MS)
  });
}

export async function readBackendJson(response: Response): Promise<unknown | null> {
  const contentType = response.headers.get("content-type");
  if (!contentType?.toLowerCase().includes("application/json")) {
    return null;
  }
  try {
    return await response.json();
  } catch {
    return null;
  }
}
