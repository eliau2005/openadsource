export function Topbar({ email }: { email: string }) {
  return (
    <header className="h-12 border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 px-4 flex items-center justify-end gap-4 text-sm">
      <span className="text-zinc-500">{email}</span>
      <a
        href="/api/auth/logout"
        className="text-zinc-700 dark:text-zinc-300 hover:text-zinc-950 dark:hover:text-zinc-50 underline underline-offset-4"
      >
        Sign out
      </a>
    </header>
  );
}
