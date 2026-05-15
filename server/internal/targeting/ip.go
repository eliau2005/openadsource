package targeting

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// IPResolver picks the canonical client IP for a request. It only honours
// X-Forwarded-For when the immediate TCP peer is in the trusted-proxy list,
// to keep public-internet clients from spoofing arbitrary origins.
type IPResolver struct {
	trusted []netip.Prefix
}

// NewIPResolver parses a comma-separated CIDR list (e.g.
// "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"). An empty list means "trust no
// peer" — the TCP source is always used.
func NewIPResolver(cidrList string) (*IPResolver, error) {
	r := &IPResolver{}
	for _, raw := range strings.Split(cidrList, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy CIDR %q: %w", raw, err)
		}
		r.trusted = append(r.trusted, p)
	}
	return r, nil
}

// Resolve returns the client IP. When the TCP peer is trusted and the
// request carries an X-Forwarded-For header, the leftmost address in the XFF
// chain that is NOT itself in the trusted list is returned. Otherwise the
// TCP peer is returned. An invalid input returns the zero netip.Addr.
func (r *IPResolver) Resolve(req *http.Request) netip.Addr {
	peer := peerAddr(req.RemoteAddr)
	if !peer.IsValid() {
		return peer
	}
	if !r.isTrusted(peer) {
		return peer
	}
	xff := req.Header.Get("X-Forwarded-For")
	if xff == "" {
		return peer
	}
	for _, part := range strings.Split(xff, ",") {
		ip, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil || !ip.IsValid() {
			continue
		}
		if r.isTrusted(ip) {
			continue
		}
		return ip
	}
	return peer
}

func (r *IPResolver) isTrusted(ip netip.Addr) bool {
	for _, p := range r.trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

func peerAddr(remote string) netip.Addr {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return ip
}
