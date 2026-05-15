import { AdvertiserForm } from "@/components/advertisers/advertiser-form";
import { createAdvertiserAction } from "../_actions";

export const dynamic = "force-dynamic";

export default function NewAdvertiserPage() {
  return (
    <div className="space-y-4 max-w-lg">
      <h1 className="text-xl font-semibold tracking-tight">New advertiser</h1>
      <AdvertiserForm action={createAdvertiserAction} submitLabel="Create" />
    </div>
  );
}
