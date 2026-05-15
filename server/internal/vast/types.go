// Package vast emits VAST 4.2 XML responses for the /vast delivery endpoint.
//
// The struct hierarchy mirrors the IAB VAST 4.2 specification but only models
// the subset we actually emit in Phase 1: a single InLine Ad with a Linear
// creative, one MediaFile, one Impression, and an optional ClickThrough.
// Wrappers, companion ads, non-linear ads, and tracking events beyond the
// Impression are deliberately out of scope until later phases.
//
// The package is namespace-less by convention (matching what real-world VAST
// players consume from Google Ad Manager, Spotx, etc.). The vendored XSD at
// testdata/vast_4.2.xsd is the upstream IAB schema which uses the
// `http://www.iab.com/VAST` namespace; the XSD test relaxes it at runtime so
// our namespace-less output can be validated.
package vast

import "encoding/xml"

// VAST is the root element.
type VAST struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr"`
	Ads     []Ad     `xml:"Ad,omitempty"`
}

// Ad wraps a single ad response. Phase 1 only emits InLine; Wrapper is left
// for later phases (typically used for ad-server-to-ad-server chaining).
type Ad struct {
	ID     string  `xml:"id,attr"`
	InLine *InLine `xml:"InLine,omitempty"`
}

// InLine is the actual ad payload: who, what, where to track, what to play.
type InLine struct {
	AdSystem    AdSystem     `xml:"AdSystem"`
	AdTitle     CDATAString  `xml:"AdTitle"`
	Impressions []Impression `xml:"Impression"`
	Creatives   Creatives    `xml:"Creatives"`
}

// AdSystem identifies the source ad server. Required by VAST 4.x.
type AdSystem struct {
	Version string `xml:"version,attr"`
	Name    string `xml:",chardata"`
}

// Impression is the tracking URL fired when the ad first renders on the page.
type Impression struct {
	ID  string `xml:"id,attr"`
	URL string `xml:",cdata"`
}

// Creatives is the container for one-or-more Creative elements.
type Creatives struct {
	Creatives []Creative `xml:"Creative"`
}

// Creative is a single creative slot. Phase 1 only emits Linear.
type Creative struct {
	ID            string        `xml:"id,attr"`
	Sequence      int           `xml:"sequence,attr,omitempty"`
	UniversalAdID UniversalAdID `xml:"UniversalAdId"`
	Linear        *Linear       `xml:"Linear,omitempty"`
}

// UniversalAdID is required by VAST 4.x. Phase 1 uses a placeholder; a later
// phase can route real ad-registry IDs through here.
type UniversalAdID struct {
	IDRegistry string `xml:"idRegistry,attr"`
	Value      string `xml:",chardata"`
}

// Linear is a standard pre/mid/post-roll video creative.
type Linear struct {
	Duration       string          `xml:"Duration"`
	TrackingEvents *TrackingEvents `xml:"TrackingEvents,omitempty"`
	MediaFiles     MediaFiles      `xml:"MediaFiles"`
	VideoClicks    *VideoClicks    `xml:"VideoClicks,omitempty"`
}

// TrackingEvents wraps a list of <Tracking event="…"> pixels (start,
// firstQuartile, midpoint, thirdQuartile, complete). Optional under
// <Linear>.
type TrackingEvents struct {
	Trackings []Tracking `xml:"Tracking"`
}

// Tracking is a single pixel URL fired when the player reaches the
// associated quartile. event="…" attribute keys it; body is the URL.
type Tracking struct {
	Event string `xml:"event,attr"`
	URL   string `xml:",cdata"`
}

// MediaFiles is the container for one-or-more MediaFile elements (different
// bitrates / mime types of the same creative). Phase 1 emits one.
type MediaFiles struct {
	MediaFiles []MediaFile `xml:"MediaFile"`
}

// MediaFile is a single playable URL with its codec metadata.
type MediaFile struct {
	ID       string `xml:"id,attr"`
	Delivery string `xml:"delivery,attr"`
	Type     string `xml:"type,attr"`
	Width    int    `xml:"width,attr"`
	Height   int    `xml:"height,attr"`
	Bitrate  int    `xml:"bitrate,attr,omitempty"`
	URL      string `xml:",cdata"`
}

// VideoClicks wraps ClickThrough (landing page) plus optional ClickTracking
// pixels. Phase 4 emits one ClickTracking alongside the ClickThrough so the
// player fires a signed pixel back to /track when the viewer clicks.
type VideoClicks struct {
	ClickThroughs  []ClickThrough  `xml:"ClickThrough,omitempty"`
	ClickTrackings []ClickTracking `xml:"ClickTracking,omitempty"`
}

// ClickThrough is the landing-page URL the player navigates to on click.
type ClickThrough struct {
	ID  string `xml:"id,attr"`
	URL string `xml:",cdata"`
}

// ClickTracking is the pixel URL the player fires when the viewer clicks.
type ClickTracking struct {
	ID  string `xml:"id,attr"`
	URL string `xml:",cdata"`
}

// CDATAString is a tiny wrapper used for element bodies that should be
// wrapped in <![CDATA[ ... ]]> rather than character-escaped. Avoids
// duplicating an anonymous struct on every URL-bearing element.
type CDATAString struct {
	Value string `xml:",cdata"`
}
