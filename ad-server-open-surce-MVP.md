# OpenAdSource - Self-Hosted Video Ad Server MVP

An open-source system for managing and serving video ads (Ad Server) supporting the VAST standard, ad scheduling (Pre/Mid/Post Roll), and frequency capping. The system is designed for high throughput and is easy to self-host.

## 🚀 The Tech Stack

The system is based on separating the delivery engine from the management interface to ensure maximum speed:

* **Ad Delivery Engine:** Go (Golang) - for optimal performance, low memory footprint, and high concurrency handling.
* **Management Dashboard:** Next.js (TypeScript) + Tailwind CSS + Shadcn UI.
* **Database:** PostgreSQL (for metadata and campaign management).
* **Cache & Counters:** Redis (for Frequency Capping and real-time impression aggregation).
* **Storage:** S3 Compatible (AWS S3, MinIO, or Bunny.net) for storing the creatives.
* **Infrastructure:** Docker & Docker Compose for easy deployment in any environment.

---

## 🛠 System Architecture

1.  **Delivery API (`/vast`):** Returns XML in the VAST 4.x standard based on targeting and scheduling parameters.
2.  **Tracking API (`/track`):** Ingests events from the player (Impression, Click, Quartiles) and writes them to Redis.
3.  **Cron / Worker:** Processes data from Redis to PostgreSQL for historical reports[cite: 1].
4.  **Admin UI:** Interface for managing advertisers, campaigns, creatives, and viewing statistics[cite: 1].

---

## 📋 MVP Features (Phase by Phase)

### Phase 1: Infrastructure & VAST Generator (The Core)
* Set up the Docker environment (Go, Postgres, Redis)[cite: 1].
* Build the VAST XML generation module in Go[cite: 1].
* Create a basic Endpoint that accepts an `ad_id` and returns valid XML[cite: 1].
* Support for Linear Ads (video) with MediaFiles and ClickThrough tracking[cite: 1].

### Phase 2: Campaign Management (Management UI)
* Create the Schema in PostgreSQL (Advertisers, Campaigns, Ads)[cite: 1].
* Build a Next.js dashboard for adding campaigns[cite: 1]:
    * Upload video files to Storage[cite: 1].
    * Set budget (maximum impressions)[cite: 1].
    * Select ad position: Pre-roll, Mid-roll (at which specific second), Post-roll[cite: 1].

### Phase 3: Decision Engine
* Implement the Ad Selection Logic[cite: 1]:
    * Check campaign validity (dates and budget)[cite: 1].
    * Basic targeting (country via Geo-IP, device type)[cite: 1].
    * Support for a `?pos=mid` parameter in the URL to return a matching ad[cite: 1].

### Phase 4: Tracking & Frequency Capping
* Embed Tracking Pixels inside the VAST XML[cite: 1].
* Implement a Redis-based Frequency Capping mechanism[cite: 1]:
    * User identification (User ID / Fingerprint)[cite: 1].
    * Block ad delivery if the user exceeds the quota (e.g., 3 times a day)[cite: 1].
* Aggregate Impressions and Clicks in Redis and sync them back to the DB[cite: 1].

---

## 🗄 Database Schema

```sql
-- Campaigns Table
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    advertiser_id UUID,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    total_budget_impressions INTEGER,
    status TEXT DEFAULT 'active' -- active, paused, completed
);

-- Ads (Creatives) Table
CREATE TABLE ads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id),
    video_url TEXT NOT NULL,
    landing_page_url TEXT,
    position_type TEXT, -- pre-roll, mid-roll, post-roll
    mid_roll_offset INTEGER, -- seconds into video
    priority INTEGER DEFAULT 1
);

-- Frequency Capping Rules
CREATE TABLE cap_rules (
    ad_id UUID REFERENCES ads(id),
    max_impressions INTEGER,
    time_window_seconds INTEGER -- e.g. 86400 for 24h
);