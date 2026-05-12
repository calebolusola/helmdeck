// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package packs

// Tests for the T811 subprocess pack type. We use the classic Go
// "self-exec" pattern: the test binary itself can act as the
// subprocess when invoked with HELMDECK_PACK_TEST_FIXTURE=<mode>.
// This avoids needing python/bash/jq in the test environment.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain dispatches the self-exec fixture modes. Real Go tests
// run when HELMDECK_PACK_TEST_FIXTURE is empty.
func TestMain(m *testing.M) {
	switch os.Getenv("HELMDECK_PACK_TEST_FIXTURE") {
	case "":
		os.Exit(m.Run())
	case "echo":
		// Read JSON from stdin, write it back to stdout. Used as a
		// trivial happy-path subprocess.
		body, _ := io.ReadAll(os.Stdin)
		_, _ = os.Stdout.Write(body)
		os.Exit(0)
	case "uppercase":
		// Read JSON {"text": "..."} from stdin, return {"text":
		// upper(...)}.
		body, _ := io.ReadAll(os.Stdin)
		var in struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			fmt.Fprintf(os.Stderr, "parse: %v", err)
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"text": strings.ToUpper(in.Text)})
		os.Exit(0)
	case "fail":
		fmt.Fprint(os.Stderr, "subprocess deliberate failure: bad input\n")
		os.Exit(7)
	case "non-json":
		fmt.Fprint(os.Stdout, "this is not JSON\n")
		os.Exit(0)
	case "empty":
		os.Exit(0)
	case "slow":
		time.Sleep(5 * time.Second)
		_, _ = os.Stdout.Write([]byte(`{"ok":true}`))
		os.Exit(0)
	case "binary":
		// Emit non-JSON bytes — sniff branch.
		_, _ = os.Stdout.Write([]byte{0x01, 0x02, 0x03})
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown fixture mode: %s", os.Getenv("HELMDECK_PACK_TEST_FIXTURE"))
		os.Exit(99)
	}
}

// selfExec returns a CommandSpec that re-invokes the test binary
// in the given fixture mode. Tests use this everywhere instead of
// shipping a separate helper binary.
func selfExec(t *testing.T, mode string) CommandSpec {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return CommandSpec{
		Path: exe,
		Env:  []string{"HELMDECK_PACK_TEST_FIXTURE=" + mode},
	}
}

func runCommandPack(t *testing.T, pack *Pack, input string) (json.RawMessage, error) {
	t.Helper()
	ec := &ExecutionContext{
		Pack:  pack,
		Input: json.RawMessage(input),
	}
	return pack.Handler(context.Background(), ec)
}

func TestCommandPack_HappyPath_Echo(t *testing.T) {
	pack := NewCommandPack("cmd.echo", "v1", "echo",
		BasicSchema{}, BasicSchema{},
		selfExec(t, "echo"))

	body := `{"hello":"world","n":42}`
	out, err := runCommandPack(t, pack, body)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	// Echo subprocess writes stdin verbatim. Strip surrounding
	// whitespace because json.Marshal in the fixture may add a
	// trailing newline.
	got := strings.TrimSpace(string(out))
	if got != body {
		t.Errorf("output = %q, want %q", got, body)
	}
}

func TestCommandPack_TransformsInput(t *testing.T) {
	pack := NewCommandPack("cmd.upper", "v1", "uppercase the `text` field",
		BasicSchema{Required: []string{"text"}, Properties: map[string]string{"text": "string"}},
		BasicSchema{Required: []string{"text"}, Properties: map[string]string{"text": "string"}},
		selfExec(t, "uppercase"))

	out, err := runCommandPack(t, pack, `{"text":"hello"}`)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if parsed["text"] != "HELLO" {
		t.Errorf("text = %q, want HELLO", parsed["text"])
	}
}

func TestCommandPack_NonZeroExitSurfacesStderr(t *testing.T) {
	pack := NewCommandPack("cmd.fail", "v1", "always fail",
		BasicSchema{}, BasicSchema{},
		selfExec(t, "fail"))

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) || perr.Code != CodeHandlerFailed {
		t.Fatalf("err = %v, want CodeHandlerFailed", err)
	}
	if !strings.Contains(perr.Message, "exit 7") {
		t.Errorf("error should include exit code, got %q", perr.Message)
	}
	if !strings.Contains(perr.Message, "deliberate failure") {
		t.Errorf("error should include stderr verbatim, got %q", perr.Message)
	}
}

func TestCommandPack_NonJSONStdoutFailsLoud(t *testing.T) {
	pack := NewCommandPack("cmd.bad", "v1", "non-json output",
		BasicSchema{}, BasicSchema{},
		selfExec(t, "non-json"))

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) || perr.Code != CodeInvalidOutput {
		t.Fatalf("err = %v, want CodeInvalidOutput", err)
	}
	if !strings.Contains(perr.Message, "not JSON") && !strings.Contains(perr.Message, "not valid JSON") {
		t.Errorf("error should mention JSON validity, got %q", perr.Message)
	}
}

