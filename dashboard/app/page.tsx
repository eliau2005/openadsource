export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 bg-zinc-50 px-6 text-center dark:bg-black">
      <h1 className="text-3xl font-semibold tracking-tight text-zinc-950 dark:text-zinc-50">
        OpenAdSource
      </h1>
      <p className="max-w-md text-zinc-600 dark:text-zinc-400">
        Phase 0 scaffold. The campaign management UI lands in Phase 2 of the
        roadmap.
      </p>
      <a
        href="/api/healthz"
        className="text-sm text-zinc-500 underline underline-offset-4 hover:text-zinc-700 dark:hover:text-zinc-300"
      >
        /api/healthz
      </a>
    </main>
  );
}
