import Link from "next/link";
import { eq, sql } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers, campaigns, dailyStats } from "@/lib/db/schema";

export const dynamic = "force-dynamic";

export default async function ReportsListPage() {
  // Aggregate over all dates per campaign. Includes campaigns with no
  // stats yet (LEFT JOIN) so an operator can see what's expected even
  // before the worker has flushed the first tick.
  const rows = await db
    .select({
      campaignId: campaigns.id,
      campaignName: campaigns.name,
      advertiserName: advertisers.name,
      campaignStatus: campaigns.status,
      budget: campaigns.totalBudgetImpressions,
      impressions: sql<number>`COALESCE(SUM(${dailyStats.impressions}), 0)::int`,
      clicks: sql<number>`COALESCE(SUM(${dailyStats.clicks}), 0)::int`,
      completes: sql<number>`COALESCE(SUM(${dailyStats.complete}), 0)::int`,
    })
    .from(campaigns)
    .leftJoin(advertisers, eq(advertisers.id, campaigns.advertiserId))
    .leftJoin(dailyStats, eq(dailyStats.campaignId, campaigns.id))
    .groupBy(campaigns.id, campaigns.name, advertisers.name, campaigns.status, campaigns.totalBudgetImpressions)
    .orderBy(sql`SUM(${dailyStats.impressions}) DESC NULLS LAST`);

  return (
    <div className="space-y-4 max-w-5xl">
      <h1 className="text-xl font-semibold tracking-tight">Reports</h1>
      <p className="text-sm text-zinc-500">Totals across the lifetime of each campaign. Click any row for the funnel and 30-day breakdown.</p>

      {rows.length === 0 ? (
        <div className="rounded-md border border-dashed border-zinc-300 dark:border-zinc-700 p-8 text-center text-sm text-zinc-500">
          No campaigns yet. Create one in <Link href="/campaigns" className="underline">/campaigns</Link>, then load it in the test player to start collecting events.
        </div>
      ) : (
        <div className="rounded-md border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 dark:bg-zinc-900 text-left text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Campaign</th>
                <th className="px-4 py-2 font-medium">Advertiser</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium text-right">Impressions</th>
                <th className="px-4 py-2 font-medium text-right">CTR</th>
                <th className="px-4 py-2 font-medium text-right">Completion</th>
                <th className="px-4 py-2 font-medium w-20" />
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {rows.map((r) => {
                const ctr = r.impressions > 0 ? (r.clicks / r.impressions) * 100 : 0;
                const comp = r.impressions > 0 ? (r.completes / r.impressions) * 100 : 0;
                return (
                  <tr key={r.campaignId}>
                    <td className="px-4 py-2">{r.campaignName}</td>
                    <td className="px-4 py-2 text-zinc-500">{r.advertiserName}</td>
                    <td className="px-4 py-2 capitalize">{r.campaignStatus}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{r.impressions.toLocaleString()}</td>
                    <td className="px-4 py-2 text-right tabular-nums text-zinc-500">{ctr.toFixed(2)}%</td>
                    <td className="px-4 py-2 text-right tabular-nums text-zinc-500">{comp.toFixed(1)}%</td>
                    <td className="px-4 py-2 text-right">
                      <Link href={`/reports/${r.campaignId}`} className="text-zinc-700 dark:text-zinc-300 underline underline-offset-4">View</Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
