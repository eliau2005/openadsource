package registry

import "github.com/google/uuid"

// WildcardKey is the conventional Bitset key for ads whose targeting array
// for a dimension was NULL ("match anything"). The snapshot indexes them
// here, and Snapshot.matchOrWildcard falls back to it when the request's
// concrete country/device doesn't have its own bitset.
const WildcardKey = "*"

// Ad is the in-memory representation of a single creative. It carries
// everything /vast needs to build a response — no DB joins required at
// request time.
type Ad struct {
	ID         uuid.UUID
	CampaignID uuid.UUID
	Name       string

	PositionType  string // "pre" | "mid" | "post"
	MidRollOffset int32  // 0 when PositionType != "mid"
	Priority      int32

	LandingPageURL  string
	MediaSource     string // "external_url" | "internal_s3"
	MediaURL        string
	MediaMime       string
	MediaWidth      int32
	MediaHeight     int32
	MediaBitrate    int32
	MediaDurationMs int32

	// Targeting; nil = "all".
	Countries []string
	Devices   []string

	// Campaign-level budget cap (0 = unlimited). Enforced live by
	// capping.Enforcer via Redis. Stored here so the snapshot can be
	// queried for it without joining back.
	BudgetTotal int32

	// PacingWeight is the snapshot-time pacing target — selection uses it
	// as the weight in weighted-random tie-breaking inside a priority
	// bucket. Recomputed on every reload.
	PacingWeight float64
}
