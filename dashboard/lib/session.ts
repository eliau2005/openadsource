// Cookie + session helpers. Server-only (uses next/headers).
import "server-only";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { isProduction } from "@/lib/env";
import { signSession, verifySession, SESSION_TTL_SECONDS, type SessionPayload } from "@/lib/jwt";

const COOKIE_NAME = "oas_session";

export async function setSessionCookie(payload: SessionPayload): Promise<void> {
  const token = await signSession(payload);
  const jar = await cookies();
  jar.set(COOKIE_NAME, token, {
    httpOnly: true,
    sameSite: "lax",
    secure: isProduction,
    path: "/",
    maxAge: SESSION_TTL_SECONDS,
  });
}

export async function clearSessionCookie(): Promise<void> {
  const jar = await cookies();
  jar.delete(COOKIE_NAME);
}

export async function getSession(): Promise<SessionPayload | null> {
  const jar = await cookies();
  const token = jar.get(COOKIE_NAME)?.value;
  if (!token) return null;
  return verifySession(token);
}

export async function requireSession(): Promise<SessionPayload> {
  const session = await getSession();
  if (!session) redirect("/login");
  return session;
}

export { COOKIE_NAME };
