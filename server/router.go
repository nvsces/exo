package server

import (
	"net"
	"strings"
	"sync"
)

type Router struct {
	mu      sync.RWMutex
	tunnels map[string]*Tunnel
}

func NewRouter() *Router {
	return &Router{
		tunnels: make(map[string]*Tunnel),
	}
}

func (r *Router) Register(subdomain string, t *Tunnel) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tunnels[subdomain]; exists {
		return false
	}
	r.tunnels[subdomain] = t
	return true
}

func (r *Router) Unregister(subdomain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tunnels, subdomain)
}

func (r *Router) Get(subdomain string) *Tunnel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tunnels[subdomain]
}

// ExtractSubdomain extracts subdomain from Host header.
// e.g. "myapp.example.com" with domain "example.com" => "myapp"
func ExtractSubdomain(host, domain string) string {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	suffix := "." + domain
	if strings.HasSuffix(h, suffix) {
		return strings.TrimSuffix(h, suffix)
	}
	return ""
}
