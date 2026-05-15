package registry

import "time"

// BuildSnapshotForTest exposes the internal buildSnapshot helper for use by
// downstream package tests (selection benchmarks especially) that want a
// fully-formed snapshot without spinning up Postgres. Not for production
// callers — the loader is the supported way to construct snapshots.
func BuildSnapshotForTest(ads []*Ad, endDates []*time.Time, now time.Time) *Snapshot {
	return buildSnapshot(ads, endDates, now)
}
