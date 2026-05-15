import { asc } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers } from "@/lib/db/schema";
import { CampaignForm } from "@/components/campaigns/campaign-form";

import { createCampaignAction } from "../_actions";

export const dynamic = "force-dynamic";

export default async function NewCampaignPage() {
  const advertiserOptions = await db
    .select({ id: advertisers.id, name: advertisers.name })
    .from(advertisers)
    .orderBy(asc(advertisers.name));

  return (
    <div className="space-y-4 max-w-lg">
      <h1 className="text-xl font-semibold tracking-tight">New campaign</h1>
      <CampaignForm
        action={createCampaignAction}
        advertisers={advertiserOptions}
        submitLabel="Create"
      />
    </div>
  );
}
