package vast

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Version is the VAST spec version emitted on every response.
const Version = "4.2"

// Default MediaFile geometry used when the ad row has null width/height.
// 1280x720 is a safe baseline that virtually every player accepts.
const (
	defaultWidth      = 1280
	defaultHeight     = 720
	defaultDurationHi = "00:00:30"
)

// InlineInput is the shape the delivery handler hands to BuildInline.
// All URL fields should already be fully qualified (scheme + host) — the
// builder does no rewriting.
type InlineInput struct {
	AdID          string
	Title         string
	ImpressionURL string
	MediaURL      string
	MediaMime     string
	MediaWidth    int
	MediaHeight   int
	MediaBitrate  int    // kilobits per second; 0 = omit the bitrate attribute
	MediaDuration string // "HH:MM:SS"; empty = 00:00:30 fallback
	LandingURL    string // optional; when empty no VideoClicks block is emitted
}

// BuildInline serializes an InLine VAST 4.2 response.
func BuildInline(in InlineInput) ([]byte, error) {
	inline := &InLine{
		AdSystem: AdSystem{Version: "1.0", Name: "OpenAdSource"},
		AdTitle:  CDATAString{Value: in.Title},
		Impressions: []Impression{
			{ID: "imp-0", URL: in.ImpressionURL},
		},
		Creatives: Creatives{
			Creatives: []Creative{
				{
					ID:       "creative-" + in.AdID,
					Sequence: 1,
					UniversalAdID: UniversalAdID{
						IDRegistry: "unknown",
						Value:      "unknown",
					},
					Linear: buildLinear(in),
				},
			},
		},
	}

	v := VAST{
		Version: Version,
		Ads: []Ad{
			{ID: in.AdID, InLine: inline},
		},
	}
	return marshal(v)
}

// BuildEmpty serializes a valid no-fill VAST response. Players treat this
// as "no ad available right now" and skip cleanly.
func BuildEmpty() ([]byte, error) {
	return marshal(VAST{Version: Version})
}

func buildLinear(in InlineInput) *Linear {
	width := in.MediaWidth
	if width <= 0 {
		width = defaultWidth
	}
	height := in.MediaHeight
	if height <= 0 {
		height = defaultHeight
	}
	duration := in.MediaDuration
	if duration == "" {
		duration = defaultDurationHi
	}

	linear := &Linear{
		Duration: duration,
		MediaFiles: MediaFiles{
			MediaFiles: []MediaFile{
				{
					ID:       "media-0",
					Delivery: deliveryFor(in.MediaMime),
					Type:     in.MediaMime,
					Width:    width,
					Height:   height,
					Bitrate:  in.MediaBitrate,
					URL:      in.MediaURL,
				},
			},
		},
	}
	if in.LandingURL != "" {
		linear.VideoClicks = &VideoClicks{
			ClickThroughs: []ClickThrough{
				{ID: "click-0", URL: in.LandingURL},
			},
		}
	}
	return linear
}

// deliveryFor maps a MIME type to the VAST "delivery" attribute. Streaming
// manifests (HLS / DASH) use "streaming"; everything else is progressive.
func deliveryFor(mime string) string {
	switch strings.ToLower(mime) {
	case "application/x-mpegurl", "application/vnd.apple.mpegurl":
		return "streaming"
	case "application/dash+xml":
		return "streaming"
	default:
		return "progressive"
	}
}

func marshal(v VAST) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("marshal vast: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("flush vast: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
