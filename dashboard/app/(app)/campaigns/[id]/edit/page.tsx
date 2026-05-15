import { notFound } from "next/navigation";
import { asc, eq } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers, campaigns } from "@/lib/db/schema";
import { CampaignForm } from "@/components/campaigns/campaign-form";

import { updateCampaignAction } from "../../_actions";

export const dynamic = "force-dynamic";

export default async function EditCampaignPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [row] = await db.select().from(campaigns).where(eq(campaigns.id, id)).limit(1);
  if (!row) notFound();

  const advertiserOptions = await db
    .select({ id: advertisers.id, name: advertisers.name })
    .from(advertisers)
    .orderBy(asc(advertisers.name));

  return (
    <div className="space-y-4 max-w-lg">
      <h1 className="text-xl font-semibold tracking-tight">Edit campaign</h1>
      <CampaignForm
        action={updateCampaignAction.bind(null, id)}
        advertisers={advertiserOptions}
        defaults={row}
      />
    </div>
  );
}
