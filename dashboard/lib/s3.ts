// AWS SDK v2 client + presign helper. Constructed lazily; nil when S3 is not
// configured. The endpoint defaults to env.S3_PUBLIC_ENDPOINT so signed URLs
// route via the browser-reachable host (the dashboard never PUTs anything
// itself — the browser does).
import "server-only";

import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";

import { env, s3Configured } from "@/lib/env";

let cached: { client: S3Client; bucket: string } | null | undefined;

function buildClient(): { client: S3Client; bucket: string } | null {
  if (!s3Configured) return null;
  const endpoint = env.S3_PUBLIC_ENDPOINT || env.S3_ENDPOINT;
  if (!endpoint) return null;

  const client = new S3Client({
    region: env.S3_REGION,
    endpoint,
    forcePathStyle: env.S3_FORCE_PATH_STYLE,
    credentials: {
      accessKeyId: env.S3_ACCESS_KEY_ID!,
      secretAccessKey: env.S3_SECRET_ACCESS_KEY!,
    },
  });
  return { client, bucket: env.S3_BUCKET! };
}

export function getS3Client(): { client: S3Client; bucket: string } | null {
  if (cached === undefined) {
    cached = buildClient();
  }
  return cached;
}

export async function presignPut(
  key: string,
  contentType: string,
  ttlSeconds: number,
): Promise<{ url: string; key: string } | null> {
  const s = getS3Client();
  if (!s) return null;
  const command = new PutObjectCommand({
    Bucket: s.bucket,
    Key: key,
    ContentType: contentType,
  });
  const url = await getSignedUrl(s.client, command, { expiresIn: ttlSeconds });
  return { url, key };
}

export function publicUrlFor(key: string): string | null {
  if (!env.S3_PUBLIC_BASE_URL) return null;
  return `${env.S3_PUBLIC_BASE_URL.replace(/\/$/, "")}/${key.replace(/^\//, "")}`;
}
