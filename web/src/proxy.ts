// Route guard (session 1.01). Learner pages need a session; without the session marker cookie
// the learner goes to sign-in and comes back afterwards (?next=). This is a fast redirect only:
// the marker holds no secret, the client then proves the session with a refresh call, and the
// API checks the access token on every request.

import { NextResponse, type NextRequest } from "next/server";

import { SIGNED_IN_COOKIE } from "@/features/auth/server/cookies";

export function proxy(request: NextRequest) {
  if (request.cookies.get(SIGNED_IN_COOKIE)?.value === "1") return NextResponse.next();
  const url = request.nextUrl.clone();
  const next = `${url.pathname}${url.search}`;
  url.pathname = "/auth/sign-in";
  url.search = "";
  url.searchParams.set("next", next);
  return NextResponse.redirect(url);
}

export const config = {
  matcher: ["/app", "/app/:path*", "/exam", "/exam/:path*", "/setup", "/setup/:path*"],
};
