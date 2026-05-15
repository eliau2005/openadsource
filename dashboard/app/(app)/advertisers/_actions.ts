"use server";

import { eq } from "drizzle-orm";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { db } from "@/lib/db/client";
import { advertisers } from "@/lib/db/schema";
import { requireSession } from "@/lib/session";

const advertiserSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(200),
  status: z.enum(["active", "archived"]).default("active"),
});

export type AdvertiserActionState = { error?: string } | undefined;

export async function createAdvertiserAction(
  _prev: AdvertiserActionState,
  formData: FormData,
): Promise<AdvertiserActionState> {
  await requireSession();
  const parsed = advertiserSchema.safeParse({
    name: String(formData.get("name") ?? ""),
    status: String(formData.get("status") ?? "active"),
  });
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db.insert(advertisers).values(parsed.data);
  revalidatePath("/advertisers");
  redirect("/advertisers");
}

export async function updateAdvertiserAction(
  id: string,
  _prev: AdvertiserActionState,
  formData: FormData,
): Promise<AdvertiserActionState> {
  await requireSession();
  const parsed = advertiserSchema.safeParse({
    name: String(formData.get("name") ?? ""),
    status: String(formData.get("status") ?? "active"),
  });
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db
    .update(advertisers)
    .set({ ...parsed.data, updatedAt: new Date() })
    .where(eq(advertisers.id, id));
  revalidatePath("/advertisers");
  redirect("/advertisers");
}

export async function archiveAdvertiserAction(id: string): Promise<void> {
  await requireSession();
  await db
    .update(advertisers)
    .set({ status: "archived", updatedAt: new Date() })
    .where(eq(advertisers.id, id));
  revalidatePath("/advertisers");
}
