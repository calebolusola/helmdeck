// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package builtin

// command_pack_example.go (T811 MVP) — operator-supplied subprocess
// packs.
//
// LoadCommandPacks scans a directory for executable files and
// registers each as a pack named `cmd.<basename>`. Schemas default
// to BasicSchema{} (passthrough — accepts any JSON, returns any
// JSON) so operators can drop a binary in and immediately call it
// from the agent or the Management UI's Test Runner panel (#172).
//
// Typed schemas via a manifest format (helmdeck-pack.yaml or
// similar) ship in v0.13.0. Until then, schema enforcement is the
// subprocess's responsibility — invalid input surfaces as exit ≠0
// + stderr.
//
// Security note: subprocess egress is the HOST environment's
// responsibility today. helmdeck's EgressGuard intercepts HTTP
// calls inside Go pack handlers but not exec() invocations to
// arbitrary binaries. Operators wanting subprocess egress
// confinement should run the control-plane inside a network
// namespace with an outbound allowlist (or wait for the
// T811-followup: subprocess egress sandbox issue).

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tosin2013/helmdeck/internal/packs"
)

// LoadCommandPacks scans dir for executable files and returns one
// *packs.Pack per binary found. Pack name is `cmd.<basename>`
// (extension stripped), version is "v1", description points
// operators at the binary path so they know what they registered.
//
// Errors on individual files are logged but don't abort scanning —
// one broken binary should not block the others from registering.
//
// Returns an empty slice when dir is empty or doesn't exist; that's
// the expected case for operators who haven't enabled the feature.
func LoadCommandPacks(ctx context.Context, logger *slog.Logger, dir string) []*packs.Pack {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		logger.Warn("command packs dir scan failed", "dir", dir, "err", err)
		return nil
	}

	out := make([]*packs.Pack, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			logger.Warn("command pack stat failed", "name", e.Name(), "err", err)
			continue
		}
		// Skip non-executable files (matches the convention of
		// dropping a binary in to enable a pack; config-only files
		// like manifests will live alongside in a future ship).
		if info.Mode()&0o111 == 0 {
			continue
		}

		path := filepath.Join(dir, e.Name())
		// pack name = cmd.<basename without extension>. Trim
		// common script extensions so operators dropping
		// upper.sh / upper.py both get "cmd.upper".
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		base = sanitizePackBasename(base)
		if base == "" {
			logger.Warn("command pack skipped (unusable basename)", "name", e.Name())
			continue
		}
		packName := "cmd." + base

		pack := packs.NewCommandPack(
			packName,
			"v1",
			fmt.Sprintf("Operator-supplied command pack backed by %s (stdin JSON → stdout JSON). v0.13.0 will add manifest-declared typed schemas; today the input/output are passthrough.", path),
			packs.BasicSchema{}, // passthrough — any JSON in
			packs.BasicSchema{}, // passthrough — any JSON out
			packs.CommandSpec{
				Path:    path,
				Timeout: 60 * time.Second,
			},
		)
		out = append(out, pack)
		logger.Info("command pack registered",
			"name", packName, "binary", path)
	}
	return out
}

// sanitizePackBasename ensures the pack name's basename matches
// the engine's pack-name conventions: lowercase, alnum + hyphen.
// Anything else gets stripped so a binary named "Upper Case!" can
// still be loaded (as cmd.uppercase) rather than rejected.
func sanitizePackBasename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
