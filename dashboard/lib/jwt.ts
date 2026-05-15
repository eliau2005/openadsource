// JWT signing + verification on jose so it runs in both Node and Edge
// runtimes (proxy.ts runs on Edge by default in Next 16). Kept separate from
// lib/password.ts (bcryptjs) so proxy.ts can import only the JWT half without
// pulling in the Node-only bcrypt code.
import "server-only";

import { SignJWT, jwtVerify } from "jose";

import { env } from "@/lib/env";

const SESSION_TTL_SECONDS = 7 * 24 * 60 * 60;
const ISSUER = "openadsource";
const AUDIENCE = "openadsource-dashboard";

export interface SessionPayload {
  sub: string;
  email: string;
}

let cachedKey: Uint8Array | null = null;
function getSecretKey(): Uint8Array {
  if (cachedKey) return cachedKey;
  cachedKey = new TextEncoder().encode(env.JWT_SECRET);
  return cachedKey;
}

export async function signSession(payload: SessionPayload): Promise<string> {
  return new SignJWT({ email: payload.email })
    .setProtectedHeader({ alg: "HS256" })
    .setSubject(payload.sub)
    .setIssuedAt()
    .setIssuer(ISSUER)
    .setAudience(AUDIENCE)
    .setExpirationTime(`${SESSION_TTL_SECONDS}s`)
    .sign(getSecretKey());
}

export async function verifySession(token: string): Promise<SessionPayload | null> {
  try {
    const { payload } = await jwtVerify(token, getSecretKey(), {
      issuer: ISSUER,
      audience: AUDIENCE,
    });
    if (typeof payload.sub !== "string" || typeof payload.email !== "string") {
      return null;
    }
    return { sub: payload.sub, email: payload.email };
  } catch {
    return null;
  }
}

export { SESSION_TTL_SECONDS };
