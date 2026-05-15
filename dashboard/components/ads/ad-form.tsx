"use client";

import { useActionState, useState } from "react";
import Link from "next/link";

import type { Ad } from "@/lib/db/schema";
import type { AdActionState } from "@/app/(app)/ads/_actions";
import { CreativePicker, type CreativeValue } from "@/components/ads/creative-picker";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

type Action = (prev: AdActionState, fd: FormData) => Promise<AdActionState>;
type CampaignOption = { id: string; name: string };

export function AdForm({
  action,
  campaigns,
  defaults,
  s3Configured,
  s3PublicBaseUrl,
  submitLabel = "Save",
}: {
  action: Action;
  campaigns: CampaignOption[];
  defaults?: Partial<Ad>;
  s3Configured: boolean;
  s3PublicBaseUrl: string | null;
  submitLabel?: string;
}) {
  const [state, formAction, isPending] = useActionState<AdActionState, FormData>(action, undefined);

  const initialCreative: CreativeValue | null = defaults?.mediaUrl
    ? {
        mediaSource: (defaults.mediaSource as "external_url" | "internal_s3") ?? "external_url",
        mediaUrl: defaults.mediaUrl,
        mediaMime: defaults.mediaMime ?? "video/mp4",
        publicUrl:
          defaults.mediaSource === "internal_s3" && s3PublicBaseUrl
            ? `${s3PublicBaseUrl.replace(/\/$/, "")}/${defaults.mediaUrl.replace(/^\//, "")}`
            : undefined,
      }
    : null;

  const [creative, setCreative] = useState<CreativeValue | null>(initialCreative);

  const selectCls =
    "w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm disabled:opacity-50";

  return (
    <form action={formAction} className="space-y-6 max-w-2xl">
      <section className="space-y-3">
        <h2 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Basics</h2>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1 col-span-2">
            <Label htmlFor="campaignId">Campaign</Label>
            <select
              id="campaignId"
              name="campaignId"
              defaultValue={defaults?.campaignId ?? ""}
              disabled={isPending || campaigns.length === 0}
              required
              className={selectCls}
            >
              <option value="" disabled>
                {campaigns.length === 0 ? "Create a campaign first" : "Pick one"}
              </option>
              {campaigns.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1 col-span-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" name="name" type="text" defaultValue={defaults?.name ?? ""} required disabled={isPending} />
          </div>

          <div className="space-y-1">
            <Label htmlFor="status">Status</Label>
            <select id="status" name="status" defaultValue={defaults?.status ?? "active"} disabled={isPending} className={selectCls}>
              <option value="active">Active</option>
              <option value="paused">Paused</option>
              <option value="archived">Archived</option>
            </select>
          </div>

          <div className="space-y-1">
            <Label htmlFor="positionType">Position</Label>
            <select id="positionType" name="positionType" defaultValue={defaults?.positionType ?? "pre"} disabled={isPending} className={selectCls}>
              <option value="pre">Pre-roll</option>
              <option value="mid">Mid-roll</option>
              <option value="post">Post-roll</option>
            </select>
          </div>

          <div className="space-y-1">
            <Label htmlFor="midRollOffset">Mid-roll offset (sec)</Label>
            <Input id="midRollOffset" name="midRollOffset" type="number" min={0} step={1} defaultValue={defaults?.midRollOffset ?? ""} disabled={isPending} />
          </div>

          <div className="space-y-1">
            <Label htmlFor="priority">Priority</Label>
            <Input id="priority" name="priority" type="number" min={1} step={1} defaultValue={defaults?.priority ?? 1} disabled={isPending} />
          </div>

          <div className="space-y-1 col-span-2">
            <Label htmlFor="landingPageUrl">Landing page URL</Label>
            <Input id="landingPageUrl" name="landingPageUrl" type="url" defaultValue={defaults?.landingPageUrl ?? ""} disabled={isPending} placeholder="https://example.com/product" />
          </div>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Creative</h2>
        <CreativePicker
          value={creative}
          onChange={setCreative}
          s3Configured={s3Configured}
          defaultPreviewPublicUrl={initialCreative?.publicUrl}
        />
        {/* Picker state drives these hidden inputs that the server action reads. */}
        <input type="hidden" name="mediaSource" value={creative?.mediaSource ?? "external_url"} />
        <input type="hidden" name="mediaUrl" value={creative?.mediaUrl ?? ""} />
        <input type="hidden" name="mediaMime" value={creative?.mediaMime ?? "video/mp4"} />
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Media metadata (optional)
        </h2>
        <p className="text-xs text-zinc-500">
          Defaults kick in when these are blank: 1280x720 dimensions and 30s duration in the VAST
          output.
        </p>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="mediaDurationMs">Duration (ms)</Label>
            <Input id="mediaDurationMs" name="mediaDurationMs" type="number" min={0} step={1} defaultValue={defaults?.mediaDurationMs ?? ""} disabled={isPending} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="mediaBitrateKbps">Bitrate (kbps)</Label>
            <Input id="mediaBitrateKbps" name="mediaBitrateKbps" type="number" min={0} step={1} defaultValue={defaults?.mediaBitrateKbps ?? ""} disabled={isPending} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="mediaWidth">Width</Label>
            <Input id="mediaWidth" name="mediaWidth" type="number" min={0} step={1} defaultValue={defaults?.mediaWidth ?? ""} disabled={isPending} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="mediaHeight">Height</Label>
            <Input id="mediaHeight" name="mediaHeight" type="number" min={0} step={1} defaultValue={defaults?.mediaHeight ?? ""} disabled={isPending} />
          </div>
        </div>
      </section>

      {state?.error && <p className="text-sm text-red-600 dark:text-red-400">{state.error}</p>}

      <div className="flex items-center gap-3 pt-2">
        <Button type="submit" disabled={isPending || campaigns.length === 0 || !creative}>
          {isPending ? "Saving..." : submitLabel}
        </Button>
        <Link href="/ads" className="text-sm text-zinc-600 dark:text-zinc-400 underline underline-offset-4">
          Cancel
        </Link>
      </div>
    </form>
  );
}
