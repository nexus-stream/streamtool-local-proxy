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
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultPort = 48151

func main() {
	upstream := flag.String("url", "", "upstream URL to proxy (overrides environment selection)")
	envFlag := flag.String("env", "", "starting environment: prod or staging")
	port := flag.Int("port", defaultPort, "local port to listen on")
	host := flag.String("host", "127.0.0.1", "interface to bind (advanced)")
	flag.Parse()

	current := resolveEnvironment(*upstream, *envFlag)

	target, err := url.Parse(current.url)
	if err != nil {
		fatal("invalid upstream %q: %v", current.url, err)
	}
	if target.Scheme == "" || target.Host == "" {
		fatal("upstream must be an absolute URL, e.g. https://example.com/")
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fatal("could not start on %s: %v", addr, err)
	}

	sw := newSwitcher(target)
	srv := &http.Server{Handler: sw}

	p := tea.NewProgram(&ui{sw: sw, env: current, port: *port}, tea.WithAltScreen())

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			p.Quit()
		}
	}()

	// SIGTERM (and SIGINT where the terminal doesn't deliver it as a key)
	// stop the program so the server can shut down cleanly.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fatal("ui error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// resolveEnvironment picks the starting target: an explicit --url wins, then
// --env, then the last saved environment, then prod.
func resolveEnvironment(upstream, envFlag string) environment {
	if upstream != "" {
		return environment{name: "custom", url: upstream}
	}
	if envFlag != "" {
		name := strings.ToLower(envFlag)
		if e, ok := envByName(name); ok {
			return e
		}
		fatal("invalid --env %q: choose prod or staging", envFlag)
	}
	if name, ok := loadSavedEnv(); ok {
		e, _ := envByName(name)
		return e
	}
	e, _ := envByName("prod")
	return e
}

// ui is the persistent full-screen display. Bubble Tea re-renders it on every
// update, so toggling replaces the banner instead of appending to it.
type ui struct {
	sw   *switcher
	env  environment
	port int
}

func (m *ui) Init() tea.Cmd { return nil }

func (m *ui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+e":
			e := otherEnv(m.env.name)
			target, err := url.Parse(e.url)
			if err != nil {
				return m, nil
			}
			m.sw.set(target)
			m.env = e
			saveEnv(m.env.name)
			return m, nil
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *ui) View() string {
	return fmt.Sprintf(
		"%s\n\nCtrl+E switches environment\n\nhttp://localhost:%d/\n\nKeep this window open while streaming.\n",
		renderWord(labelFor(m.env)), m.port,
	)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nerror: "+format+"\n", args...)
	fmt.Fprintln(os.Stderr, "Press Enter to close.")
	_, _ = fmt.Scanln()
	os.Exit(1)
}
