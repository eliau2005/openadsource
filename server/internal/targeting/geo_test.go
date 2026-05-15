package targeting

import (
	"net/netip"
	"testing"
)

func TestNewGeoResolver_EmptyPath_ReturnsStub(t *testing.T) {
	g, err := NewGeoResolver("")
	if err != nil {
		t.Fatalf("empty path should not error: %v", err)
	}
	defer g.Close()
	if got := g.CountryISO(netip.MustParseAddr("8.8.8.8")); got != "" {
		t.Errorf("stub should return empty country, got %q", got)
	}
}

func TestNewGeoResolver_MissingFile_ReturnsStub(t *testing.T) {
	g, err := NewGeoResolver("/definitely/does/not/exist/GeoLite2-Country.mmdb")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	defer g.Close()
	if got := g.CountryISO(netip.MustParseAddr("1.1.1.1")); got != "" {
		t.Errorf("stub should return empty country, got %q", got)
	}
}

func TestStubResolver_InvalidIPSafe(t *testing.T) {
	g, _ := NewGeoResolver("")
	defer g.Close()
	// Zero / invalid Addr must not panic.
	if got := g.CountryISO(netip.Addr{}); got != "" {
		t.Errorf("zero Addr: want '', got %q", got)
	}
}
