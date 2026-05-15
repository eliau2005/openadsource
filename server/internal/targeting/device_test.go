package targeting

import "testing"

func TestClassifyUA(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{"empty", "", ""},
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15", "mobile"},
		{"android phone", "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36", "mobile"},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15", "tablet"},
		{"android tablet", "Mozilla/5.0 (Linux; Android 13; Tablet) AppleWebKit/537.36", "tablet"},
		{"chrome desktop", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/123", "desktop"},
		{"roku", "Roku/DVP-12.0 (12.0.0.4218-65)", "ctv"},
		{"apple tv", "AppleTV/tvOS_15.0", "ctv"},
		{"smarttv", "Mozilla/5.0 (SMART-TV; Linux; Tizen 6.0)", "ctv"},
		{"playstation", "Mozilla/5.0 (PlayStation; PlayStation 5/2.0) AppleWebKit/605.1.15", "ctv"},
		{"junk", "totally not a user agent", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyUA(tc.ua); got != tc.want {
				t.Errorf("ClassifyUA(%q) = %q, want %q", tc.ua, got, tc.want)
			}
		})
	}
}

func TestNormaliseDevice(t *testing.T) {
	if NormaliseDevice("mobile") != "mobile" {
		t.Error("expected mobile passthrough")
	}
	if NormaliseDevice("phone") != "" {
		t.Error("expected unknown -> empty")
	}
}
