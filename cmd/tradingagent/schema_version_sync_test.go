package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	postgres "github.com/PatrickFanella/get-rich-quick/internal/repository/postgres"
)

func TestSchemaVersionSync(t *testing.T) {
	entries, err := os.ReadDir(tradingAgentMigrationsDir(t))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	latest := -1
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		version, _, ok := strings.Cut(name, "_")
		if !ok {
			t.Fatalf("migration filename %q missing version separator", name)
		}

		n, err := strconv.Atoi(version)
		if err != nil {
			t.Fatalf("migration filename %q has invalid version %q: %v", name, version, err)
		}
		if n > latest {
			latest = n
		}
	}

	if latest < 0 {
		t.Fatal("no .up.sql migrations found")
	}

	if got, want := latest, postgres.RequiredSchemaVersion; got != want {
		t.Fatalf("required schema version mismatch: latest migration=%d, RequiredSchemaVersion=%d", got, want)
	}
}

func tradingAgentMigrationsDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}
