import type { NextRequest, NextResponse } from "next/server";

import { backendFetch, readBackendJson, requestIdFor } from "@/lib/backend";
import { parseLogoutEnvelope } from "@/lib/contracts";
import { isJsonRequest, isTrustedMutation } from "@/lib/csrf";
import { bffError, noStoreJson } from "@/lib/http";
import { clearSessionCookies, readSessionCookies } from "@/lib/session-cookies";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

function clearLocalSession(response: NextResponse): NextResponse {
  clearSessionCookies(response);
  return response;
}

export async function POST(request: NextRequest) {
  const requestId = requestIdFor(request);
  if (!isTrustedMutation(request)) {
    return bffError(403, "invalid_origin", "Permintaan tidak berasal dari aplikasi ini.", requestId);
  }
  if (!isJsonRequest(request)) {
    return clearLocalSession(
      bffError(415, "unsupported_media_type", "Gunakan Content-Type application/json.", requestId)
    );
  }

  const { accessToken, refreshToken } = readSessionCookies(request);
  let upstreamRevocation: "confirmed" | "unconfirmed" = refreshToken ? "unconfirmed" : "confirmed";

  if (refreshToken) {
    try {
      const headers = new Headers({ "Content-Type": "application/json" });
      if (accessToken) {
        headers.set("Authorization", `Bearer ${accessToken}`);
      }
      const upstream = await backendFetch("/api/v1/auth/logout", requestId, {
        method: "POST",
        headers,
        body: JSON.stringify({ refreshToken })
      });
      const payload = await readBackendJson(upstream);
      if (upstream.ok && parseLogoutEnvelope(payload)) {
        upstreamRevocation = "confirmed";
      }
    } catch {
      upstreamRevocation = "unconfirmed";
    }
  }

  const response = noStoreJson(
    {
      data: { loggedOut: true, localSessionCleared: true },
      meta: { requestId, upstreamRevocation }
    },
    200
  );
  return clearLocalSession(response);
}
