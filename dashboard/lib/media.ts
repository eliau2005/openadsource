// BYO-URL ingestion: probe a URL the operator pasted to confirm it's
// reachable, video-shaped, not too large, and not pointing at a private
// network (which would be an SSRF foothold).
import "server-only";

import { lookup } from "node:dns/promises";

import { isProduction } from "@/lib/env";

export const ALLOWED_MIMES = [
  "video/mp4",
  "application/x-mpegurl",
  "application/vnd.apple.mpegurl",
  "application/dash+xml",
] as const;

const MAX_BYTES = 500 * 1024 * 1024;
const PROBE_TIMEOUT_MS = 8_000;

export type ProbeResult =
  | { ok: true; mime: string; bytes: number | null }
  | { ok: false; reason: string };

function isPrivateIPv4(ip: string): boolean {
  const parts = ip.split(".").map((n) => Number(n));
  if (parts.length !== 4 || parts.some((p) => !Number.isFinite(p))) return false;
  const [a, b] = parts;
  if (a === 10) return true;
  if (a === 172 && b >= 16 && b <= 31) return true;
  if (a === 192 && b === 168) return true;
  if (a === 127) return true;
  if (a === 169 && b === 254) return true;
  if (a === 0) return true;
  return false;
}

function isPrivateIPv6(ip: string): boolean {
  const lower = ip.toLowerCase();
  if (lower === "::1" || lower === "::") return true;
  if (lower.startsWith("fc") || lower.startsWith("fd")) return true; // fc00::/7 ULA
  if (lower.startsWith("fe80:")) return true; // link-local
  return false;
}

async function isPrivateHost(hostname: string): Promise<boolean> {
  // Literal IPs short-circuit DNS.
  if (/^\d+\.\d+\.\d+\.\d+$/.test(hostname)) return isPrivateIPv4(hostname);
  if (hostname.includes(":")) return isPrivateIPv6(hostname.replace(/^\[|\]$/g, ""));

  if (hostname === "localhost") return true;
  try {
    const records = await lookup(hostname, { all: true });
    for (const r of records) {
      const ip = r.address;
      if (r.family === 4 && isPrivateIPv4(ip)) return true;
      if (r.family === 6 && isPrivateIPv6(ip)) return true;
    }
  } catch {
    // Refuse rather than fall through — un-resolvable hostnames shouldn't
    // sneak past the SSRF guard.
    return true;
  }
  return false;
}

function normalizeMime(value: string | null): string | null {
  if (!value) return null;
  return value.split(";")[0]?.trim().toLowerCase() ?? null;
}

async function fetchWithTimeout(url: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

export async function probeExternalMediaURL(input: string): Promise<ProbeResult> {
  let parsed: URL;
  try {
    parsed = new URL(input);
  } catch {
    return { ok: false, reason: "Not a valid URL" };
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    return { ok: false, reason: "URL must use http or https" };
  }
  if (isProduction && parsed.protocol !== "https:") {
    return { ok: false, reason: "Production requires https URLs" };
  }
  if (await isPrivateHost(parsed.hostname)) {
    return { ok: false, reason: "URL resolves to a private network address" };
  }

  let resp: Response;
  try {
    resp = await fetchWithTimeout(parsed.toString(), { method: "HEAD", redirect: "follow" });
  } catch {
    return { ok: false, reason: "Request timed out or failed" };
  }
  // Some CDNs reject HEAD; fall back to a 1-byte ranged GET.
  if (resp.status === 405 || resp.status === 501) {
    try {
      resp = await fetchWithTimeout(parsed.toString(), {
        method: "GET",
        redirect: "follow",
        headers: { Range: "bytes=0-0" },
      });
    } catch {
      return { ok: false, reason: "Request timed out or failed" };
    }
  }
  if (!resp.ok && resp.status !== 206) {
    return { ok: false, reason: `Upstream returned HTTP ${resp.status}` };
  }

  const mime = normalizeMime(resp.headers.get("content-type"));
  if (!mime || !(ALLOWED_MIMES as readonly string[]).includes(mime)) {
    return {
      ok: false,
      reason: `Mime ${mime ?? "(unknown)"} not in allowlist: ${ALLOWED_MIMES.join(", ")}`,
    };
  }

  let bytes: number | null = null;
  const cl = resp.headers.get("content-length");
  const cr = resp.headers.get("content-range");
  if (cl) bytes = Number(cl);
  if (!bytes && cr) {
    const total = cr.split("/")[1];
    if (total && total !== "*") bytes = Number(total);
  }
  if (bytes != null && bytes > MAX_BYTES) {
    return {
      ok: false,
      reason: `File is ${bytes} bytes; max is ${MAX_BYTES}`,
    };
  }

  return { ok: true, mime, bytes };
}

// Re-export the s3Configured flag so the picker can check it without
// re-importing env directly from a client-ish module.
export { s3Configured } from "@/lib/env";
