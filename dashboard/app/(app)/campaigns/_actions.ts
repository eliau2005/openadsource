"use server";

import { eq } from "drizzle-orm";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { db } from "@/lib/db/client";
import { campaigns } from "@/lib/db/schema";
import { requireSession } from "@/lib/session";

// Permissive UUID regex — Zod 4's .uuid() rejects non-versioned UUIDs like
// the all-zeros / all-ones sentinels we use in seed data, but Postgres
// accepts them happily.
const UUID_REGEX = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

const campaignSchema = z.object({
  advertiserId: z.string().regex(UUID_REGEX, { message: "Pick an advertiser" }),
  name: z.string().trim().min(1, "Name is required").max(200),
  startDate: z
    .string()
    .optional()
    .transform((v) => (v ? new Date(v) : null)),
  endDate: z
    .string()
    .optional()
    .transform((v) => (v ? new Date(v) : null)),
  totalBudgetImpressions: z
    .string()
    .optional()
    .transform((v) => (v && v.trim() !== "" ? Number(v) : null))
    .refine((v) => v === null || (Number.isInteger(v) && v >= 0), {
      message: "Budget must be a non-negative integer",
    }),
  status: z.enum(["active", "paused", "completed", "archived"]).default("active"),
});

export type CampaignActionState = { error?: string } | undefined;

function parseForm(formData: FormData) {
  return campaignSchema.safeParse({
    advertiserId: String(formData.get("advertiserId") ?? ""),
    name: String(formData.get("name") ?? ""),
    startDate: String(formData.get("startDate") ?? "") || undefined,
    endDate: String(formData.get("endDate") ?? "") || undefined,
    totalBudgetImpressions: String(formData.get("totalBudgetImpressions") ?? "") || undefined,
    status: String(formData.get("status") ?? "active"),
  });
}

export async function createCampaignAction(
  _prev: CampaignActionState,
  formData: FormData,
): Promise<CampaignActionState> {
  await requireSession();
  const parsed = parseForm(formData);
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db.insert(campaigns).values(parsed.data);
  revalidatePath("/campaigns");
  redirect("/campaigns");
}

export async function updateCampaignAction(
  id: string,
  _prev: CampaignActionState,
  formData: FormData,
): Promise<CampaignActionState> {
  await requireSession();
  const parsed = parseForm(formData);
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db
    .update(campaigns)
    .set({ ...parsed.data, updatedAt: new Date() })
    .where(eq(campaigns.id, id));
  revalidatePath("/campaigns");
  redirect("/campaigns");
}

export async function archiveCampaignAction(id: string): Promise<void> {
  await requireSession();
  await db
    .update(campaigns)
    .set({ status: "archived", updatedAt: new Date() })
    .where(eq(campaigns.id, id));
  revalidatePath("/campaigns");
}
