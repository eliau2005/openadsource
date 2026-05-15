"use client";

import { useActionState } from "react";

import { createAdminAction, type SetupState } from "@/app/(auth)/setup/_actions";

export function SetupForm() {
  const [state, formAction, isPending] = useActionState<SetupState, FormData>(createAdminAction, undefined);

  return (
    <form action={formAction} className="space-y-4">
      <div>
        <h2 className="text-lg font-medium">Create the admin account</h2>
        <p className="text-sm text-zinc-500 mt-1">
          This is the first time the dashboard is being used. The account you create here will
          have full access.
        </p>
      </div>

      <div className="space-y-1">
        <label htmlFor="email" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Email
        </label>
        <input
          id="email"
          name="email"
          type="email"
          autoComplete="username"
          required
          disabled={isPending}
          className="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-400 disabled:opacity-50"
        />
      </div>

      <div className="space-y-1">
        <label htmlFor="password" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Password (min 8 characters)
        </label>
        <input
          id="password"
          name="password"
          type="password"
          autoComplete="new-password"
          minLength={8}
          required
          disabled={isPending}
          className="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-400 disabled:opacity-50"
        />
      </div>

      <div className="space-y-1">
        <label htmlFor="confirm" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Confirm password
        </label>
        <input
          id="confirm"
          name="confirm"
          type="password"
          autoComplete="new-password"
          minLength={8}
          required
          disabled={isPending}
          className="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-400 disabled:opacity-50"
        />
      </div>

      {state?.error ? (
        <div className="text-sm text-red-600 dark:text-red-400">{state.error}</div>
      ) : null}

      <button
        type="submit"
        disabled={isPending}
        className="w-full rounded-md bg-zinc-900 dark:bg-zinc-50 text-white dark:text-zinc-900 px-3 py-2 text-sm font-medium hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50"
      >
        {isPending ? "Creating..." : "Create admin account"}
      </button>
    </form>
  );
}
