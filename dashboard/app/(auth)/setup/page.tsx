import { redirect } from "next/navigation";
import { sql } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { users } from "@/lib/db/schema";
import { SetupForm } from "@/components/auth/setup-form";

export const dynamic = "force-dynamic";

export default async function SetupPage() {
  const [{ count }] = await db
    .select({ count: sql<number>`COUNT(*)::int` })
    .from(users);
  if (count > 0) redirect("/login");

  return <SetupForm />;
}
