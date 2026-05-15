package targeting

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog/log"
)

// GeoResolver answers "which ISO country code is this IP from?" using a
// mmap'd GeoLite2-Country database. The interface lets tests inject a stub.
type GeoResolver interface {
	CountryISO(ip netip.Addr) string
	Close() error
}

// NewGeoResolver opens the MaxMind .mmdb at path. Missing or unreadable
// files are non-fatal — the loader logs a warning and returns a stub
// resolver that maps every IP to "" (unknown country). That keeps dev
// environments running without a MaxMind account.
func NewGeoResolver(path string) (GeoResolver, error) {
	if path == "" {
		log.Warn().Msg("GEOIP_DB_PATH unset; country targeting will treat every request as unknown country")
		return &stubGeoResolver{}, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Str("path", path).Msg("GeoLite2 .mmdb not found; country targeting disabled (stub)")
			return &stubGeoResolver{}, nil
		}
		return nil, err
	}
	reader, err := geoip2.Open(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("GeoLite2 .mmdb failed to open; falling back to stub")
		return &stubGeoResolver{}, nil
	}
	log.Info().Str("path", path).Msg("GeoLite2 .mmdb loaded")
	return &mmdbResolver{reader: reader}, nil
}

type stubGeoResolver struct{}

func (s *stubGeoResolver) CountryISO(_ netip.Addr) string { return "" }
func (s *stubGeoResolver) Close() error                   { return nil }

type mmdbResolver struct {
	reader *geoip2.Reader
}

func (m *mmdbResolver) CountryISO(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	stdIP := net.IP(ip.AsSlice())
	rec, err := m.reader.Country(stdIP)
	if err != nil {
		return ""
	}
	return strings.ToUpper(rec.Country.IsoCode)
}

func (m *mmdbResolver) Close() error { return m.reader.Close() }
