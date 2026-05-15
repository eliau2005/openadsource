"use server";

import { requireSession } from "@/lib/session";
import { probeExternalMediaURL, type ProbeResult } from "@/lib/media";

// Wraps the BYO-URL probe behind a server action so the client-side picker
// (commit P2-4) can validate a URL without leaking the dashboard's pg pool
// or DNS resolver into the browser bundle.
export async function validateExternalMediaAction(url: string): Promise<ProbeResult> {
  await requireSession();
  if (!url || typeof url !== "string") {
    return { ok: false, reason: "Empty URL" };
  }
  return probeExternalMediaURL(url);
}
