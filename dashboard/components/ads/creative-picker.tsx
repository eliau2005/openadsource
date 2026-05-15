"use client";

import { useState, useRef, useTransition } from "react";
import { toast } from "sonner";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { validateExternalMediaAction } from "@/app/(app)/ads/creative-actions";

export type CreativeValue = {
  mediaSource: "external_url" | "internal_s3";
  mediaUrl: string;
  mediaMime: string;
  publicUrl?: string;
};

const ALLOWED_MIME_HEADERS = [
  "video/mp4",
  "application/x-mpegurl",
  "application/vnd.apple.mpegurl",
  "application/dash+xml",
];

function previewUrl(v: CreativeValue | null): string | null {
  if (!v) return null;
  if (v.mediaSource === "external_url") return v.mediaUrl;
  return v.publicUrl ?? null;
}

export function CreativePicker({
  value,
  onChange,
  s3Configured,
  defaultPreviewPublicUrl,
}: {
  value: CreativeValue | null;
  onChange: (v: CreativeValue | null) => void;
  s3Configured: boolean;
  defaultPreviewPublicUrl?: string;
}) {
  const [activeTab, setActiveTab] = useState<"url" | "upload">(
    value?.mediaSource === "internal_s3" ? "upload" : "url",
  );

  return (
    <div className="space-y-3">
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as "url" | "upload")}>
        <TabsList>
          <TabsTrigger value="url">Use existing URL</TabsTrigger>
          <TabsTrigger value="upload" disabled={!s3Configured} title={s3Configured ? undefined : "S3 is not configured on this deployment"}>
            Upload file{s3Configured ? "" : " (S3 not configured)"}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="url" className="pt-3">
          <UrlTab
            initialValue={value?.mediaSource === "external_url" ? value : null}
            onCommit={onChange}
          />
        </TabsContent>

        <TabsContent value="upload" className="pt-3">
          <UploadTab
            initialValue={value?.mediaSource === "internal_s3" ? value : null}
            onCommit={onChange}
            disabled={!s3Configured}
          />
        </TabsContent>
      </Tabs>

      <PreviewArea
        url={previewUrl(value) ?? defaultPreviewPublicUrl ?? null}
        mime={value?.mediaMime ?? "video/mp4"}
      />
    </div>
  );
}

function UrlTab({
  initialValue,
  onCommit,
}: {
  initialValue: CreativeValue | null;
  onCommit: (v: CreativeValue | null) => void;
}) {
  const [url, setUrl] = useState(initialValue?.mediaUrl ?? "");
  const [pending, startTransition] = useTransition();
  const [result, setResult] = useState<
    | { kind: "ok"; mime: string }
    | { kind: "err"; reason: string }
    | null
  >(initialValue ? { kind: "ok", mime: initialValue.mediaMime } : null);

  function validate() {
    const target = url.trim();
    if (!target) {
      setResult({ kind: "err", reason: "Enter a URL first" });
      return;
    }
    startTransition(async () => {
      const r = await validateExternalMediaAction(target);
      if (r.ok) {
        setResult({ kind: "ok", mime: r.mime });
        onCommit({ mediaSource: "external_url", mediaUrl: target, mediaMime: r.mime });
        toast.success("URL validated", { description: `${r.mime}` });
      } else {
        setResult({ kind: "err", reason: r.reason });
        onCommit(null);
        toast.error("URL rejected", { description: r.reason });
      }
    });
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="byo-url">Public media URL</Label>
      <div className="flex gap-2">
        <Input
          id="byo-url"
          type="url"
          placeholder="https://cdn.example.com/clip.mp4"
          value={url}
          onChange={(e) => {
            setUrl(e.target.value);
            setResult(null);
            onCommit(null);
          }}
          disabled={pending}
        />
        <Button type="button" variant="secondary" onClick={validate} disabled={pending}>
          {pending ? "Validating..." : "Validate"}
        </Button>
      </div>
      {result && (
        <p
          className={
            result.kind === "ok"
              ? "text-sm text-green-700 dark:text-green-400"
              : "text-sm text-red-600 dark:text-red-400"
          }
        >
          {result.kind === "ok"
            ? `OK — mime ${result.mime}`
            : `Failed: ${result.reason}`}
        </p>
      )}
      <p className="text-xs text-zinc-500">
        Must be a public CDN URL (HTTPS in production). Allowed mimes:{" "}
        {ALLOWED_MIME_HEADERS.join(", ")}.
      </p>
    </div>
  );
}

