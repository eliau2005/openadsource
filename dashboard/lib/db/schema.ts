// Drizzle schema mirroring the Postgres tables created by golang-migrate
// (server/migrations/000{1,2,3}). This file does NOT own migrations — it is
// read/write only. When the SQL schema changes, update both this file and
// the relevant .up.sql in server/migrations/.
import {
  pgTable,
  uuid,
  text,
  integer,
  timestamp,
  index,
  uniqueIndex,
} from "drizzle-orm/pg-core";
import { sql, type InferSelectModel, type InferInsertModel } from "drizzle-orm";

export const users = pgTable(
  "users",
  {
    id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
    email: text("email").notNull(),
    passwordHash: text("password_hash").notNull(),
    role: text("role").notNull().default("admin"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => ({
    emailUnique: uniqueIndex("idx_users_email_unique").on(sql`LOWER(${t.email})`),
  }),
);

export const advertisers = pgTable("advertisers", {
  id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
  name: text("name").notNull(),
  status: text("status").notNull().default("active"),
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
});

export const campaigns = pgTable(
  "campaigns",
  {
    id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
    advertiserId: uuid("advertiser_id")
      .notNull()
      .references(() => advertisers.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    startDate: timestamp("start_date", { withTimezone: true }),
    endDate: timestamp("end_date", { withTimezone: true }),
    totalBudgetImpressions: integer("total_budget_impressions"),
    status: text("status").notNull().default("active"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => ({
    advertiserIdx: index("idx_campaigns_advertiser_id").on(t.advertiserId),
    statusIdx: index("idx_campaigns_status").on(t.status),
  }),
);

export const ads = pgTable(
  "ads",
  {
    id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
    campaignId: uuid("campaign_id")
      .notNull()
      .references(() => campaigns.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    status: text("status").notNull().default("active"),
    positionType: text("position_type").notNull().default("pre"),
    midRollOffset: integer("mid_roll_offset"),
    priority: integer("priority").notNull().default(1),
    landingPageUrl: text("landing_page_url"),
    mediaSource: text("media_source").notNull().default("external_url"),
    mediaUrl: text("media_url").notNull(),
    mediaMime: text("media_mime").notNull(),
    mediaDurationMs: integer("media_duration_ms"),
    mediaWidth: integer("media_width"),
    mediaHeight: integer("media_height"),
    mediaBitrateKbps: integer("media_bitrate_kbps"),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => ({
    campaignIdx: index("idx_ads_campaign_id").on(t.campaignId),
    statusPosIdx: index("idx_ads_status_position").on(t.status, t.positionType),
  }),
);

export const capRules = pgTable(
  "cap_rules",
  {
    id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
    adId: uuid("ad_id")
      .notNull()
      .references(() => ads.id, { onDelete: "cascade" }),
    maxImpressions: integer("max_impressions").notNull(),
    timeWindowSeconds: integer("time_window_seconds").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => ({
    adIdx: index("idx_cap_rules_ad_id").on(t.adId),
  }),
);

export type User = InferSelectModel<typeof users>;
export type NewUser = InferInsertModel<typeof users>;
export type Advertiser = InferSelectModel<typeof advertisers>;
export type NewAdvertiser = InferInsertModel<typeof advertisers>;
export type Campaign = InferSelectModel<typeof campaigns>;
export type NewCampaign = InferInsertModel<typeof campaigns>;
export type Ad = InferSelectModel<typeof ads>;
export type NewAd = InferInsertModel<typeof ads>;
