// Password hashing on bcryptjs. Kept out of lib/jwt.ts so proxy.ts (Edge
// runtime) doesn't transitively pull bcryptjs into its bundle.
import "server-only";

import bcrypt from "bcryptjs";

const ROUNDS = 12;

export async function hashPassword(plain: string): Promise<string> {
  return bcrypt.hash(plain, ROUNDS);
}

export async function verifyPassword(plain: string, hash: string): Promise<boolean> {
  if (!plain || !hash) return false;
  try {
    return await bcrypt.compare(plain, hash);
  } catch {
    return false;
  }
}
