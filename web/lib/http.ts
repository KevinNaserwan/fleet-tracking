import { NextResponse } from "next/server";

import type { ErrorDetail, ErrorEnvelope } from "@/lib/contracts";

export function noStoreJson(body: unknown, status = 200): NextResponse {
  const response = NextResponse.json(body, { status });
  response.headers.set("Cache-Control", "private, no-store, max-age=0");
  response.headers.set("Pragma", "no-cache");
  return response;
}

export function bffError(
  status: number,
  code: string,
  message: string,
  requestId: string,
  details?: ErrorDetail[]
): NextResponse {
  const body: ErrorEnvelope = {
    error: {
      code,
      message,
      requestId,
      ...(details ? { details } : {})
    }
  };
  return noStoreJson(body, status);
}
