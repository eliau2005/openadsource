import Link from "next/link";

const nav = [
  { href: "/campaigns", label: "Campaigns" },
  { href: "/advertisers", label: "Advertisers" },
  { href: "/ads", label: "Ads" },
  { href: "/reports", label: "Reports" },
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
      </nav>
    </aside>
  );
}
