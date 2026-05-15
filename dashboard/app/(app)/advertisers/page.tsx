import Link from "next/link";
import { desc } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers } from "@/lib/db/schema";

import { archiveAdvertiserAction } from "./_actions";

export const dynamic = "force-dynamic";

export default async function AdvertisersListPage() {
  const rows = await db.select().from(advertisers).orderBy(desc(advertisers.createdAt));

  return (
    <div className="space-y-4 max-w-5xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Advertisers</h1>
        <Link
          href="/advertisers/new"
          className="rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-3 py-1.5 text-sm font-medium"
        >
          New advertiser
        </Link>
      </div>

      {rows.length === 0 ? (
        <div className="rounded-md border border-dashed border-zinc-300 dark:border-zinc-700 p-8 text-center text-sm text-zinc-500">
          No advertisers yet. <Link href="/advertisers/new" className="underline">Create your first one</Link>.
        </div>
      ) : (
        <div className="rounded-md border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 dark:bg-zinc-900 text-left text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Name</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Created</th>
                <th className="px-4 py-2 font-medium w-32" />
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {rows.map((row) => (
                <tr key={row.id}>
                  <td className="px-4 py-2">{row.name}</td>
                  <td className="px-4 py-2 capitalize">{row.status}</td>
                  <td className="px-4 py-2 text-zinc-500">
                    {row.createdAt.toLocaleDateString()}
                  </td>
                  <td className="px-4 py-2 text-right space-x-3">
                    <Link
                      href={`/advertisers/${row.id}/edit`}
                      className="text-zinc-700 dark:text-zinc-300 underline underline-offset-4"
                    >
                      Edit
                    </Link>
                    {row.status !== "archived" && (
                      <form action={archiveAdvertiserAction.bind(null, row.id)} className="inline">
                        <button
                          type="submit"
                          className="text-red-600 dark:text-red-400 underline underline-offset-4"
                        >
                          Archive
                        </button>
                      </form>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
