import { requireSession } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function CampaignsPlaceholderPage() {
  const session = await requireSession();
  return (
    <div className="min-h-screen p-8 max-w-4xl mx-auto">
      <h1 className="text-2xl font-semibold tracking-tight">Campaigns</h1>
      <p className="mt-2 text-sm text-zinc-500">
        Signed in as <code className="font-mono">{session.email}</code>.
      </p>
      <p className="mt-6 text-zinc-600 dark:text-zinc-400">
        Campaign management UI lands in the next commit. For now, this page just
        proves the auth flow is working.
      </p>
      <p className="mt-4">
        <a
          href="/api/auth/logout"
          className="text-sm text-zinc-600 dark:text-zinc-400 underline underline-offset-4"
        >
          Sign out
        </a>
      </p>
    </div>
  );
}
