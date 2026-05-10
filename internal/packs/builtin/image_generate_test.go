// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tosin2013/helmdeck/internal/packs"
	"github.com/tosin2013/helmdeck/internal/store"
	"github.com/tosin2013/helmdeck/internal/vault"
)

// stubFalAPI returns a test server emulating fal.run/{model_id} +
// the CDN URL it points at. The fal endpoint accepts POST + Auth
// header "Key <token>" and returns a synthetic 1×1 PNG URL pointing
// back at itself; the CDN endpoint serves the actual PNG bytes.
//
// Tests redirect ImageGenFalBaseURL to the stub via the package var.
func stubFalAPI(t *testing.T, wantKey string, numImages int) (*httptest.Server, []byte) {
	t.Helper()
	// 1×1 transparent PNG (smallest possible PNG bytes — 67 bytes).
	pngBytes, _ := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP8//8/AwAI/AL+CWnKEQAAAABJRU5ErkJggg==")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The CDN download path serves the raw PNG.
		if strings.HasSuffix(r.URL.Path, "/cdn/image-1.png") {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
			return
		}
		// Everything else is treated as the fal.run model invoke.
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", 405)
			return
		}
		if r.Header.Get("Authorization") != "Key "+wantKey {
			http.Error(w, "bad key", 401)
			return
		}
		images := make([]map[string]any, 0, numImages)
		for i := 0; i < numImages; i++ {
			images = append(images, map[string]any{
				"url":          "http://" + r.Host + "/cdn/image-1.png",
				"content_type": "image/png",
				"width":        1, "height": 1,
			})
		}
		out := map[string]any{
			"images": images,
			"seed":   42,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	prev := ImageGenFalBaseURL
	ImageGenFalBaseURL = srv.URL
	t.Cleanup(func() { ImageGenFalBaseURL = prev })
	return srv, pngBytes
}

