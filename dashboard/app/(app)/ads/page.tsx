import Link from "next/link";
import { desc, eq } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { ads, campaigns } from "@/lib/db/schema";

import { archiveAdAction } from "./_actions";

export const dynamic = "force-dynamic";

export default async function AdsListPage() {
  const rows = await db
    .select({
      id: ads.id,
      name: ads.name,
      status: ads.status,
      position: ads.positionType,
      priority: ads.priority,
      mediaSource: ads.mediaSource,
      mediaUrl: ads.mediaUrl,
      campaignName: campaigns.name,
      createdAt: ads.createdAt,
    })
    .from(ads)
    .leftJoin(campaigns, eq(campaigns.id, ads.campaignId))
    .orderBy(desc(ads.createdAt));

  return (
    <div className="space-y-4 max-w-6xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Ads</h1>
        <Link
          href="/ads/new"
          className="rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-3 py-1.5 text-sm font-medium"
        >
          New ad
        </Link>
      </div>

      {rows.length === 0 ? (
        <div className="rounded-md border border-dashed border-zinc-300 dark:border-zinc-700 p-8 text-center text-sm text-zinc-500">
          No ads yet. <Link href="/ads/new" className="underline">Create your first one</Link>.
        </div>
      ) : (
        <div className="rounded-md border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 dark:bg-zinc-900 text-left text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Name</th>
                <th className="px-4 py-2 font-medium">Campaign</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Pos</th>
                <th className="px-4 py-2 font-medium">Source</th>
                <th className="px-4 py-2 font-medium w-32" />
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {rows.map((row) => (
                <tr key={row.id}>
                  <td className="px-4 py-2">
                    <div>{row.name}</div>
                    <div className="text-xs text-zinc-400 font-mono">{row.id}</div>
                  </td>
                  <td className="px-4 py-2 text-zinc-500">{row.campaignName}</td>
                  <td className="px-4 py-2 capitalize">{row.status}</td>
                  <td className="px-4 py-2 capitalize">{row.position}</td>
                  <td className="px-4 py-2 text-zinc-500">
                    {row.mediaSource === "external_url" ? "external" : "internal"}
                  </td>
                  <td className="px-4 py-2 text-right space-x-3">
                    <Link
                      href={`/ads/${row.id}/edit`}
                      className="text-zinc-700 dark:text-zinc-300 underline underline-offset-4"
                    >
                      Edit
                    </Link>
                    {row.status !== "archived" && (
                      <form action={archiveAdAction.bind(null, row.id)} className="inline">
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
