package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
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
