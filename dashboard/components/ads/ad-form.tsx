"use client";

import { useActionState } from "react";
import Link from "next/link";

import type { Ad } from "@/lib/db/schema";
import type { AdActionState } from "@/app/(app)/ads/_actions";

type Action = (prev: AdActionState, fd: FormData) => Promise<AdActionState>;
type CampaignOption = { id: string; name: string };

export function AdForm({
  action,
  campaigns,
  defaults,
  submitLabel = "Save",
}: {
  action: Action;
  campaigns: CampaignOption[];
  defaults?: Partial<Ad>;
  submitLabel?: string;
}) {
  const [state, formAction, isPending] = useActionState<AdActionState, FormData>(action, undefined);

  const inputCls =
    "w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm disabled:opacity-50";

  return (
    <form action={formAction} className="space-y-4 max-w-2xl">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1 col-span-2">
          <label htmlFor="campaignId" className="block text-sm font-medium">Campaign</label>
          <select
            id="campaignId"
            name="campaignId"
            defaultValue={defaults?.campaignId ?? ""}
            disabled={isPending || campaigns.length === 0}
            required
            className={inputCls}
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
          <label htmlFor="name" className="block text-sm font-medium">Name</label>
          <input
            id="name"
            name="name"
            type="text"
            defaultValue={defaults?.name ?? ""}
            required
            disabled={isPending}
            className={inputCls}
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="status" className="block text-sm font-medium">Status</label>
          <select id="status" name="status" defaultValue={defaults?.status ?? "active"} disabled={isPending} className={inputCls}>
            <option value="active">Active</option>
            <option value="paused">Paused</option>
            <option value="archived">Archived</option>
          </select>
        </div>

        <div className="space-y-1">
          <label htmlFor="positionType" className="block text-sm font-medium">Position</label>
          <select
            id="positionType"
            name="positionType"
            defaultValue={defaults?.positionType ?? "pre"}
            disabled={isPending}
            className={inputCls}
          >
            <option value="pre">Pre-roll</option>
            <option value="mid">Mid-roll</option>
            <option value="post">Post-roll</option>
          </select>
        </div>

        <div className="space-y-1">
          <label htmlFor="midRollOffset" className="block text-sm font-medium">Mid-roll offset (sec)</label>
          <input
            id="midRollOffset"
            name="midRollOffset"
            type="number"
            min={0}
            step={1}
            defaultValue={defaults?.midRollOffset ?? ""}
            disabled={isPending}
            className={inputCls}
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="priority" className="block text-sm font-medium">Priority</label>
          <input
            id="priority"
            name="priority"
            type="number"
            min={1}
            step={1}
            defaultValue={defaults?.priority ?? 1}
            disabled={isPending}
            className={inputCls}
          />
        </div>

        <div className="space-y-1 col-span-2">
          <label htmlFor="landingPageUrl" className="block text-sm font-medium">Landing page URL</label>
          <input
            id="landingPageUrl"
            name="landingPageUrl"
            type="url"
            defaultValue={defaults?.landingPageUrl ?? ""}
            disabled={isPending}
            placeholder="https://example.com/product"
            className={inputCls}
          />
        </div>
      </div>

      <fieldset className="border border-zinc-200 dark:border-zinc-800 rounded-md p-4 space-y-3">
        <legend className="text-sm font-medium px-1">Creative</legend>
        <p className="text-xs text-zinc-500">
          The dual-tab BYO / Upload picker lands in the next commit. For now, enter the URL or
          object key directly. For <code>internal_s3</code>, the URL field holds the S3 object
          key (e.g. <code>uploads/2026/clip.mp4</code>).
        </p>

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <label htmlFor="mediaSource" className="block text-sm font-medium">Source</label>
            <select
              id="mediaSource"
              name="mediaSource"
              defaultValue={defaults?.mediaSource ?? "external_url"}
              disabled={isPending}
              className={inputCls}
            >
              <option value="external_url">External URL (BYO)</option>
              <option value="internal_s3">Internal S3 (uploaded)</option>
            </select>
          </div>

          <div className="space-y-1">
            <label htmlFor="mediaMime" className="block text-sm font-medium">Mime type</label>
            <select
              id="mediaMime"
              name="mediaMime"
              defaultValue={defaults?.mediaMime ?? "video/mp4"}
              disabled={isPending}
              className={inputCls}
            >
              <option value="video/mp4">video/mp4</option>
              <option value="application/x-mpegURL">application/x-mpegURL (HLS)</option>
              <option value="application/vnd.apple.mpegurl">application/vnd.apple.mpegurl (HLS)</option>
              <option value="application/dash+xml">application/dash+xml (DASH)</option>
            </select>
          </div>

          <div className="space-y-1 col-span-2">
            <label htmlFor="mediaUrl" className="block text-sm font-medium">Media URL or S3 key</label>
            <input
              id="mediaUrl"
              name="mediaUrl"
              type="text"
              defaultValue={defaults?.mediaUrl ?? ""}
              required
              disabled={isPending}
              placeholder="https://cdn.example.com/clip.mp4   or   uploads/2026/clip.mp4"
              className={inputCls}
            />
          </div>

          <div className="space-y-1">
            <label htmlFor="mediaDurationMs" className="block text-sm font-medium">Duration (ms)</label>
            <input id="mediaDurationMs" name="mediaDurationMs" type="number" min={0} step={1} defaultValue={defaults?.mediaDurationMs ?? ""} disabled={isPending} className={inputCls} />
          </div>
          <div className="space-y-1">
            <label htmlFor="mediaBitrateKbps" className="block text-sm font-medium">Bitrate (kbps)</label>
            <input id="mediaBitrateKbps" name="mediaBitrateKbps" type="number" min={0} step={1} defaultValue={defaults?.mediaBitrateKbps ?? ""} disabled={isPending} className={inputCls} />
          </div>
          <div className="space-y-1">
            <label htmlFor="mediaWidth" className="block text-sm font-medium">Width</label>
            <input id="mediaWidth" name="mediaWidth" type="number" min={0} step={1} defaultValue={defaults?.mediaWidth ?? ""} disabled={isPending} className={inputCls} />
          </div>
          <div className="space-y-1">
            <label htmlFor="mediaHeight" className="block text-sm font-medium">Height</label>
            <input id="mediaHeight" name="mediaHeight" type="number" min={0} step={1} defaultValue={defaults?.mediaHeight ?? ""} disabled={isPending} className={inputCls} />
          </div>
        </div>
      </fieldset>

      {state?.error && <p className="text-sm text-red-600 dark:text-red-400">{state.error}</p>}

      <div className="flex items-center gap-3 pt-2">
        <button
          type="submit"
          disabled={isPending || campaigns.length === 0}
          className="rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {isPending ? "Saving..." : submitLabel}
        </button>
        <Link
          href="/ads"
          className="text-sm text-zinc-600 dark:text-zinc-400 underline underline-offset-4"
        >
          Cancel
        </Link>
      </div>
    </form>
  );
}
