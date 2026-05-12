// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package builtin

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// writeExecutable drops a tiny POSIX shell script in dir under
// name and chmod +x. Tests assert that LoadCommandPacks picks it
// up. Using sh -c keeps the test portable across CI runners.
func writeExecutable(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCommandPacks_EmptyDirReturnsNil(t *testing.T) {
	dir := t.TempDir()
	got := LoadCommandPacks(context.Background(), nil, dir)
	if got != nil && len(got) != 0 {
		t.Errorf("expected nil/empty, got %v", got)
	}
}

func TestLoadCommandPacks_NonExistentDirReturnsNil(t *testing.T) {
	got := LoadCommandPacks(context.Background(), nil, "/nonexistent-12345")
	if got != nil && len(got) != 0 {
		t.Errorf("expected nil for missing dir, got %v", got)
	}
}

func TestLoadCommandPacks_EmptyStringReturnsNil(t *testing.T) {
	got := LoadCommandPacks(context.Background(), nil, "")
	if got != nil {
		t.Errorf("expected nil for empty path, got %v", got)
	}
}

func TestLoadCommandPacks_FindsExecutables(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "echo", `cat`)
	writeExecutable(t, dir, "upper.sh", `tr '[:lower:]' '[:upper:]'`)
	// Non-executable file: should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	got := LoadCommandPacks(context.Background(), logger, dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 packs, got %d", len(got))
	}
	names := make(map[string]bool)
	for _, p := range got {
		names[p.Name] = true
	}
	if !names["cmd.echo"] {
		t.Errorf("missing cmd.echo, got names %v", names)
	}
	if !names["cmd.upper"] {
		t.Errorf("missing cmd.upper, got names %v", names)
	}
}

func TestLoadCommandPacks_EchoBinaryRoundTrips(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "echo", `cat`)
	got := LoadCommandPacks(context.Background(), nil, dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 pack, got %d", len(got))
	}
	pack := got[0]
	if pack.Name != "cmd.echo" {
		t.Errorf("pack name = %q, want cmd.echo", pack.Name)
	}
	// Don't actually run the handler here — the engine wires up
	// ec.Input + schema validation, which we're not duplicating in
	// this unit test. The command_pack_test.go in internal/packs
	// covers the handler behavior end-to-end.
}

func TestSanitizePackBasename(t *testing.T) {
	tests := map[string]string{
		"echo":         "echo",
		"Upper":        "upper",
		"my-pack":      "my-pack",
		"my_pack":      "my-pack",
		"Upper Case!":  "uppercase",
		"!!!":          "",
		"  spaced  ":   "spaced",
		"123pack":      "123pack",
		"-leading":     "leading",
		"trailing-":    "trailing",
	}
	for in, want := range tests {
		got := sanitizePackBasename(in)
		if got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
