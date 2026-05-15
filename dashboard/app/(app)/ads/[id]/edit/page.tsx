import { notFound } from "next/navigation";
import { asc, eq } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { ads, campaigns } from "@/lib/db/schema";
import { AdForm } from "@/components/ads/ad-form";

import { updateAdAction } from "../../_actions";

export const dynamic = "force-dynamic";

export default async function EditAdPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [row] = await db.select().from(ads).where(eq(ads.id, id)).limit(1);
  if (!row) notFound();

  const campaignOptions = await db
    .select({ id: campaigns.id, name: campaigns.name })
    .from(campaigns)
    .orderBy(asc(campaigns.name));

  return (
    <div className="space-y-4 max-w-2xl">
      <h1 className="text-xl font-semibold tracking-tight">Edit ad</h1>
      <AdForm
        action={updateAdAction.bind(null, id)}
        campaigns={campaignOptions}
        defaults={row}
      />
    </div>
  );
}
