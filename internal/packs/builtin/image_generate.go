// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package builtin

// image_generate.go (#71) — text → image via fal.ai's sync `fal.run`
// endpoint. Day 1 ships fal.ai because its sync API returns the
// generated image URL in the same POST response, no polling loop —
// which keeps the pack ~150 lines and lets it run inside the normal
// sync MCP tools/call timeout. Replicate (the issue's original
// suggestion) is queue+poll-only and will land as a community PR;
// the `engine` input field is here from day 1 so adding it is a
// new switch arm rather than a schema change.
//
// Default model is `fal-ai/flux/schnell` — fast, ~$0.003/image,
// indexed by image-gen leaderboards as the most cost-effective open
// model with photorealistic output. Operators pass their own
// `model` to use FLUX dev/pro/SDXL/etc.
//
// Credentials: vault entry `fal-key` (canonical) with
// `HELMDECK_FAL_KEY` env-var fallback, mirroring the #138 ladder
// shape from elevenlabs_creds.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tosin2013/helmdeck/internal/packs"
	"github.com/tosin2013/helmdeck/internal/security"
	"github.com/tosin2013/helmdeck/internal/vault"
)

const (
	// imageGenFalBaseURL is the fal.run sync endpoint root. Exported
	// as a var (below) so tests redirect without injecting clients.
	imageGenDefaultEngine = "fal"
	imageGenDefaultModel  = "fal-ai/flux/schnell"
	imageGenFalCredName   = "fal-key"
	imageGenFalEnvVar     = "HELMDECK_FAL_KEY"

	// Cap the downloaded image bytes — the largest reasonable
	// generation (1024×1024 PNG) is ~3 MiB; 16 MiB gives plenty of
	// headroom without letting a hostile/malformed response OOM the
	// control plane.
	imageGenMaxBytes = 16 << 20

	// Timeout for the synchronous fal.run call. Schnell is ~1-3s;
	// FLUX-pro can be 10-15s. 60s caps the upper bound while still
	// fitting within MCP's 60s default JSON-RPC timeout.
	imageGenHTTPTimeout = 60 * time.Second
)

// ImageGenFalBaseURL is the API host. Exported as a package var
// (rather than a const) so tests redirect to httptest stubs without
// threading a client through every call site — same pattern as
// internal/voices.ElevenLabsBaseURL.
var ImageGenFalBaseURL = "https://fal.run"

// ImageGenerate constructs the pack. Vault is required for the
// credential lookup; egress guard wraps the outbound call to
// catch operators pointing the pack at internal hosts (it shouldn't
// happen for fal.ai but the guard is consistent across packs).
func ImageGenerate(v *vault.Store, eg *security.EgressGuard) *packs.Pack {
	return &packs.Pack{
		Name:        "image.generate",
		Version:     "v1",
		Description: "Generate an image from a text prompt via fal.ai (FLUX schnell/dev/pro, SDXL, etc.). Requires HELMDECK_FAL_KEY in .env.local (auto-hydrated to vault as 'fal-key' once #142 lands), or pass `credential: \"<vault-name>\"` to use an explicit entry.",
		InputSchema: packs.BasicSchema{
			Required: []string{"prompt"},
			Properties: map[string]string{
				"prompt":     "string",
				"engine":     "string", // "fal" (default); future: "replicate"
				"model":      "string", // default fal-ai/flux/schnell
				"image_size": "string", // landscape_16_9 / square_hd / portrait_4_3 / etc. — model-specific
				"num_images": "number", // default 1; capped at 4 to bound cost-per-call
				"seed":       "number", // optional; fal.ai's reproducibility hook
				"credential": "string", // optional explicit vault name override
			},
		},
		OutputSchema: packs.BasicSchema{
			Required: []string{"image_artifact_key", "engine", "model_used"},
			Properties: map[string]string{
				"image_artifact_key":  "string",
				"image_size":          "number",
				"engine":              "string",
				"model_used":          "string",
				"prompt_used":         "string",
				"seed_used":           "number",
				"image_artifact_keys": "array", // present when num_images > 1
			},
		},
		Handler: imageGenerateHandler(v, eg),
	}
}

type imageGenerateInput struct {
	Prompt     string  `json:"prompt"`
	Engine     string  `json:"engine"`
	Model      string  `json:"model"`
	ImageSize  string  `json:"image_size"`
	NumImages  int     `json:"num_images"`
	Seed       int64   `json:"seed"`
	Credential string  `json:"credential"`
}

