import { NextResponse } from "next/server";

import { clearSessionCookie } from "@/lib/session";

// GET so the topbar's logout link works without JS. Clears the HTTP-only
// cookie and 307s back to /login via a *relative* Location header — the
// browser resolves it against the request's own origin, which side-steps the
// fact that Next standalone constructs absolute URLs from its HOSTNAME env
// (0.0.0.0 in our Docker setup), not the inbound Host header.
export async function GET() {
  await clearSessionCookie();
  return new NextResponse(null, {
    status: 307,
    headers: { Location: "/login" },
  });
}
