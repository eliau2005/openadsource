import { notFound } from "next/navigation";
import { eq } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers } from "@/lib/db/schema";
import { AdvertiserForm } from "@/components/advertisers/advertiser-form";
import { updateAdvertiserAction } from "../../_actions";

export const dynamic = "force-dynamic";

export default async function EditAdvertiserPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [row] = await db.select().from(advertisers).where(eq(advertisers.id, id)).limit(1);
  if (!row) notFound();

  return (
    <div className="space-y-4 max-w-lg">
      <h1 className="text-xl font-semibold tracking-tight">Edit advertiser</h1>
      <AdvertiserForm
        action={updateAdvertiserAction.bind(null, id)}
        defaults={{ name: row.name, status: row.status }}
      />
    </div>
  );
}
