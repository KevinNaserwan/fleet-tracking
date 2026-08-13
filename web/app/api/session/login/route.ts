import type { NextRequest } from "next/server";

import { backendFetch, readBackendJson, requestIdFor } from "@/lib/backend";
import { parseAuthEnvelope, parseErrorEnvelope } from "@/lib/contracts";
import { isJsonRequest, isTrustedMutation } from "@/lib/csrf";
import { bffError, noStoreJson } from "@/lib/http";
import { setSessionCookies } from "@/lib/session-cookies";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

interface LoginInput {
  tenant_slug: string;
  email: string;
  password: string;
}

function parseLoginInput(value: unknown): LoginInput | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  const input = value as Record<string, unknown>;
  const expectedKeys = ["tenant_slug", "email", "password"];
  if (Object.keys(input).length !== expectedKeys.length || !expectedKeys.every((key) => Object.hasOwn(input, key))) {
    return null;
  }
  if (
    typeof input.tenant_slug !== "string" ||
    input.tenant_slug.length < 1 ||
    input.tenant_slug.length > 63 ||
    !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(input.tenant_slug) ||
    typeof input.email !== "string" ||
    input.email.length > 320 ||
    !input.email.includes("@") ||
    typeof input.password !== "string" ||
    input.password.length < 8 ||
    input.password.length > 1024
  ) {
    return null;
  }
  return input as unknown as LoginInput;
}

function publicStatus(upstreamStatus: number): number {
  return [400, 401, 422, 429].includes(upstreamStatus) ? upstreamStatus : 502;
}

export async function POST(request: NextRequest) {
  const requestId = requestIdFor(request);

  if (!isTrustedMutation(request)) {
    return bffError(403, "invalid_origin", "Permintaan tidak berasal dari aplikasi ini.", requestId);
  }
  if (!isJsonRequest(request)) {
    return bffError(415, "unsupported_media_type", "Gunakan Content-Type application/json.", requestId);
  }

  let rawInput: unknown;
  try {
    rawInput = await request.json();
  } catch {
    return bffError(400, "bad_request", "Isi permintaan bukan JSON yang valid.", requestId);
  }

  const input = parseLoginInput(rawInput);
  if (!input) {
    return bffError(422, "validation_failed", "Periksa tenant, email, dan kata sandi.", requestId);
  }

  let upstream: Response;
  try {
    upstream = await backendFetch("/api/v1/auth/login", requestId, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        tenantSlug: input.tenant_slug,
        email: input.email,
        password: input.password
      })
    });
  } catch {
    return bffError(502, "backend_unavailable", "Layanan identitas tidak dapat dijangkau.", requestId);
  }

  const payload = await readBackendJson(upstream);
  if (!upstream.ok) {
    const parsedError = parseErrorEnvelope(payload);
    const response = parsedError
      ? noStoreJson(parsedError, publicStatus(upstream.status))
      : bffError(publicStatus(upstream.status), "backend_error", "Layanan identitas menolak permintaan.", requestId);
    const retryAfter = upstream.headers.get("retry-after");
    if (retryAfter && upstream.status === 429) {
      response.headers.set("Retry-After", retryAfter);
    }
    return response;
  }

  const envelope = parseAuthEnvelope(payload);
  if (!envelope) {
    return bffError(502, "invalid_backend_response", "Respons layanan identitas tidak sesuai kontrak.", requestId);
  }

  const response = noStoreJson(
    {
      data: { user: envelope.data.user },
      meta: envelope.meta ?? { requestId }
    },
    200
  );
  setSessionCookies(response, envelope.data.tokens);
  return response;
}
