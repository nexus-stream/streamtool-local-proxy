package main

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

// transport is shared by every reverse proxy instance. It deliberately mirrors
// the previously working nginx setup: a fresh connection per request
// (DisableKeepAlives, like nginx's default HTTP/1.0 + Connection: close) and
// plain HTTP/1.1 with no HTTP/2 multiplexing. Explicit timeouts bound how long
// a stalled or half-open connection can block a request.
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     false,
	DisableKeepAlives:     true,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	ExpectContinueTimeout: time.Second,
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	// Go 1.26's NewSingleHostReverseProxy no longer rewrites the outgoing Host
	// header, only req.URL. Firebase Hosting routes on Host, so forward the
	// upstream host explicitly.
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
	}
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
