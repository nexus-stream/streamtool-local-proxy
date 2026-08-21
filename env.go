package main

import (
	"os"
	"path/filepath"
	"strings"
)

type environment struct {
	name string
	url  string
}

var environments = []environment{
	{name: "prod", url: "https://project-nexus-stream.web.app/"},
	{name: "staging", url: "https://project-nexus-stream-develop.web.app/"},
}

func envByName(name string) (environment, bool) {
	for _, e := range environments {
		if e.name == name {
			return e, true
		}
	}
	return environment{}, false
}

// otherEnv returns the environment to flip to. Custom targets (from --url)
// have no pair, so they flip to prod.
func otherEnv(name string) environment {
	if name == "prod" {
		e, _ := envByName("staging")
		return e
	}
	e, _ := envByName("prod")
	return e
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "nexus-stream-proxy"), nil
}

// loadSavedEnv returns the persisted environment name, if it names a known
// environment. Any failure (missing config dir, unreadable file, unknown name)
// is treated as "nothing saved".
func loadSavedEnv() (string, bool) {
	dir, err := configDir()
	if err != nil {
		return "", false
	}
	b, err := os.ReadFile(filepath.Join(dir, "environment"))
	if err != nil {
		return "", false
	}
	name := strings.ToLower(strings.TrimSpace(string(b)))
	if _, ok := envByName(name); !ok {
		return "", false
	}
	return name, true
}

func saveEnv(name string) {
	if _, ok := envByName(name); !ok {
		return
	}
	dir, err := configDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "environment"), []byte(name+"\n"), 0o644)
}