// vaultWithFalKey seeds an in-memory vault with the fal-key
// credential + wildcard ACL. Empty string seeds nothing.
func vaultWithFalKey(t *testing.T, key string) *vault.Store {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	master := make([]byte, 32)
	v, err := vault.New(db, master)
	if err != nil {
		t.Fatal(err)
	}
	if key == "" {
		return v
	}
	rec, err := v.Create(context.Background(), vault.CreateInput{
		Name:        "fal-key",
		Type:        vault.TypeAPIKey,
		HostPattern: "fal.run",
		Plaintext:   []byte(key),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Grant(context.Background(), rec.ID, vault.Grant{ActorSubject: "*"}); err != nil {
		t.Fatal(err)
	}
	return v
}

func runImageGenerate(t *testing.T, v *vault.Store, input string) (json.RawMessage, error) {
	t.Helper()
	pack := ImageGenerate(v, nil)
	ec := &packs.ExecutionContext{
		Pack:      pack,
		Input:     json.RawMessage(input),
		Artifacts: packs.NewMemoryArtifactStore(),
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return pack.Handler(context.Background(), ec)
}

func TestImageGenerate_HappyPath_SingleImage(t *testing.T) {
	stubFalAPI(t, "sk_test", 1)
	v := vaultWithFalKey(t, "sk_test")

	raw, err := runImageGenerate(t, v, `{
		"prompt": "a cat sitting on a podcast microphone",
		"model": "fal-ai/flux/schnell"
	}`)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var out struct {
		ImageArtifactKey string  `json:"image_artifact_key"`
		ImageSize        int64   `json:"image_size"`
		Engine           string  `json:"engine"`
		ModelUsed        string  `json:"model_used"`
		PromptUsed       string  `json:"prompt_used"`
		SeedUsed         int64   `json:"seed_used"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.ImageArtifactKey == "" {
		t.Error("image_artifact_key empty")
	}
	if out.ImageSize == 0 {
		t.Errorf("image_size = 0 (artifact should have bytes)")
	}
	if out.Engine != "fal" {
		t.Errorf("engine = %q, want fal", out.Engine)
	}
	if out.ModelUsed != "fal-ai/flux/schnell" {
		t.Errorf("model_used = %q", out.ModelUsed)
	}
	if out.PromptUsed != "a cat sitting on a podcast microphone" {
		t.Errorf("prompt_used round-trip wrong: %q", out.PromptUsed)
	}
	if out.SeedUsed != 42 {
		t.Errorf("seed_used = %d, want 42 (echoed from stub)", out.SeedUsed)
	}
}

func TestImageGenerate_HappyPath_MultipleImages(t *testing.T) {
	stubFalAPI(t, "sk_test", 3)
	v := vaultWithFalKey(t, "sk_test")
	raw, err := runImageGenerate(t, v, `{
		"prompt": "three cats",
		"num_images": 3
	}`)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var out struct {
		ImageArtifactKey  string   `json:"image_artifact_key"`
		ImageArtifactKeys []string `json:"image_artifact_keys"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ImageArtifactKey == "" {
		t.Error("primary image_artifact_key should still be present")
	}
	if len(out.ImageArtifactKeys) != 3 {
		t.Errorf("image_artifact_keys count = %d, want 3", len(out.ImageArtifactKeys))
	}
}

func TestImageGenerate_DefaultsToFluxSchnell(t *testing.T) {
	stubFalAPI(t, "sk_test", 1)
	v := vaultWithFalKey(t, "sk_test")
	raw, _ := runImageGenerate(t, v, `{"prompt":"test"}`)
	var out struct {
		ModelUsed string `json:"model_used"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ModelUsed != "fal-ai/flux/schnell" {
		t.Errorf("default model = %q, want fal-ai/flux/schnell", out.ModelUsed)
	}
}

func TestImageGenerate_NoCredential_HardFails(t *testing.T) {
	stubFalAPI(t, "sk_test", 1)
	v := vaultWithFalKey(t, "") // no key seeded
	t.Setenv("HELMDECK_FAL_KEY", "") // ensure env doesn't fall through
	_, err := runImageGenerate(t, v, `{"prompt":"test"}`)
	pe := &packs.PackError{}
	if !errors.As(err, &pe) || pe.Code != packs.CodeInvalidInput {
		t.Fatalf("want invalid_input, got %v", err)
	}
	if !strings.Contains(pe.Message, "fal.ai key not found") {
		t.Errorf("message should explain missing credential: %q", pe.Message)
	}
	if !strings.Contains(pe.Message, "HELMDECK_FAL_KEY") {
		t.Errorf("message should hint at the env var: %q", pe.Message)
	}
}

func TestImageGenerate_FallsBackToEnvVar(t *testing.T) {
	stubFalAPI(t, "sk_from_env", 1)
	v := vaultWithFalKey(t, "") // vault empty
	t.Setenv("HELMDECK_FAL_KEY", "sk_from_env")
	raw, err := runImageGenerate(t, v, `{"prompt":"test"}`)
	if err != nil {
		t.Fatalf("env-fallback handler: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil response from env-fallback path")
	}
}

func TestImageGenerate_BadEngine(t *testing.T) {
	v := vaultWithFalKey(t, "sk_test")
	_, err := runImageGenerate(t, v, `{"prompt":"test","engine":"replicate"}`)
	if err == nil || !strings.Contains(err.Error(), `engine must be "fal"`) {
		t.Errorf("want engine-rejected error, got %v", err)
	}
}

func TestImageGenerate_NumImagesValidation(t *testing.T) {
	v := vaultWithFalKey(t, "sk_test")
	cases := []struct {
		num  int
		want bool // wantError
	}{
		{0, false}, // 0 means "default to 1"
		{1, false},
		{4, false},
		{5, true},
		{-1, true},
	}
	for _, tc := range cases {
		stubFalAPI(t, "sk_test", maxInt(tc.num, 1))
		input, _ := json.Marshal(map[string]any{
			"prompt":     "test",
			"num_images": tc.num,
		})
		_, err := runImageGenerate(t, v, string(input))
		gotErr := err != nil
		if gotErr != tc.want {
			t.Errorf("num_images=%d: got error=%v want=%v (err: %v)", tc.num, gotErr, tc.want, err)
		}
	}
}

func TestImageGenerate_FalRejects401Surfaces(t *testing.T) {
	stubFalAPI(t, "sk_correct", 1)
	v := vaultWithFalKey(t, "sk_wrong")
	_, err := runImageGenerate(t, v, `{"prompt":"test"}`)
	pe := &packs.PackError{}
	if !errors.As(err, &pe) || pe.Code != packs.CodeHandlerFailed {
		t.Fatalf("want handler_failed, got %v", err)
	}
	if !strings.Contains(pe.Message, "fal.ai 401") {
		t.Errorf("message should surface 401: %q", pe.Message)
	}
}

func TestImageGenerate_EmptyPrompt(t *testing.T) {
	v := vaultWithFalKey(t, "sk_test")
	_, err := runImageGenerate(t, v, `{"prompt":""}`)
	if err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Errorf("want prompt-required, got %v", err)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
