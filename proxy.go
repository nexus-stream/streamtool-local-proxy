package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

func newProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	// Go 1.26's NewSingleHostReverseProxy no longer rewrites the outgoing Host
	// header, only req.URL. Firebase Hosting routes on Host, so forward the
	// upstream host explicitly.
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
	}
	// Flush promptly so SSE and other streaming responses are not buffered.
	proxy.FlushInterval = 100 * time.Millisecond
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}
	return proxy
}

// switcher serves the reverse proxy for the currently selected target and
// swaps it atomically when the user changes environment, so in-flight requests
// keep their old target while new ones use the new one.
type switcher struct {
	current atomic.Pointer[httputil.ReverseProxy]
}

func newSwitcher(target *url.URL) *switcher {
	s := &switcher{}
	s.current.Store(newProxy(target))
	return s
}

func (s *switcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.current.Load().ServeHTTP(w, r)
}

func (s *switcher) set(target *url.URL) {
	s.current.Store(newProxy(target))
}
