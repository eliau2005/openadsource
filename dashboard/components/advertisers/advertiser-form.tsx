"use client";

import { useActionState } from "react";
import Link from "next/link";

import type { Advertiser } from "@/lib/db/schema";
import type { AdvertiserActionState } from "@/app/(app)/advertisers/_actions";

type Action = (prev: AdvertiserActionState, fd: FormData) => Promise<AdvertiserActionState>;

export function AdvertiserForm({
  action,
  defaults,
  submitLabel = "Save",
}: {
  action: Action;
  defaults?: Pick<Advertiser, "name" | "status">;
  submitLabel?: string;
}) {
  const [state, formAction, isPending] = useActionState<AdvertiserActionState, FormData>(
    action,
    undefined,
  );

  return (
    <form action={formAction} className="space-y-4 max-w-lg">
      <div className="space-y-1">
        <label htmlFor="name" className="block text-sm font-medium">Name</label>
        <input
          id="name"
          name="name"
          type="text"
          defaultValue={defaults?.name ?? ""}
          required
          disabled={isPending}
          className="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm disabled:opacity-50"
        />
      </div>

      <div className="space-y-1">
        <label htmlFor="status" className="block text-sm font-medium">Status</label>
        <select
          id="status"
          name="status"
          defaultValue={defaults?.status ?? "active"}
          disabled={isPending}
          className="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm disabled:opacity-50"
        >
          <option value="active">Active</option>
          <option value="archived">Archived</option>
        </select>
      </div>

      {state?.error && <p className="text-sm text-red-600 dark:text-red-400">{state.error}</p>}

      <div className="flex items-center gap-3 pt-2">
        <button
          type="submit"
          disabled={isPending}
          className="rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {isPending ? "Saving..." : submitLabel}
        </button>
        <Link
          href="/advertisers"
          className="text-sm text-zinc-600 dark:text-zinc-400 underline underline-offset-4"
        >
          Cancel
        </Link>
      </div>
    </form>
  );
}
