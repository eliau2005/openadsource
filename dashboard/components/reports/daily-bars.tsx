// DailyBars renders one short vertical bar per day in the supplied window.
// Each bar's height is proportional to that day's impressions divided by
// the 30-day max. Hover tooltip shows the exact counts.
//
// Pure Tailwind + inline height style — no chart library.

export type DailyPoint = {
  date: string;            // YYYY-MM-DD
  impressions: number;
  clicks: number;
  complete: number;
};

const BAR_MAX_PX = 96; // 6rem

export function DailyBars({ points }: { points: DailyPoint[] }) {
  const max = points.reduce((m, p) => (p.impressions > m ? p.impressions : m), 0);
  if (points.length === 0 || max === 0) {
    return (
      <div className="text-sm text-zinc-500">No daily data yet — fire an impression through /track and wait one worker tick.</div>
    );
  }
  return (
    <div>
      <div className="flex items-end gap-1 h-24 border-b border-zinc-200 dark:border-zinc-800">
        {points.map((p) => {
          const h = Math.max(2, Math.round((p.impressions / max) * BAR_MAX_PX));
          return (
            <div
              key={p.date}
              className="flex-1 bg-zinc-900 dark:bg-zinc-100 rounded-t"
              style={{ height: `${h}px` }}
              title={`${p.date}\nImpressions: ${p.impressions}\nClicks: ${p.clicks}\nComplete: ${p.complete}`}
            />
          );
        })}
      </div>
      <div className="mt-1 flex justify-between text-[10px] text-zinc-400 tabular-nums">
        <span>{points[0]?.date}</span>
        <span>{points[points.length - 1]?.date}</span>
      </div>
    </div>
  );
}
