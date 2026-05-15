"use server";

import { eq } from "drizzle-orm";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { db } from "@/lib/db/client";
import { ads } from "@/lib/db/schema";
import { requireSession } from "@/lib/session";

const ALLOWED_MIMES = [
  "video/mp4",
  "application/x-mpegURL",
  "application/vnd.apple.mpegurl",
  "application/dash+xml",
] as const;

function optionalInt(label: string) {
  return z
    .string()
    .optional()
    .transform((v) => (v && v.trim() !== "" ? Number(v) : null))
    .refine((v) => v === null || (Number.isInteger(v) && v >= 0), {
      message: `${label} must be a non-negative integer`,
    });
}

const adSchema = z.object({
  campaignId: z.string().uuid({ message: "Pick a campaign" }),
  name: z.string().trim().min(1, "Name is required").max(200),
  status: z.enum(["active", "paused", "archived"]).default("active"),
  positionType: z.enum(["pre", "mid", "post"]).default("pre"),
  midRollOffset: optionalInt("Mid-roll offset"),
  priority: z
    .string()
    .optional()
    .transform((v) => (v && v.trim() !== "" ? Number(v) : 1))
    .refine((v) => Number.isInteger(v) && v >= 1, { message: "Priority must be >= 1" }),
  landingPageUrl: z
    .string()
    .optional()
    .transform((v) => (v && v.trim() !== "" ? v.trim() : null))
    .refine((v) => v === null || /^https?:\/\//.test(v), {
      message: "Landing page must be a URL",
    }),
  mediaSource: z.enum(["external_url", "internal_s3"]).default("external_url"),
  mediaUrl: z.string().trim().min(1, "Media URL/key is required"),
  mediaMime: z.string().refine((v) => (ALLOWED_MIMES as readonly string[]).includes(v), {
    message: `Mime must be one of ${ALLOWED_MIMES.join(", ")}`,
  }),
  mediaDurationMs: optionalInt("Duration"),
  mediaWidth: optionalInt("Width"),
  mediaHeight: optionalInt("Height"),
  mediaBitrateKbps: optionalInt("Bitrate"),
});

export type AdActionState = { error?: string } | undefined;

function parseForm(formData: FormData) {
  return adSchema.safeParse({
    campaignId: String(formData.get("campaignId") ?? ""),
    name: String(formData.get("name") ?? ""),
    status: String(formData.get("status") ?? "active"),
    positionType: String(formData.get("positionType") ?? "pre"),
    midRollOffset: String(formData.get("midRollOffset") ?? "") || undefined,
    priority: String(formData.get("priority") ?? "") || undefined,
    landingPageUrl: String(formData.get("landingPageUrl") ?? "") || undefined,
    mediaSource: String(formData.get("mediaSource") ?? "external_url"),
    mediaUrl: String(formData.get("mediaUrl") ?? ""),
    mediaMime: String(formData.get("mediaMime") ?? "video/mp4"),
    mediaDurationMs: String(formData.get("mediaDurationMs") ?? "") || undefined,
    mediaWidth: String(formData.get("mediaWidth") ?? "") || undefined,
    mediaHeight: String(formData.get("mediaHeight") ?? "") || undefined,
    mediaBitrateKbps: String(formData.get("mediaBitrateKbps") ?? "") || undefined,
  });
}

export async function createAdAction(
  _prev: AdActionState,
  formData: FormData,
): Promise<AdActionState> {
  await requireSession();
  const parsed = parseForm(formData);
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db.insert(ads).values(parsed.data);
  revalidatePath("/ads");
  redirect("/ads");
}

export async function updateAdAction(
  id: string,
  _prev: AdActionState,
  formData: FormData,
): Promise<AdActionState> {
  await requireSession();
  const parsed = parseForm(formData);
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? "Invalid input" };

  await db
    .update(ads)
    .set({ ...parsed.data, updatedAt: new Date() })
    .where(eq(ads.id, id));
  revalidatePath("/ads");
  redirect("/ads");
}

export async function archiveAdAction(id: string): Promise<void> {
  await requireSession();
  await db
    .update(ads)
    .set({ status: "archived", updatedAt: new Date() })
    .where(eq(ads.id, id));
  revalidatePath("/ads");
}
