"use server";

import { redirect } from "next/navigation";
import { sql } from "drizzle-orm";
import { z } from "zod";

import { db } from "@/lib/db/client";
import { users } from "@/lib/db/schema";
import { hashPassword } from "@/lib/password";
import { setSessionCookie } from "@/lib/session";

const setupSchema = z
  .object({
    email: z.string().email("Enter a valid email"),
    password: z.string().min(8, "Password must be at least 8 characters"),
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, {
    message: "Passwords do not match",
    path: ["confirm"],
  });

export type SetupState = { error?: string } | undefined;

export async function createAdminAction(_prev: SetupState, formData: FormData): Promise<SetupState> {
  const parsed = setupSchema.safeParse({
    email: String(formData.get("email") ?? ""),
    password: String(formData.get("password") ?? ""),
    confirm: String(formData.get("confirm") ?? ""),
  });
  if (!parsed.success) {
    return { error: parsed.error.issues[0]?.message ?? "Invalid input" };
  }

  // Guard rails: re-check that we're still in the first-run window. Closes a
  // narrow race where two clients hit /setup simultaneously.
  const [{ count }] = await db
    .select({ count: sql<number>`COUNT(*)::int` })
    .from(users);
  if (count > 0) {
    return { error: "An admin already exists. Please sign in." };
  }

  const passwordHash = await hashPassword(parsed.data.password);
  const [inserted] = await db
    .insert(users)
    .values({ email: parsed.data.email, passwordHash, role: "admin" })
    .returning({ id: users.id, email: users.email });

  await setSessionCookie({ sub: inserted.id, email: inserted.email });
  redirect("/campaigns");
}
