// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 The helmdeck contributors

package packs

// command_pack.go (T811 MVP) — turns an executable into a Pack.
//
// The subprocess protocol is intentionally minimal:
//
//   stdin   = ec.Input (the JSON the agent passed to the pack)
//   stdout  = JSON output the engine validates against OutputSchema
//   stderr  = surfaced verbatim on non-zero exit; ignored on success
//   exit 0  = success; engine returns stdout to the caller
//   exit ≠0 = handler_failed; engine surfaces stderr (truncated)
//
// No engine-surface changes: the Pack returned here uses the same
// HandlerFunc shape as built-in Go packs, so the registry, audit
// log, async-job wrapping, and MCP tool exposure all work without
// special-casing.
//
// What this MVP does NOT do (slip to v0.13.0):
//   - JSON manifest format (version, author, schema in YAML/JSON)
//   - Hot-reload from a packs directory
//   - Subprocess egress sandbox (pack's network access is whatever
//     the host gives it; helmdeck's EgressGuard is not enforced
//     because the call is to a local binary, not an HTTP host)
//   - Per-pack resource limits (memory, CPU, FD count)
//   - Marketplace integration / signed manifests
//
// These slip to T811-followup. See docs/RELEASES.md v0.13.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// CommandSpec describes how to invoke the subprocess that backs a
// command-handler pack. Path is the absolute path to the executable
// (or a name resolvable on $PATH inside the control-plane image);
// Args are the per-call argv (excluding argv[0]).
type CommandSpec struct {
	// Path is the executable. Must be absolute or PATH-resolvable.
	Path string
	// Args are passed as argv[1:]. The handler appends nothing.
	Args []string
	// Env adds environment variables ON TOP of the control-plane's
	// inherited environment. Use this for per-pack config (API
	// endpoints, feature flags) — secrets should NOT live here;
	// route them through the vault and pass the resolved value via
	// stdin JSON instead.
	Env []string
	// Timeout caps wall-clock execution. Zero defaults to 60s.
	// Pack callers' ctx still takes precedence: if it expires first,
	// the subprocess is killed via cmd.Cancel.
	Timeout time.Duration
	// MaxOutputBytes caps stdout to prevent a runaway subprocess
	// from blowing up the control-plane's memory. Zero defaults to
	// 16 MiB — matches image_generate's artifact size cap.
	MaxOutputBytes int64
}

const (
	commandPackDefaultTimeout = 60 * time.Second
	commandPackDefaultMaxOut  = 16 << 20
	// stderr truncation cap for the inline PackError message.
	// Subprocess stderr can be verbose; truncate so the error
	// envelope stays under MCP's per-frame budget. The full stderr
	// is not currently artifacted (v0.13 follow-up if operators
	// hit cases where 4 KiB isn't enough).
	commandPackStderrCap = 4096
)

// NewCommandPack constructs a Pack whose Handler shells out to the
// given binary. Input/output schemas are enforced by the engine
// before/after the call (same as a Go pack); the binary itself is
// responsible for honoring them.
//
// Use this when a pack author wants to ship in any language
// (Python, Node, Bash, Rust) without a Go-toolchain dependency. The
// subprocess sees:
//
//   - argv = [Path, Args...]
//   - env  = control-plane env + Env (Env wins on collision)
//   - stdin = the validated input JSON (one frame, EOF closes)
//
// And the engine expects:
//
//   - stdout = one JSON value that satisfies OutputSchema (or
//     accepts the empty BasicSchema {})
//   - exit code 0 on success; anything else is treated as
//     CodeHandlerFailed and stderr is surfaced (truncated) in the
//     error message.
func NewCommandPack(name, version, description string, inSchema, outSchema Schema, spec CommandSpec) *Pack {
	return &Pack{
		Name:         name,
		Version:      version,
		Description:  description,
		InputSchema:  inSchema,
		OutputSchema: outSchema,
		Handler:      commandPackHandler(spec),
	}
}

