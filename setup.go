package scw

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
)

// Handler represents a middleware instance
type Handler struct {
	Next       httpserver.Handler
	BlockedIPs *BlockedIPs
	Config     Config
}

var privateIPBlocks []*net.IPNet

// Init initializes the plugin
func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}

	caddy.RegisterPlugin("scw", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func setup(c *caddy.Controller) error {
	config, err := parseConfig(c)
	if err != nil {
		return err
	}

	blip, err := NewBlockedIPs(config.RedisURI, config.UpdateInterval)
	if err != nil {
		return c.Err("scw: Can't connect to redis: " + config.RedisURI)
	}
	// Create new middleware
	newMiddleWare := func(next httpserver.Handler) httpserver.Handler {
		return &Handler{
			Next:       next,
			BlockedIPs: blip,
			Config:     config,
		}
	}
	// Add middleware
	cfg := httpserver.GetConfig(c)
	cfg.AddMiddleware(newMiddleWare)

	return nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	h.lookupIP(w, r)
	return h.Next.ServeHTTP(w, r)
}

func (h Handler) lookupIP(w http.ResponseWriter, r *http.Request) {
	replacer := newReplacer(r)
	isBlocked := false

	rip := r.RemoteAddr[0:strings.Index(r.RemoteAddr, ":")]
	if !isPrivateIP(net.ParseIP(rip)) && h.BlockedIPs.IsBlocked(rip, true) {
		isBlocked = true
	}

OUTER:
	for _, xff := range r.Header.Values("X-Forwarded-For") {
		for _, ip := range strings.Split(xff, ",") {
			rip = ip[0:strings.Index(strings.TrimSpace(ip), ":")]

			if !isPrivateIP(net.ParseIP(rip)) && h.BlockedIPs.IsBlocked(rip, true) {
				isBlocked = true
				break OUTER
			}
		}
	}

	if !isBlocked {
		return
	}

	replacer.Set("scw_is_blocked", "true")
	if rr, ok := w.(*httpserver.ResponseRecorder); ok {
		rr.Replacer = replacer
	}
}

func newReplacer(r *http.Request) httpserver.Replacer {
	return httpserver.NewReplacer(r, nil, "")
}
