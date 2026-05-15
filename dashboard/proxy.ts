// Optimistic route protection. Per the Next 16 docs, proxy is not a real
// authorization boundary — every server action also calls requireSession()
// before mutating state. The check here exists to redirect un-cookied
// visitors to /login without spinning up a full server component render.
import { NextResponse, type NextRequest } from "next/server";

import { verifySession } from "@/lib/jwt";

export async function proxy(request: NextRequest) {
  const token = request.cookies.get("oas_session")?.value;
  const session = token ? await verifySession(token) : null;
  if (!session) {
    const url = request.nextUrl.clone();
    url.pathname = "/login";
    url.search = "";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

// Run on every path except the auth pages, the auth API, the healthz probe,
// and Next.js internals (build artifacts, favicon).
export const config = {
  matcher: ["/((?!login|setup|api/auth|api/healthz|_next|favicon).*)"],
};
