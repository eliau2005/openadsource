// Package targeting classifies inbound /vast requests by country (GeoIP) and
// device (user-agent) so the decision engine can intersect them against the
// targeting allowlists stored on each campaign.
package targeting

import "regexp"

// ClassifyUA reduces a free-form User-Agent string to one of the device
// categories the snapshot indexes on. Returns "" for unknown UAs.
//
// The order is significant — CTV markers come before "Android" because most
// smart-TV UAs include both, and the more specific class wins.
func ClassifyUA(ua string) string {
	if ua == "" {
		return ""
	}
	for _, c := range classifiers {
		if c.pattern.MatchString(ua) {
			return c.device
		}
	}
	return ""
}

type classifier struct {
	device  string
	pattern *regexp.Regexp
}

var classifiers = []classifier{
	{"ctv", regexp.MustCompile(`(?i)\b(?:SmartTV|HbbTV|AppleTV|Roku|GoogleTV|BRAVIA|Tizen|webOS|Web0S|CrKey|PlayStation|Xbox)\b`)},
	{"tablet", regexp.MustCompile(`(?i)\b(?:iPad|Tablet|Kindle|Silk)\b`)},
	{"mobile", regexp.MustCompile(`(?i)\b(?:Mobile|iPhone|iPod|Android|Opera Mini|IEMobile)\b`)},
	{"desktop", regexp.MustCompile(`(?i)\b(?:Mozilla|Chrome|Safari|Firefox|Edge|Opera)\b`)},
}

// NormaliseDevice canonicalises an explicit ?device= query override into the
// snapshot's vocabulary. Unrecognised inputs become "".
func NormaliseDevice(s string) string {
	switch s {
	case "mobile", "tablet", "desktop", "ctv":
		return s
	default:
		return ""
	}
}
