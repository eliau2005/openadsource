// Drizzle DB client backed by node-postgres. Singleton across HMR reloads so
// we don't leak pg connections in dev. Imported by server components and
// server actions only — never by proxy.ts (which runs on Edge and can't use
// the pg driver).
import "server-only";

import { drizzle, type NodePgDatabase } from "drizzle-orm/node-postgres";
import { Pool } from "pg";

import { env } from "@/lib/env";
import * as schema from "@/lib/db/schema";

const globalForDb = globalThis as unknown as {
  __oasPool?: Pool;
  __oasDb?: NodePgDatabase<typeof schema>;
};

const pool =
  globalForDb.__oasPool ??
  new Pool({
    connectionString: env.DATABASE_URL,
    max: 10,
    idleTimeoutMillis: 30_000,
  });

export const db: NodePgDatabase<typeof schema> =
  globalForDb.__oasDb ?? drizzle(pool, { schema });

if (env.NODE_ENV !== "production") {
  globalForDb.__oasPool = pool;
  globalForDb.__oasDb = db;
}
