// FunnelBars renders the impression -> start -> Q25 -> Q50 -> Q75 -> complete
// progression as horizontal bars whose width is proportional to the
// corresponding count divided by the impression total. Clicks render as a
// separate strip below since they aren't part of the same funnel.
//
// Pure Tailwind / inline styles — no chart library. The bar width comes
// from a numeric style prop because Tailwind has no arbitrary
// percent-width utility for runtime values.

type FunnelTotals = {
  impressions: number;
  start: number;
  q25: number;
  q50: number;
  q75: number;
  complete: number;
  clicks: number;
};

const STAGES: { label: string; key: keyof Omit<FunnelTotals, "clicks"> }[] = [
  { label: "Impression", key: "impressions" },
  { label: "Start", key: "start" },
  { label: "Q25", key: "q25" },
  { label: "Q50", key: "q50" },
  { label: "Q75", key: "q75" },
  { label: "Complete", key: "complete" },
];

function pct(n: number, d: number): number {
  if (d <= 0) return 0;
  return Math.min(100, Math.round((n / d) * 1000) / 10);
}

export function FunnelBars({ totals }: { totals: FunnelTotals }) {
  const base = totals.impressions;
  return (
    <div className="space-y-3">
      {STAGES.map((s) => {
        const count = totals[s.key];
        const width = pct(count, base);
        return (
          <div key={s.key} className="space-y-1">
            <div className="flex items-baseline justify-between text-sm">
              <span className="font-medium">{s.label}</span>
              <span className="text-zinc-500 tabular-nums">
                {count.toLocaleString()}
                {base > 0 && (
                  <span className="ml-2 text-xs text-zinc-400">{width.toFixed(1)}%</span>
                )}
              </span>
            </div>
            <div className="h-3 rounded bg-zinc-100 dark:bg-zinc-900 overflow-hidden">
              <div
                className="h-full bg-zinc-900 dark:bg-zinc-100 transition-[width]"
                style={{ width: `${width}%` }}
              />
            </div>
          </div>
        );
      })}

      <div className="pt-2 mt-2 border-t border-zinc-200 dark:border-zinc-800 space-y-1">
        <div className="flex items-baseline justify-between text-sm">
          <span className="font-medium">Clicks</span>
          <span className="text-zinc-500 tabular-nums">
            {totals.clicks.toLocaleString()}
            {base > 0 && (
              <span className="ml-2 text-xs text-zinc-400">
                {pct(totals.clicks, base).toFixed(2)}% CTR
              </span>
            )}
          </span>
        </div>
        <div className="h-3 rounded bg-zinc-100 dark:bg-zinc-900 overflow-hidden">
          <div
            className="h-full bg-blue-600 dark:bg-blue-400 transition-[width]"
            style={{ width: `${pct(totals.clicks, base)}%` }}
          />
        </div>
      </div>
    </div>
  );
}
