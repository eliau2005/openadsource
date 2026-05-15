"use client";

import { useActionState } from "react";
import Link from "next/link";

import type { Campaign } from "@/lib/db/schema";
import type { CampaignActionState } from "@/app/(app)/campaigns/_actions";

type Action = (prev: CampaignActionState, fd: FormData) => Promise<CampaignActionState>;

type AdvertiserOption = { id: string; name: string };

function dateForInput(d: Date | null | undefined): string {
  if (!d) return "";
  // <input type="date"> wants YYYY-MM-DD; toISOString gives UTC which is fine
  // for dev. Production usage may want timezone awareness — Phase 5 polish.
  return d.toISOString().slice(0, 10);
}

export function CampaignForm({
  action,
  advertisers,
  defaults,
  submitLabel = "Save",
}: {
  action: Action;
  advertisers: AdvertiserOption[];
  defaults?: Partial<Campaign>;
  submitLabel?: string;
}) {
  const [state, formAction, isPending] = useActionState<CampaignActionState, FormData>(
    action,
    undefined,
  );

  const inputCls =
    "w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm disabled:opacity-50";

  return (
    <form action={formAction} className="space-y-4 max-w-lg">
      <div className="space-y-1">
        <label htmlFor="advertiserId" className="block text-sm font-medium">Advertiser</label>
        <select
          id="advertiserId"
          name="advertiserId"
          defaultValue={defaults?.advertiserId ?? ""}
          disabled={isPending || advertisers.length === 0}
          required
          className={inputCls}
        >
          <option value="" disabled>
            {advertisers.length === 0 ? "Create an advertiser first" : "Pick one"}
          </option>
          {advertisers.map((a) => (
            <option key={a.id} value={a.id}>{a.name}</option>
          ))}
        </select>
      </div>

      <div className="space-y-1">
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

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label htmlFor="startDate" className="block text-sm font-medium">Start date</label>
          <input
            id="startDate"
            name="startDate"
            type="date"
            defaultValue={dateForInput(defaults?.startDate as Date | null | undefined)}
            disabled={isPending}
            className={inputCls}
          />
        </div>
        <div className="space-y-1">
          <label htmlFor="endDate" className="block text-sm font-medium">End date</label>
          <input
            id="endDate"
            name="endDate"
            type="date"
            defaultValue={dateForInput(defaults?.endDate as Date | null | undefined)}
            disabled={isPending}
            className={inputCls}
          />
        </div>
      </div>

      <div className="space-y-1">
        <label htmlFor="totalBudgetImpressions" className="block text-sm font-medium">
          Total budget (impressions, optional)
        </label>
        <input
          id="totalBudgetImpressions"
          name="totalBudgetImpressions"
          type="number"
          min={0}
          step={1}
          defaultValue={defaults?.totalBudgetImpressions ?? ""}
          disabled={isPending}
          className={inputCls}
        />
      </div>

      <div className="space-y-1">
        <label htmlFor="status" className="block text-sm font-medium">Status</label>
        <select
          id="status"
          name="status"
          defaultValue={defaults?.status ?? "active"}
          disabled={isPending}
          className={inputCls}
        >
          <option value="active">Active</option>
          <option value="paused">Paused</option>
          <option value="completed">Completed</option>
          <option value="archived">Archived</option>
        </select>
      </div>

      {state?.error && <p className="text-sm text-red-600 dark:text-red-400">{state.error}</p>}

      <div className="flex items-center gap-3 pt-2">
        <button
          type="submit"
          disabled={isPending || advertisers.length === 0}
          className="rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {isPending ? "Saving..." : submitLabel}
        </button>
        <Link
          href="/campaigns"
          className="text-sm text-zinc-600 dark:text-zinc-400 underline underline-offset-4"
        >
          Cancel
        </Link>
      </div>
    </form>
  );
}
