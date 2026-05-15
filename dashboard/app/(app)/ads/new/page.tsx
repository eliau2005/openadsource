import { asc } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { campaigns } from "@/lib/db/schema";
import { env, s3Configured } from "@/lib/env";
import { AdForm } from "@/components/ads/ad-form";

import { createAdAction } from "../_actions";

export const dynamic = "force-dynamic";

export default async function NewAdPage() {
  const campaignOptions = await db
    .select({ id: campaigns.id, name: campaigns.name })
    .from(campaigns)
    .orderBy(asc(campaigns.name));

  return (
    <div className="space-y-4 max-w-2xl">
      <h1 className="text-xl font-semibold tracking-tight">New ad</h1>
      <AdForm
        action={createAdAction}
        campaigns={campaignOptions}
        s3Configured={s3Configured}
        s3PublicBaseUrl={env.S3_PUBLIC_BASE_URL ?? null}
        submitLabel="Create"
      />
    </div>
  );
}
