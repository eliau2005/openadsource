// POST /api/upload — mint a short-lived presigned PUT URL the browser uses
// to upload directly to MinIO/S3. The dashboard never sees the bytes.
import { NextResponse, type NextRequest } from "next/server";
import { randomUUID } from "node:crypto";
import { z } from "zod";

import { ALLOWED_MIMES } from "@/lib/media";
import { presignPut, publicUrlFor, getS3Client } from "@/lib/s3";
import { requireSession } from "@/lib/session";

const bodySchema = z.object({
  filename: z.string().min(1).max(256),
  contentType: z.enum(ALLOWED_MIMES),
});

const PRESIGN_TTL_SECONDS = 15 * 60;

function safeFilename(input: string): string {
  // Strip anything not in the conservative set; keep extension if present.
  return input.replace(/[^a-zA-Z0-9._-]+/g, "_").replace(/_+/g, "_").slice(0, 200);
}

function todayPrefix(): string {
  const d = new Date();
  return `${d.getUTCFullYear()}${String(d.getUTCMonth() + 1).padStart(2, "0")}${String(d.getUTCDate()).padStart(2, "0")}`;
}

export async function POST(request: NextRequest) {
  await requireSession();

  if (!getS3Client()) {
    return NextResponse.json(
      { error: "S3 is not configured on this deployment" },
      { status: 501 },
    );
  }

  let parsed: z.infer<typeof bodySchema>;
  try {
    const json = await request.json();
    const v = bodySchema.safeParse(json);
    if (!v.success) {
      return NextResponse.json(
        { error: v.error.issues[0]?.message ?? "Invalid body" },
        { status: 400 },
      );
    }
    parsed = v.data;
  } catch {
    return NextResponse.json({ error: "Body must be JSON" }, { status: 400 });
  }

  const key = `uploads/${todayPrefix()}/${randomUUID()}-${safeFilename(parsed.filename)}`;
  const signed = await presignPut(key, parsed.contentType, PRESIGN_TTL_SECONDS);
  if (!signed) {
    return NextResponse.json({ error: "S3 presign failed" }, { status: 500 });
  }

  return NextResponse.json({
    url: signed.url,
    key: signed.key,
    publicUrl: publicUrlFor(signed.key),
    expiresIn: PRESIGN_TTL_SECONDS,
  });
}
