"use server";

import { redirect } from "next/navigation";
import { eq, sql } from "drizzle-orm";
import { z } from "zod";

import { db } from "@/lib/db/client";
import { users } from "@/lib/db/schema";
import { verifyPassword } from "@/lib/password";
import { setSessionCookie } from "@/lib/session";

const loginSchema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
});

export type LoginState = { error?: string } | undefined;

export async function loginAction(_prev: LoginState, formData: FormData): Promise<LoginState> {
  const parsed = loginSchema.safeParse({
    email: String(formData.get("email") ?? ""),
    password: String(formData.get("password") ?? ""),
  });
  if (!parsed.success) {
    return { error: parsed.error.issues[0]?.message ?? "Invalid input" };
  }

  const [row] = await db
    .select()
    .from(users)
    .where(eq(sql`LOWER(${users.email})`, parsed.data.email.toLowerCase()))
    .limit(1);

  if (!row) return { error: "Invalid email or password" };

  const ok = await verifyPassword(parsed.data.password, row.passwordHash);
  if (!ok) return { error: "Invalid email or password" };

  await setSessionCookie({ sub: row.id, email: row.email });
  redirect("/campaigns");
}
