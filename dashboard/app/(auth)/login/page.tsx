import { redirect } from "next/navigation";
import { sql } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { users } from "@/lib/db/schema";
import { getSession } from "@/lib/session";
import { LoginForm } from "@/components/auth/login-form";

export const dynamic = "force-dynamic";

export default async function LoginPage() {
  // If we're already logged in, skip the form.
  const session = await getSession();
  if (session) redirect("/campaigns");

  // First-run bootstrap: when no admin exists, send the user to /setup.
  const [{ count }] = await db
    .select({ count: sql<number>`COUNT(*)::int` })
    .from(users);
  if (count === 0) redirect("/setup");

  return <LoginForm />;
}