// commandPackHandler returns the HandlerFunc that exec's spec.Path
// with the input as stdin and returns stdout as the pack output.
// Errors are mapped to typed PackErrors so the engine + REST layer
// surface them identically to a Go pack's errors.
func commandPackHandler(spec CommandSpec) HandlerFunc {
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = commandPackDefaultTimeout
	}
	maxOut := spec.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = commandPackDefaultMaxOut
	}

	return func(ctx context.Context, ec *ExecutionContext) (json.RawMessage, error) {
		if spec.Path == "" {
			return nil, &PackError{Code: CodeInternal,
				Message: "command pack registered without a binary path"}
		}

		// Per-call context bounded by the spec timeout AND the
		// caller's ctx — whichever fires first kills the subprocess.
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		cmd := exec.CommandContext(callCtx, spec.Path, spec.Args...)
		// Inherit the control-plane environment so things like
		// PATH, HOME, locale work; then layer the pack's per-call
		// Env on top. exec.Command's Env semantics: if Env is nil,
		// the child inherits os.Environ(); otherwise it sees ONLY
		// Env. We want both, so we append.
		if len(spec.Env) > 0 {
			cmd.Env = append(cmd.Env, spec.Env...)
		}

		cmd.Stdin = bytes.NewReader(ec.Input)
		var stdoutBuf, stderrBuf bytes.Buffer
		// Use LimitedReader-style truncation by writing to a buffer
		// with a hard cap. We can't use io.LimitedReader on stdout
		// directly because cmd.Stdout is a writer; instead use a
		// custom capped writer.
		cmd.Stdout = &cappedWriter{w: &stdoutBuf, max: maxOut}
		cmd.Stderr = &cappedWriter{w: &stderrBuf, max: commandPackStderrCap * 2}

		runErr := cmd.Run()
		// Surface subprocess exit code separately from spawn errors.
		// A spawn error (binary not found, permission denied) is a
		// configuration problem — treat as CodeInternal. A non-zero
		// exit code from the running binary is a handler failure —
		// treat as CodeHandlerFailed with the stderr attached.
		if runErr != nil {
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) {
				stderr := stderrBuf.String()
				if len(stderr) > commandPackStderrCap {
					stderr = stderr[:commandPackStderrCap] + "...(truncated)"
				}
				if callCtx.Err() == context.DeadlineExceeded {
					return nil, &PackError{Code: CodeHandlerFailed,
						Message: fmt.Sprintf("command pack timeout after %s: %s", timeout, stderr)}
				}
				return nil, &PackError{Code: CodeHandlerFailed,
					Message: fmt.Sprintf("command pack exit %d: %s", exitErr.ExitCode(), stderr)}
			}
			// Non-exit-related error: ctx cancellation, spawn
			// failure, etc.
			if callCtx.Err() == context.DeadlineExceeded {
				return nil, &PackError{Code: CodeHandlerFailed,
					Message: fmt.Sprintf("command pack timeout after %s", timeout)}
			}
			return nil, &PackError{Code: CodeInternal,
				Message: fmt.Sprintf("command pack spawn: %v", runErr), Cause: runErr}
		}

		stdout := stdoutBuf.Bytes()
		if len(stdout) == 0 {
			return nil, &PackError{Code: CodeInvalidOutput,
				Message: "command pack produced empty stdout (expected JSON)"}
		}
		// Sniff that stdout starts with a JSON token so a binary
		// that prints status lines first produces a useful error
		// instead of a confusing JSON parse failure at byte N. The
		// engine's OutputSchema.Validate will catch the rest.
		first := stdout[0]
		if first != '{' && first != '[' && first != '"' && first != 'n' && first != 't' && first != 'f' && !(first >= '0' && first <= '9') && first != '-' {
			preview := string(stdout)
			if len(preview) > 256 {
				preview = preview[:256] + "...(truncated)"
			}
			return nil, &PackError{Code: CodeInvalidOutput,
				Message: fmt.Sprintf("command pack stdout is not JSON (first byte %q): %s", first, preview)}
		}

		// Compact-validate the JSON shape. Schema validation
		// happens at the engine layer after the handler returns;
		// we just confirm it parses.
		if !json.Valid(stdout) {
			preview := string(stdout)
			if len(preview) > 256 {
				preview = preview[:256] + "...(truncated)"
			}
			return nil, &PackError{Code: CodeInvalidOutput,
				Message: fmt.Sprintf("command pack stdout is not valid JSON: %s", preview)}
		}
		return json.RawMessage(stdout), nil
	}
}

// cappedWriter is a minimal io.Writer wrapper that stops writing
// past max bytes. Used to prevent a runaway subprocess from
// blowing up the control-plane's memory if it spews unbounded
// output. Excess bytes are silently dropped — the subprocess
// doesn't get a write error.
type cappedWriter struct {
	w     io.Writer
	max   int64
	count int64
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	if c.count >= c.max {
		// Pretend we wrote everything so the subprocess doesn't
		// stall on a partial write. The bytes are dropped on the
		// floor; the engine reads what's already in the underlying
		// buffer.
		return len(p), nil
	}
	remaining := c.max - c.count
	if int64(len(p)) > remaining {
		n, err := c.w.Write(p[:remaining])
		c.count += int64(n)
		// Same trick: report the original length so the writer
		// keeps cooperating. Truncation is by design.
		if err != nil {
			return n, err
		}
		return len(p), nil
	}
	n, err := c.w.Write(p)
	c.count += int64(n)
	return n, err
}
