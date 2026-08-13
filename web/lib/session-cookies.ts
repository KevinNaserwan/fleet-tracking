import type { NextRequest, NextResponse } from "next/server";

import type { TokenPair } from "@/lib/contracts";

export const ACCESS_COOKIE = "fleet_access";
export const REFRESH_COOKIE = "fleet_refresh";

function cookieSecure(): boolean {
  return process.env.NODE_ENV === "production" || process.env.SESSION_COOKIE_SECURE === "true";
}

function cookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: cookieSecure(),
    path: "/"
  };
}

export function readSessionCookies(request: NextRequest) {
  return {
    accessToken: request.cookies.get(ACCESS_COOKIE)?.value,
    refreshToken: request.cookies.get(REFRESH_COOKIE)?.value
  };
}

export function setSessionCookies(response: NextResponse, tokens: TokenPair): void {
  response.cookies.set(ACCESS_COOKIE, tokens.accessToken, {
    ...cookieOptions(),
    maxAge: tokens.expiresIn
  });
  response.cookies.set(REFRESH_COOKIE, tokens.refreshToken, cookieOptions());
}

export function clearSessionCookies(response: NextResponse): void {
  response.cookies.set(ACCESS_COOKIE, "", {
    ...cookieOptions(),
    maxAge: 0
  });
  response.cookies.set(REFRESH_COOKIE, "", {
    ...cookieOptions(),
    maxAge: 0
  });
}