// ImageGenRequest is the parsed input for RunImageGen. Exported so
// content packs (podcast/slides/blog) that chain image.generate can
// reuse the helper without round-tripping through the pack registry
// and paying twice for vault/audit overhead.
//
// Field defaults match the pack schema: empty Engine → "fal", empty
// Model → "fal-ai/flux/schnell", zero NumImages → 1. The pack handler
// fills in defaults from JSON input; callers building this struct
// directly should match the same defaults or rely on RunImageGen's
// validation.
type ImageGenRequest struct {
	Prompt     string
	Engine     string
	Model      string
	ImageSize  string
	NumImages  int
	Seed       int64
	Credential string
}

// ImageGenResult is the typed output from RunImageGen. The pack
// handler turns this into a JSON output map; chained callers can use
// the fields directly (e.g., podcast.generate surfaces ArtifactKeys[0]
// as `cover_image_artifact_key`).
type ImageGenResult struct {
	ArtifactKeys []string // namespaced under ec.Pack.Name; len() = NumImages requested
	FirstSize    int64
	ModelUsed    string
	Engine       string
	PromptUsed   string
	SeedUsed     int64
}

// RunImageGen is the chainable entrypoint shared by image.generate's
// own handler and by content packs that auto-generate images
// (podcast.generate covers, slides.render heroes, blog.publish feature
// images). Artifacts are written via ec.Artifacts under ec.Pack.Name,
// so a podcast.generate caller gets `podcast.generate/image-000.png`
// keys rather than `image.generate/...` — the chained pack owns its
// own artifacts.
func RunImageGen(ctx context.Context, ec *packs.ExecutionContext, v *vault.Store, eg *security.EgressGuard, in ImageGenRequest) (*ImageGenResult, *packs.PackError) {
	if strings.TrimSpace(in.Prompt) == "" {
		return nil, &packs.PackError{Code: packs.CodeInvalidInput, Message: "prompt is required"}
	}
	engine := in.Engine
	if engine == "" {
		engine = imageGenDefaultEngine
	}
	if engine != "fal" {
		return nil, &packs.PackError{Code: packs.CodeInvalidInput,
			Message: fmt.Sprintf(`engine must be "fal" (got %q); other engines (e.g. "replicate") ship in future PRs`, engine)}
	}
	model := in.Model
	if model == "" {
		model = imageGenDefaultModel
	}
	num := in.NumImages
	if num == 0 {
		num = 1
	}
	if num < 1 || num > 4 {
		return nil, &packs.PackError{Code: packs.CodeInvalidInput,
			Message: "num_images must be between 1 and 4"}
	}
	if ec.Artifacts == nil {
		return nil, &packs.PackError{Code: packs.CodeInternal,
			Message: "image.generate requires an artifact store"}
	}

	apiKey := resolveFalKey(ctx, v, in.Credential)
	if apiKey == "" {
		return nil, &packs.PackError{Code: packs.CodeInvalidInput,
			Message: "fal.ai key not found. Set HELMDECK_FAL_KEY in deploy/compose/.env.local — it auto-imports into the vault as 'fal-key' on startup. Or POST a credential named 'fal-key' to /api/v1/vault/credentials."}
	}

	ec.Report(20, fmt.Sprintf("submitting fal.ai/%s", model))
	body := map[string]any{
		"prompt":     in.Prompt,
		"num_images": num,
	}
	if in.ImageSize != "" {
		body["image_size"] = in.ImageSize
	}
	if in.Seed != 0 {
		body["seed"] = in.Seed
	}
	bodyBytes, _ := json.Marshal(body)

	url := strings.TrimRight(ImageGenFalBaseURL, "/") + "/" + strings.TrimLeft(model, "/")
	if eg != nil {
		if err := eg.CheckURL(ctx, url); err != nil {
			return nil, &packs.PackError{Code: packs.CodeInvalidInput,
				Message: fmt.Sprintf("egress denied: %v", err), Cause: err}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &packs.PackError{Code: packs.CodeHandlerFailed, Message: err.Error(), Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Key "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: imageGenHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &packs.PackError{Code: packs.CodeHandlerFailed,
			Message: fmt.Sprintf("fal.ai request: %v", err), Cause: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &packs.PackError{Code: packs.CodeHandlerFailed,
			Message: fmt.Sprintf("fal.ai %d: %s", resp.StatusCode, truncateStr(string(respBody), 512))}
	}

	var parsed struct {
		Images []struct {
			URL         string `json:"url"`
			ContentType string `json:"content_type"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
		} `json:"images"`
		Seed int64 `json:"seed"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &packs.PackError{Code: packs.CodeHandlerFailed,
			Message: fmt.Sprintf("parse fal.ai response: %v", err), Cause: err}
	}
	if len(parsed.Images) == 0 {
		return nil, &packs.PackError{Code: packs.CodeHandlerFailed,
			Message: "fal.ai returned no images"}
	}

	ec.Report(60, fmt.Sprintf("downloading %d image(s)", len(parsed.Images)))
	artKeys := make([]string, 0, len(parsed.Images))
	var firstSize int64
	for i, img := range parsed.Images {
		imgBytes, ct, err := fetchImageBytes(ctx, img.URL, imgImageContentType(img.ContentType))
		if err != nil {
			return nil, &packs.PackError{Code: packs.CodeHandlerFailed,
				Message: fmt.Sprintf("download image %d: %v", i, err), Cause: err}
		}
		art, err := ec.Artifacts.Put(ctx, ec.Pack.Name,
			fmt.Sprintf("image-%03d.%s", i, contentTypeToExt(ct)),
			imgBytes, ct)
		if err != nil {
			return nil, &packs.PackError{Code: packs.CodeArtifactFailed,
				Message: err.Error(), Cause: err}
		}
		artKeys = append(artKeys, art.Key)
		if i == 0 {
			firstSize = art.Size
		}
	}

	ec.Report(95, "image.generate complete")
	return &ImageGenResult{
		ArtifactKeys: artKeys,
		FirstSize:    firstSize,
		ModelUsed:    model,
		Engine:       engine,
		PromptUsed:   in.Prompt,
		SeedUsed:     parsed.Seed,
	}, nil
}

func imageGenerateHandler(v *vault.Store, eg *security.EgressGuard) packs.HandlerFunc {
	return func(ctx context.Context, ec *packs.ExecutionContext) (json.RawMessage, error) {
		var in imageGenerateInput
		if err := json.Unmarshal(ec.Input, &in); err != nil {
			return nil, &packs.PackError{Code: packs.CodeInvalidInput, Message: err.Error(), Cause: err}
		}
		result, perr := RunImageGen(ctx, ec, v, eg, ImageGenRequest{
			Prompt:     in.Prompt,
			Engine:     in.Engine,
			Model:      in.Model,
			ImageSize:  in.ImageSize,
			NumImages:  in.NumImages,
			Seed:       in.Seed,
			Credential: in.Credential,
		})
		if perr != nil {
			return nil, perr
		}
		out := map[string]any{
			"image_artifact_key": result.ArtifactKeys[0],
			"image_size":         result.FirstSize,
			"engine":             result.Engine,
			"model_used":         result.ModelUsed,
			"prompt_used":        result.PromptUsed,
		}
		if result.SeedUsed != 0 {
			out["seed_used"] = result.SeedUsed
		}
		if len(result.ArtifactKeys) > 1 {
			out["image_artifact_keys"] = result.ArtifactKeys
		}
		return json.Marshal(out)
	}
}

// resolveFalKey walks the credential ladder for fal.ai. Mirrors the
// elevenLabs resolver in elevenlabs_creds.go but doesn't share code
// with it (the back-compat-alias step has no analogue here — fal.ai
// has only ever been documented as `fal-key`).
func resolveFalKey(ctx context.Context, v *vault.Store, explicit string) string {
	if v != nil && explicit != "" {
		if res, err := v.ResolveByName(ctx, vault.Actor{Subject: "*"}, explicit); err == nil {
			return string(res.Plaintext)
		}
	}
	if v != nil {
		if res, err := v.ResolveByName(ctx, vault.Actor{Subject: "*"}, imageGenFalCredName); err == nil {
			return string(res.Plaintext)
		}
	}
	return os.Getenv(imageGenFalEnvVar)
}

// fetchImageBytes downloads an image URL with a strict size cap +
// timeout. fal.ai returns CDN URLs that don't require auth headers
// (the signed URL itself encodes access).
func fetchImageBytes(ctx context.Context, url, fallbackCT string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{Timeout: imageGenHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("download %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, imageGenMaxBytes))
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("empty image body")
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = fallbackCT
	}
	if ct == "" {
		ct = "image/png" // fal.ai's most common default
	}
	return body, ct, nil
}

// imgImageContentType normalizes the fal.ai response's content_type
// field — sometimes returned as "image/jpeg", sometimes as "jpeg".
func imgImageContentType(s string) string {
	if s == "" {
		return ""
	}
	if strings.Contains(s, "/") {
		return s
	}
	return "image/" + s
}

// contentTypeToExt maps a MIME type to the file extension we use in
// the artifact filename. Defaults to "png" — the most common fal.ai
// output and always-correct for the FLUX family.
func contentTypeToExt(ct string) string {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png"
	}
}

// truncateStr is a small helper for inline error messages — same
// shape as truncStr in slides_narrate.go but doesn't share code so
// the two files stay independent. (truncStr's duplication has been
// noted; it'll consolidate when a third caller appears.)
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