func TestCommandPack_EmptyStdoutFails(t *testing.T) {
	pack := NewCommandPack("cmd.empty", "v1", "no output",
		BasicSchema{}, BasicSchema{},
		selfExec(t, "empty"))

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) || perr.Code != CodeInvalidOutput {
		t.Fatalf("err = %v, want CodeInvalidOutput, got %v", err, err)
	}
	if !strings.Contains(perr.Message, "empty stdout") {
		t.Errorf("error should mention empty stdout, got %q", perr.Message)
	}
}

func TestCommandPack_TimeoutKillsSubprocess(t *testing.T) {
	spec := selfExec(t, "slow")
	spec.Timeout = 50 * time.Millisecond
	pack := NewCommandPack("cmd.slow", "v1", "deliberately slow",
		BasicSchema{}, BasicSchema{},
		spec)

	start := time.Now()
	_, err := runCommandPack(t, pack, `{}`)
	dur := time.Since(start)
	if dur > 2*time.Second {
		t.Errorf("subprocess should have been killed, ran for %s", dur)
	}
	var perr *PackError
	if !errors.As(err, &perr) {
		t.Fatalf("err = %v, want PackError", err)
	}
	if !strings.Contains(perr.Message, "timeout") {
		t.Errorf("error should mention timeout, got %q", perr.Message)
	}
}

func TestCommandPack_OutputSchemaValidatesAtEngine(t *testing.T) {
	// The handler itself only validates "is JSON". Schema mismatch
	// is caught at the engine layer (engine.go runs OutputSchema
	// validation after the handler returns). Here we directly
	// invoke the handler so any schema check would be skipped —
	// just confirm the handler returns the raw JSON unmodified.
	pack := NewCommandPack("cmd.echo", "v1", "echo",
		BasicSchema{}, BasicSchema{Required: []string{"required_field_not_present"}},
		selfExec(t, "echo"))

	// Handler returns the JSON; the engine would then reject it
	// because OutputSchema requires "required_field_not_present".
	out, err := runCommandPack(t, pack, `{"hello":"world"}`)
	if err != nil {
		t.Fatalf("handler should not enforce OutputSchema directly: %v", err)
	}
	if !bytes.Contains(out, []byte(`"hello":"world"`)) {
		t.Errorf("output should be the raw JSON, got %s", out)
	}
}

func TestCommandPack_MissingPathFailsCleanly(t *testing.T) {
	pack := NewCommandPack("cmd.broken", "v1", "no path configured",
		BasicSchema{}, BasicSchema{},
		CommandSpec{ /* Path: "" */ })

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) || perr.Code != CodeInternal {
		t.Fatalf("err = %v, want CodeInternal", err)
	}
	if !strings.Contains(perr.Message, "binary path") {
		t.Errorf("error should mention the missing path, got %q", perr.Message)
	}
}

func TestCommandPack_NonexistentBinarySurfacesSpawnError(t *testing.T) {
	pack := NewCommandPack("cmd.ghost", "v1", "binary does not exist",
		BasicSchema{}, BasicSchema{},
		CommandSpec{Path: filepath.Join(t.TempDir(), "does-not-exist")})

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) {
		t.Fatalf("err = %v, want PackError", err)
	}
	// Could be CodeInternal (spawn) or CodeHandlerFailed depending
	// on how the OS surfaces the missing-binary error; just make
	// sure we got a typed error and not a bare exec.Error.
	if !strings.Contains(perr.Message, "does-not-exist") && !strings.Contains(perr.Message, "spawn") && !strings.Contains(perr.Message, "no such file") {
		t.Errorf("error should reference the missing binary, got %q", perr.Message)
	}
}

func TestCommandPack_RawBinaryOutputCaughtBySniff(t *testing.T) {
	pack := NewCommandPack("cmd.binary", "v1", "writes raw binary",
		BasicSchema{}, BasicSchema{},
		selfExec(t, "binary"))

	_, err := runCommandPack(t, pack, `{}`)
	var perr *PackError
	if !errors.As(err, &perr) || perr.Code != CodeInvalidOutput {
		t.Fatalf("err = %v, want CodeInvalidOutput", err)
	}
	if !strings.Contains(perr.Message, "not JSON") {
		t.Errorf("error should mention JSON validity, got %q", perr.Message)
	}
}

func TestCappedWriter_TruncatesAtMax(t *testing.T) {
	var buf bytes.Buffer
	cw := &cappedWriter{w: &buf, max: 10}

	n, err := cw.Write([]byte("hello world this is a long string"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	// We always claim the full length so the source doesn't stall.
	if n != 33 {
		t.Errorf("Write returned n=%d, want 33 (we lie to the source)", n)
	}
	if buf.String() != "hello worl" {
		t.Errorf("buf = %q, want %q (first 10 bytes only)", buf.String(), "hello worl")
	}
	// Second write should be dropped entirely.
	n2, err := cw.Write([]byte("MORE"))
	if err != nil {
		t.Fatalf("write 2: %v", err)
	}
	if n2 != 4 {
		t.Errorf("Write 2 returned n=%d, want 4", n2)
	}
	if buf.String() != "hello worl" {
		t.Errorf("buf after second write = %q, want unchanged", buf.String())
	}
}
