import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

export function middleware(request: NextRequest) {
  const accessTokenCookie = request.cookies.get("safebucket_access_token");

  if (!accessTokenCookie) {
    return NextResponse.redirect(new URL("/auth/login", request.url));
  }
}

export const config = {
  matcher: ["/", "/buckets/:path*"],
};
