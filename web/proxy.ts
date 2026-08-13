import { type NextRequest, NextResponse } from "next/server";

import { ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/session-cookies";

export function proxy(request: NextRequest) {
  const hasSession =
    Boolean(request.cookies.get(ACCESS_COOKIE)?.value) ||
    Boolean(request.cookies.get(REFRESH_COOKIE)?.value);

  if (!hasSession) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("next", `${request.nextUrl.pathname}${request.nextUrl.search}`);
    return NextResponse.redirect(loginUrl, 307);
  }

  const response = NextResponse.next();
  response.headers.set("Cache-Control", "private, no-store, max-age=0");
  return response;
}

export const config = {
  matcher: ["/dashboard/:path*"]
};