function UploadTab({
  initialValue,
  onCommit,
  disabled,
}: {
  initialValue: CreativeValue | null;
  onCommit: (v: CreativeValue | null) => void;
  disabled: boolean;
}) {
  const [progress, setProgress] = useState<number | null>(null);
  const [uploaded, setUploaded] = useState<CreativeValue | null>(initialValue);
  const [error, setError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  async function handleFile(file: File) {
    setError(null);
    setProgress(0);
    try {
      const presignResp = await fetch("/api/upload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          filename: file.name,
          contentType: file.type || "video/mp4",
        }),
      });
      if (!presignResp.ok) {
        const j = await presignResp.json().catch(() => ({}));
        throw new Error(j.error ?? `HTTP ${presignResp.status}`);
      }
      const { url, key, publicUrl } = await presignResp.json();

      // XHR so we can show real upload progress (fetch doesn't expose it).
      await new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("PUT", url);
        xhr.setRequestHeader("Content-Type", file.type || "video/mp4");
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) setProgress(Math.round((e.loaded / e.total) * 100));
        };
        xhr.onload = () =>
          xhr.status >= 200 && xhr.status < 300
            ? resolve()
            : reject(new Error(`PUT ${xhr.status}: ${xhr.responseText.slice(0, 200)}`));
        xhr.onerror = () => reject(new Error("Network error during upload"));
        xhr.send(file);
      });

      setProgress(100);
      const next: CreativeValue = {
        mediaSource: "internal_s3",
        mediaUrl: key,
        mediaMime: file.type || "video/mp4",
        publicUrl,
      };
      setUploaded(next);
      onCommit(next);
      toast.success("Upload complete", { description: file.name });
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      setProgress(null);
      onCommit(null);
      toast.error("Upload failed", { description: msg });
    }
  }

  if (disabled) {
    return (
      <p className="text-sm text-zinc-500">
        S3 is not configured on this deployment. Configure S3_ENDPOINT and friends in your env to
        enable uploads.
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="upload-file">Pick an MP4 (or HLS / DASH) file</Label>
      <Input
        id="upload-file"
        type="file"
        accept="video/mp4,application/x-mpegURL,application/vnd.apple.mpegurl,application/dash+xml"
        disabled={progress !== null && progress < 100}
        ref={fileRef}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) void handleFile(file);
        }}
      />
      {progress !== null && (
        <div className="space-y-1">
          <div className="h-2 rounded bg-zinc-200 dark:bg-zinc-800 overflow-hidden">
            <div
              className="h-full bg-zinc-900 dark:bg-zinc-50 transition-[width]"
              style={{ width: `${progress}%` }}
            />
          </div>
          <p className="text-xs text-zinc-500">{progress}%</p>
        </div>
      )}
      {uploaded && progress === 100 && (
        <p className="text-sm text-green-700 dark:text-green-400">
          Uploaded as <code className="font-mono text-xs">{uploaded.mediaUrl}</code>
        </p>
      )}
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
    </div>
  );
}

function PreviewArea({ url, mime }: { url: string | null; mime: string }) {
  if (!url) return null;
  const isStream = /mpegurl|dash/i.test(mime);
  return (
    <div className="rounded-md border border-zinc-200 dark:border-zinc-800 p-3 space-y-2 bg-zinc-50 dark:bg-zinc-900">
      <p className="text-xs text-zinc-500">Preview</p>
      {isStream ? (
        <p className="text-xs text-zinc-500">
          HLS/DASH preview requires an external player. URL captured:
          <br />
          <code className="font-mono text-xs break-all">{url}</code>
        </p>
      ) : (
        <video src={url} controls className="w-full max-h-80 bg-black" preload="metadata" />
      )}
    </div>
  );
}
