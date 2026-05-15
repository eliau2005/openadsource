import Link from "next/link";

const nav = [
  { href: "/campaigns", label: "Campaigns" },
  { href: "/advertisers", label: "Advertisers" },
  { href: "/ads", label: "Ads" },
] as const;

const placeholderNav = [
  { href: "#", label: "Reports", badge: "Phase 4" },
] as const;

export function Sidebar() {
  return (
    <aside className="w-56 shrink-0 border-r border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950">
      <div className="px-4 py-4 border-b border-zinc-200 dark:border-zinc-800">
        <Link
          href="/campaigns"
          className="text-sm font-semibold tracking-tight text-zinc-950 dark:text-zinc-50"
        >
          OpenAdSource
        </Link>
      </div>

      <nav className="px-2 py-3 space-y-1 text-sm">
        {nav.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="block rounded-md px-3 py-2 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-900"
          >
            {item.label}
          </Link>
        ))}

        <div className="pt-3 mt-3 border-t border-zinc-100 dark:border-zinc-900" />

        {placeholderNav.map((item) => (
          <div
            key={item.label}
            className="flex items-center justify-between rounded-md px-3 py-2 text-zinc-400 dark:text-zinc-600 cursor-not-allowed"
          >
            <span>{item.label}</span>
            <span className="text-[10px] uppercase tracking-wider rounded bg-zinc-100 dark:bg-zinc-900 px-1.5 py-0.5">
              {item.badge}
            </span>
          </div>
        ))}
      </nav>
    </aside>
  );
}
