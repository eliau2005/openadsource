import Link from "next/link";
import { notFound } from "next/navigation";
import { and, asc, eq, gte, sql } from "drizzle-orm";

import { db } from "@/lib/db/client";
import { advertisers, campaigns, dailyStats } from "@/lib/db/schema";
import { FunnelBars } from "@/components/reports/funnel-bars";
import { DailyBars, type DailyPoint } from "@/components/reports/daily-bars";

export const dynamic = "force-dynamic";

export default async function CampaignReportPage({ params }: { params: Promise<{ campaignId: string }> }) {
  const { campaignId } = await params;

  const [campaign] = await db
    .select({
      id: campaigns.id,
      name: campaigns.name,
      status: campaigns.status,
      budget: campaigns.totalBudgetImpressions,
      advertiserName: advertisers.name,
    })
    .from(campaigns)
    .leftJoin(advertisers, eq(advertisers.id, campaigns.advertiserId))
    .where(eq(campaigns.id, campaignId))
    .limit(1);
  if (!campaign) notFound();

  // Totals across the campaign's lifetime.
  const [totals] = await db
    .select({
      impressions: sql<number>`COALESCE(SUM(${dailyStats.impressions}), 0)::int`,
      clicks:      sql<number>`COALESCE(SUM(${dailyStats.clicks}), 0)::int`,
      start:       sql<number>`COALESCE(SUM(${dailyStats.startCount}), 0)::int`,
      q25:         sql<number>`COALESCE(SUM(${dailyStats.q25}), 0)::int`,
      q50:         sql<number>`COALESCE(SUM(${dailyStats.q50}), 0)::int`,
      q75:         sql<number>`COALESCE(SUM(${dailyStats.q75}), 0)::int`,
      complete:    sql<number>`COALESCE(SUM(${dailyStats.complete}), 0)::int`,
    })
    .from(dailyStats)
    .where(eq(dailyStats.campaignId, campaignId));

  // Last 30 days; pad missing dates so the bar chart stays continuous.
  const now = new Date();
  const cutoffMs = now.getTime() - 29 * 24 * 60 * 60 * 1000;
  const cutoff = new Date(cutoffMs).toISOString().slice(0, 10);
  const dailyRows = await db
    .select({
      date:        sql<string>`to_char(${dailyStats.date}, 'YYYY-MM-DD')`,
      impressions: sql<number>`SUM(${dailyStats.impressions})::int`,
      clicks:      sql<number>`SUM(${dailyStats.clicks})::int`,
      complete:    sql<number>`SUM(${dailyStats.complete})::int`,
    })
    .from(dailyStats)
    .where(and(eq(dailyStats.campaignId, campaignId), gte(sql`${dailyStats.date}::date`, sql`${cutoff}::date`)))
    .groupBy(sql`${dailyStats.date}`)
    .orderBy(asc(sql`${dailyStats.date}`));

  // Pad missing days with zeroes so the chart shows continuity.
  const byDate = new Map(dailyRows.map((r) => [r.date, r]));
  const points: DailyPoint[] = [];
  for (let i = 0; i < 30; i++) {
    const d = new Date(cutoffMs + i * 24 * 60 * 60 * 1000);
    const date = d.toISOString().slice(0, 10);
    const row = byDate.get(date);
    points.push({
      date,
      impressions: row?.impressions ?? 0,
      clicks: row?.clicks ?? 0,
      complete: row?.complete ?? 0,
    });
  }

  const ctr = totals.impressions > 0 ? (totals.clicks / totals.impressions) * 100 : 0;
  const completion = totals.impressions > 0 ? (totals.complete / totals.impressions) * 100 : 0;
  const spent = campaign.budget && campaign.budget > 0 ? Math.min(100, (totals.impressions / campaign.budget) * 100) : 0;

  return (
    <div className="space-y-6 max-w-5xl">
      <div>
        <Link href="/reports" className="text-sm text-zinc-500 underline underline-offset-4">← All reports</Link>
        <h1 className="text-xl font-semibold tracking-tight mt-2">{campaign.name}</h1>
        <div className="text-sm text-zinc-500 mt-1">
          {campaign.advertiserName} · <span className="capitalize">{campaign.status}</span>
          {campaign.budget && campaign.budget > 0 ? <> · Budget {campaign.budget.toLocaleString()} impressions</> : <> · Unlimited budget</>}
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
        <Card label="Impressions" value={totals.impressions.toLocaleString()} />
        <Card label="Clicks (CTR)" value={`${totals.clicks.toLocaleString()}`} sub={totals.impressions > 0 ? `${ctr.toFixed(2)}%` : ""} />
        <Card label="Completion" value={`${totals.complete.toLocaleString()}`} sub={totals.impressions > 0 ? `${completion.toFixed(1)}%` : ""} />
        <BudgetCard impressions={totals.impressions} budget={campaign.budget} spentPct={spent} />
      </div>

      <section className="rounded-md border border-zinc-200 dark:border-zinc-800 p-4">
        <h2 className="text-sm font-medium mb-3">Quartile funnel</h2>
        <FunnelBars totals={totals} />
      </section>

      <section className="rounded-md border border-zinc-200 dark:border-zinc-800 p-4">
        <h2 className="text-sm font-medium mb-3">Last 30 days · daily impressions</h2>
        <DailyBars points={points} />
      </section>
    </div>
  );
}

function Card({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-md border border-zinc-200 dark:border-zinc-800 p-4">
      <div className="text-xs uppercase tracking-wider text-zinc-500">{label}</div>
      <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
      {sub ? <div className="text-xs text-zinc-500 mt-1">{sub}</div> : null}
    </div>
  );
}

function BudgetCard({ impressions, budget, spentPct }: { impressions: number; budget: number | null; spentPct: number }) {
  if (!budget || budget <= 0) {
    return (
      <div className="rounded-md border border-zinc-200 dark:border-zinc-800 p-4">
        <div className="text-xs uppercase tracking-wider text-zinc-500">Spent</div>
        <div className="mt-1 text-2xl font-semibold tabular-nums">{impressions.toLocaleString()}</div>
        <div className="text-xs text-zinc-500 mt-1">no budget cap</div>
      </div>
    );
  }
  return (
    <div className="rounded-md border border-zinc-200 dark:border-zinc-800 p-4">
      <div className="text-xs uppercase tracking-wider text-zinc-500">Spent</div>
      <div className="mt-1 text-2xl font-semibold tabular-nums">{impressions.toLocaleString()}</div>
      <div className="text-xs text-zinc-500 mt-1">of {budget.toLocaleString()} · {spentPct.toFixed(1)}%</div>
      <div className="mt-2 h-2 rounded bg-zinc-100 dark:bg-zinc-900 overflow-hidden">
        <div className="h-full bg-zinc-900 dark:bg-zinc-100 transition-[width]" style={{ width: `${spentPct}%` }} />
      </div>
    </div>
  );
}
