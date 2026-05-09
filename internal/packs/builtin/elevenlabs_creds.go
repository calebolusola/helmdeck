// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package builtin

// elevenlabs_creds.go — shared credential resolver for the two ElevenLabs-
// dependent packs (podcast.generate, slides.narrate). Centralizes the
// resolve-order ladder defined in #138 so both packs see the same
// behavior and a future audit-log addition only needs to touch one place.

import (
	"context"
	"os"

	"github.com/tosin2013/helmdeck/internal/vault"
)

const (
	// envHelmdeckElevenLabsKey is the canonical env var. The vault
	// env-hydrate routine (#142) auto-imports it as elevenlabsCredCanonical;
	// we still check os.Getenv as a last-resort so operators who
	// disable hydrate (or run a non-vault flow) aren't stuck.
	envHelmdeckElevenLabsKey = "HELMDECK_ELEVENLABS_API_KEY"

	// elevenlabsCredCanonical is the vault-name set by env-hydrate
	// (#142) and historically by the README. Always tried first when
	// no explicit credential is named.
	elevenlabsCredCanonical = "elevenlabs-key"

	// elevenlabsCredAlias is the back-compat alias for operators who
	// created their credential under the natural-feeling
	// "elevenlabs-api-key" name (matches HELMDECK_ELEVENLABS_API_KEY
	// minus the prefix). Tried after the canonical name so existing
	// installs don't have to rename.
	elevenlabsCredAlias = "elevenlabs-api-key"
)

// elevenLabsKeySource identifies which step of the resolve ladder
// produced the key. Returned alongside the apiKey so handlers can
// log a helpful hint on success and a precise failure-mode list when
// nothing matches.
type elevenLabsKeySource string

const (
	keySrcExplicit  elevenLabsKeySource = "credential-input"
	keySrcCanonical elevenLabsKeySource = "vault:" + elevenlabsCredCanonical
	keySrcAlias     elevenLabsKeySource = "vault:" + elevenlabsCredAlias
	keySrcEnv       elevenLabsKeySource = "env:" + envHelmdeckElevenLabsKey
	keySrcNone      elevenLabsKeySource = ""
)

// resolveElevenLabsKey runs the four-step ladder defined in #138:
//   1. Explicit `credential` input
//   2. Vault entry under elevenlabsCredCanonical (env-hydrate fills this)
//   3. Vault entry under elevenlabsCredAlias (back-compat alias)
//   4. os.Getenv(envHelmdeckElevenLabsKey)  (last-resort)
//
// Returns the first match. If `explicit` is non-empty and not found,
// the function continues down the ladder rather than failing — the
// alternative would surprise operators who ship `credential` only as
// a hint and rely on env-hydrate for the real value.
func resolveElevenLabsKey(ctx context.Context, vs *vault.Store, explicit string) (string, elevenLabsKeySource) {
	tryVault := func(name string) string {
		if vs == nil || name == "" {
			return ""
		}
		res, err := vs.ResolveByName(ctx, vault.Actor{Subject: "*"}, name)
		if err != nil {
			return ""
		}
		return string(res.Plaintext)
	}

	if explicit != "" {
		if k := tryVault(explicit); k != "" {
			return k, keySrcExplicit
		}
		// Fall through — explicit name didn't resolve, but we don't
		// hard-fail here; the canonical/env paths might still cover.
	}
	if k := tryVault(elevenlabsCredCanonical); k != "" {
		return k, keySrcCanonical
	}
	if k := tryVault(elevenlabsCredAlias); k != "" {
		return k, keySrcAlias
	}
	if k := os.Getenv(envHelmdeckElevenLabsKey); k != "" {
		return k, keySrcEnv
	}
	return "", keySrcNone
}

// elevenLabsMissingCredentialMessage is the canonical error body for
// the "no key found anywhere" case. Centralized so both packs return
// identical guidance — operators who hit it once shouldn't have to
// translate two different phrasings.
const elevenLabsMissingCredentialMessage = "ElevenLabs key not found. " +
	"Set HELMDECK_ELEVENLABS_API_KEY in deploy/compose/.env.local — " +
	"it auto-imports into the vault as 'elevenlabs-key' on startup (#142). " +
	"Or POST a credential named 'elevenlabs-key' to /api/v1/vault/credentials. " +
	"To produce a silence-padded MP3 instead (CI smoke / placeholder use), " +
	"pass `\"allow_silent_output\": true`."
