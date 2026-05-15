// Single source of truth for the dashboard's runtime configuration.
// Validates at module load so a missing-or-malformed env produces a clean
// error message instead of an obscure crash deep inside a route handler.
import { z } from "zod";

const schema = z.object({
  DATABASE_URL: z.string().min(1, "DATABASE_URL is required"),
  JWT_SECRET: z.string().min(8, "JWT_SECRET must be at least 8 characters"),
  PUBLIC_BASE_URL: z.string().url().default("http://localhost:8080"),

  // S3-compatible storage. Optional — uploads are gated on s3Configured.
  S3_ENDPOINT: z.string().url().optional(),
  S3_PUBLIC_ENDPOINT: z.string().url().optional(),
  S3_REGION: z.string().default("us-east-1"),
  S3_BUCKET: z.string().optional(),
  S3_ACCESS_KEY_ID: z.string().optional(),
  S3_SECRET_ACCESS_KEY: z.string().optional(),
  S3_FORCE_PATH_STYLE: z
    .string()
    .optional()
    .transform((v) => v === undefined ? true : v === "true" || v === "1"),
  S3_PUBLIC_BASE_URL: z.string().url().optional(),

  NODE_ENV: z.enum(["development", "production", "test"]).default("development"),
});

export const env = schema.parse(process.env);

export const s3Configured =
  !!env.S3_ENDPOINT &&
  !!env.S3_BUCKET &&
  !!env.S3_ACCESS_KEY_ID &&
  !!env.S3_SECRET_ACCESS_KEY;

export const isProduction = env.NODE_ENV === "production";
