package tracking

// Event names tracked by the system. The strings are baked into the URL
// query param (?event=…) and the Redis counter key
// (ad:{ad_id}:event:{event}:{date}), so changing them is a breaking change.
const (
	EventImpression    = "impression"
	EventClick         = "click"
	EventStart         = "start"
	EventFirstQuartile = "firstQuartile"
	EventMidpoint      = "midpoint"
	EventThirdQuartile = "thirdQuartile"
	EventComplete      = "complete"
)

// TrackedEvents is the allowlist /track validates against and the worker
// drains. Anything off this list is silently dropped on the inbound side.
var TrackedEvents = []string{
	EventImpression,
	EventClick,
	EventStart,
	EventFirstQuartile,
	EventMidpoint,
	EventThirdQuartile,
	EventComplete,
}

// QuartileEventsInOrder is the deterministic order the VAST builder uses
// when emitting <TrackingEvents><Tracking …/></TrackingEvents>. Players
// don't care about order but stable output makes the golden-file diff
// readable.
var QuartileEventsInOrder = []string{
	EventStart,
	EventFirstQuartile,
	EventMidpoint,
	EventThirdQuartile,
	EventComplete,
}

// IsTracked is a constant-time-ish lookup; with seven events linear scan
// beats a map allocation.
func IsTracked(event string) bool {
	for _, e := range TrackedEvents {
		if e == event {
			return true
		}
	}
	return false
}
