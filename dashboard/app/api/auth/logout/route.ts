import { NextResponse } from "next/server";

import { clearSessionCookie } from "@/lib/session";

// GET so the topbar's logout link works without JS. The route clears the
// HTTP-only cookie and redirects back to /login.
export async function GET(request: Request) {
  await clearSessionCookie();
  return NextResponse.redirect(new URL("/login", request.url));
}
