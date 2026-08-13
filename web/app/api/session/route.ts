import { createHash } from "node:crypto";
import type { NextRequest } from "next/server";

import { backendFetch, readBackendJson, requestIdFor } from "@/lib/backend";
import {
  parseErrorEnvelope,
  parseMeEnvelope,
  parseTokenEnvelope,
  type ErrorEnvelope,
  type TokenPair
} from "@/lib/contracts";
import { bffError, noStoreJson } from "@/lib/http";
import { clearSessionCookies, readSessionCookies, setSessionCookies } from "@/lib/session-cookies";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

function bearerHeaders(token: string): HeadersInit {
  return { Authorization: `Bearer ${token}` };
}

function publicStatus(status: number): number {
  return [400, 401, 403, 422, 429].includes(status) ? status : 502;
}

interface RefreshFailure {
  status: number;
  forwardedError?: ErrorEnvelope;
  code: string;
  message: string;
}

type RefreshResult = { tokens: TokenPair } | { failure: RefreshFailure };

const refreshFlights = new Map<string, Promise<RefreshResult>>();

async function meResult(upstream: Response, requestId: string) {
  const payload = await readBackendJson(upstream);
  if (upstream.ok) {
    const envelope = parseMeEnvelope(payload);
    return envelope
      ? noStoreJson(envelope, 200)
      : bffError(502, "invalid_backend_response", "Respons profil tidak sesuai kontrak.", requestId);
  }
  const parsedError = parseErrorEnvelope(payload);
  return parsedError
    ? noStoreJson(parsedError, publicStatus(upstream.status))
    : bffError(publicStatus(upstream.status), "backend_error", "Profil tidak dapat dimuat.", requestId);
}

async function refreshTokens(refreshToken: string, requestId: string): Promise<RefreshResult> {
  const upstream = await backendFetch("/api/v1/auth/refresh", requestId, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken })
  });
  const payload = await readBackendJson(upstream);
  if (!upstream.ok) {
    const parsedError = parseErrorEnvelope(payload);
    return {
      failure: {
        status: publicStatus(upstream.status),
        ...(parsedError ? { forwardedError: parsedError } : {}),
        code: "refresh_failed",
        message: "Sesi tidak dapat diperbarui."
      }
    };
  }
  const envelope = parseTokenEnvelope(payload);
  return envelope
    ? { tokens: envelope.data }
    : {
        failure: {
          status: 502,
          code: "invalid_backend_response",
          message: "Respons pembaruan sesi tidak sesuai kontrak."
        }
      };
}

function refreshTokensSingleFlight(refreshToken: string, requestId: string): Promise<RefreshResult> {
  const key = createHash("sha256").update(refreshToken).digest("base64url");
  const existing = refreshFlights.get(key);
  if (existing) {
    return existing;
  }

  const flight = refreshTokens(refreshToken, requestId);
  refreshFlights.set(key, flight);
  const removeFlight = () => {
    if (refreshFlights.get(key) === flight) {
      refreshFlights.delete(key);
    }
  };
  void flight.then(removeFlight, removeFlight);
  return flight;
}

function refreshFailureResponse(failure: RefreshFailure, requestId: string) {
  return failure.forwardedError
    ? noStoreJson(failure.forwardedError, failure.status)
    : bffError(failure.status, failure.code, failure.message, requestId);
}

export async function GET(request: NextRequest) {
  const requestId = requestIdFor(request);
  const { accessToken, refreshToken } = readSessionCookies(request);

  if (!accessToken && !refreshToken) {
    return bffError(401, "unauthorized", "Sesi tidak tersedia.", requestId);
  }

  try {
    if (accessToken) {
      const firstAttempt = await backendFetch("/api/v1/me", requestId, {
        method: "GET",
        headers: bearerHeaders(accessToken)
      });
      if (firstAttempt.status !== 401) {
        return meResult(firstAttempt, requestId);
      }
    }

    if (!refreshToken) {
      const response = bffError(401, "unauthorized", "Sesi telah berakhir.", requestId);
      clearSessionCookies(response);
      return response;
    }

    const refreshed = await refreshTokensSingleFlight(refreshToken, requestId);
    if ("failure" in refreshed) {
      const response = refreshFailureResponse(refreshed.failure, requestId);
      if ([400, 401].includes(response.status)) {
        clearSessionCookies(response);
      }
      return response;
    }

    try {
      const finalAttempt = await backendFetch("/api/v1/me", requestId, {
        method: "GET",
        headers: bearerHeaders(refreshed.tokens.accessToken)
      });
      const response = await meResult(finalAttempt, requestId);
      if (finalAttempt.status === 401) {
        clearSessionCookies(response);
      } else {
        setSessionCookies(response, refreshed.tokens);
      }
      return response;
    } catch {
      const response = bffError(502, "backend_unavailable", "Layanan identitas tidak dapat dijangkau.", requestId);
      setSessionCookies(response, refreshed.tokens);
      return response;
    }
  } catch {
    return bffError(502, "backend_unavailable", "Layanan identitas tidak dapat dijangkau.", requestId);
  }
}
