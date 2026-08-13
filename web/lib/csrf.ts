import type { NextRequest } from "next/server";

export function isTrustedMutation(request: NextRequest): boolean {
  const origin = request.headers.get("origin");
  if (!origin) {
    return false;
  }

  const fetchSite = request.headers.get("sec-fetch-site");
  if (fetchSite && fetchSite !== "same-origin") {
    return false;
  }

  try {
    const configuredOrigin = new URL(process.env.NEXT_PUBLIC_APP_URL ?? request.url).origin;
    return new URL(origin).origin === configuredOrigin;
  } catch {
    return false;
  }
}

export function isJsonRequest(request: NextRequest): boolean {
  return request.headers.get("content-type")?.toLowerCase().startsWith("application/json") ?? false;
}
