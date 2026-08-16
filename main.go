package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	defaultUpstream = "https://project-nexus-stream.web.app/"
	defaultPort     = 48151
)

func main() {
	upstream := flag.String("url", defaultUpstream, "upstream URL to proxy")
	port := flag.Int("port", defaultPort, "local port to listen on")
	host := flag.String("host", "127.0.0.1", "interface to bind (advanced)")
	flag.Parse()

	target, err := url.Parse(*upstream)
	if err != nil {
		fatal("invalid --url %q: %v", *upstream, err)
	}
	if target.Scheme == "" || target.Host == "" {
		fatal("--url must be an absolute URL, e.g. https://example.com/")
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fatal("could not start on %s: %v", addr, err)
	}

	fmt.Printf("\nNexus Stream proxy running\n")
	// fmt.Printf("  Upstream: %s\n", *upstream)
	fmt.Printf("  URL:    http://localhost:%d/\n\n", *port)
	// fmt.Printf("Copy the local URL into your tool.\n")
	fmt.Printf("Leave this window open while streaming.\n\n")

	srv := &http.Server{Handler: newProxy(target)}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		fmt.Println()
		fmt.Println("stopping…")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fatal("server error: %v", err)
	}
	fmt.Println("server stopped")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nerror: "+format+"\n", args...)
	fmt.Fprintln(os.Stderr, "Press Enter to close.")
	_, _ = fmt.Scanln()
	os.Exit(1)
}
