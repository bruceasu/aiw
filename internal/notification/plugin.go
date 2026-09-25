// Package notification implements the project-local aiw-notify v2 adapter.
package notification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"aiw/internal/workflow"
)

type PluginDispatcher struct { ProjectRoot string }

// Updated with reviewed bundled sources. Unknown local variants fail closed;
// no script is overwritten or silently replaced with a global installation.
const notifySourceDigest = "91e414c50dda8a91e3e526d697ca116a0c01c198d47e7297fb3afff8c00fb243"
const teamsSourceDigest = "601c9b87bb4300b341fa11c8ba30ccbe16818f87cde4470d9eb4777b0732010f"

type boundedOutput struct { bytes.Buffer; overflow bool }
func (b *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 64*1024-b.Len()
	if len(p) > remaining { p = p[:remaining]; b.overflow = true }
	_, _ = b.Buffer.Write(p)
	return n, nil
}

func supportedScript(root, name, expected string) (string, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil { return "", errors.New("notification project unavailable") }
	path, err := filepath.EvalSymlinks(filepath.Join(canonical, "plugins", name))
	if err != nil { return "", errors.New("notification script unavailable") }
	rel, err := filepath.Rel(canonical, path)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) { return "", errors.New("notification script outside project") }
	f, err := os.Open(path)
	if err != nil { return "", errors.New("notification script unavailable") }
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if err != nil || len(body) > 256*1024 { return "", errors.New("notification script unavailable") }
	// Source checkout line ending conversion does not change supported code.
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != expected { return "", errors.New("unsupported notification script version") }
	return path, nil
}

func strictResult(raw []byte, target any) error {
	if !utf8.Valid(raw) || len(raw) > 64*1024 { return errors.New("notification protocol") }
	// Reject duplicate/missing fields and null booleans before typed decoding.
	scan := json.NewDecoder(bytes.NewReader(raw))
	token, err := scan.Token()
	if err != nil || token != json.Delim('{') { return errors.New("notification protocol") }
	seen := map[string]bool{}
	for scan.More() {
		token, err := scan.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] { return errors.New("notification protocol") }
		seen[key] = true
		var value json.RawMessage
		if scan.Decode(&value) != nil || (bytes.Equal(value, []byte("null")) && key != "http_status") { return errors.New("notification protocol") }
	}
	want := 8
	if _, ok := target.(*workflow.NotificationConfig); ok { want = 5 }
	if len(seen) != want { return errors.New("notification protocol") }
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil { return errors.New("notification protocol") }
	if decoder.Decode(new(any)) != io.EOF { return errors.New("notification protocol") }
	return nil
}

func (p PluginDispatcher) invoke(parent context.Context, operation string, request any, target any) error {
	root, err := filepath.Abs(p.ProjectRoot)
	if err != nil || p.ProjectRoot == "" { return errors.New("notification project required") }
	script, err := supportedScript(root, "aiw-notify.py", notifySourceDigest)
	if err != nil { return err }
	if _, err := supportedScript(root, "send_teams_msg.py", teamsSourceDigest); err != nil { return err }
	body, err := json.Marshal(request)
	if err != nil || len(body) > 64*1024 { return errors.New("notification request limit") }
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "python", "-I", script, operation, "--json")
	command.Dir = root
	command.Stdin = bytes.NewReader(body)
	var output, diagnostics boundedOutput
	command.Stdout, command.Stderr = &output, &diagnostics
	command.WaitDelay = time.Second
	if err := command.Run(); err != nil { return errors.New("notification process result unknown") }
	if output.overflow || diagnostics.overflow { return errors.New("notification output limit") }
	return strictResult(output.Bytes(), target)
}

func (p PluginDispatcher) Preflight(ctx context.Context, n workflow.Notification) (workflow.NotificationConfig, error) {
	var config workflow.NotificationConfig
	err := p.invoke(ctx, "preflight", map[string]int{"schema_version": 2}, &config)
	if err == nil && config.Enabled && n.Managed != nil && n.Managed.Config.ConfigReference == "" && len(n.Managed.ConfigSnapshot) == 64 {
		f, readErr := os.Open(filepath.Join(p.ProjectRoot, "aiw.toml"))
		if readErr != nil { return config, errors.New("notification original configuration unavailable") }
		body, readErr := io.ReadAll(io.LimitReader(f, 64*1024+1))
		_ = f.Close()
		sum := sha256.Sum256(body)
		if readErr != nil || len(body) > 64*1024 || hex.EncodeToString(sum[:]) != n.Managed.ConfigSnapshot { return config, errors.New("notification original configuration changed") }
	}
	if err == nil && config.Enabled && (config.ConfigReference == "" || config.TargetReference == "" || (config.Channel != "console" && config.Channel != "teams")) { err = errors.New("notification configuration invalid") }
	return config, err
}

func (p PluginDispatcher) Send(ctx context.Context, n workflow.Notification) workflow.NotificationResult {
	unknown := workflow.UnknownNotificationResult(n)
	if n.Managed == nil { return unknown }
	m := n.Managed
	request := map[string]any{"schema_version": 2, "notification_id": n.ID, "attempt_id": unknown.AttemptID, "content_digest": m.ContentDigest, "text": m.Text, "channel": m.Config.Channel, "target_reference": m.Config.TargetReference, "config_reference": m.Config.ConfigReference}
	var result workflow.NotificationResult
	if err := p.invoke(ctx, "dispatch", request, &result); err != nil || !workflow.ValidNotificationResult(n, result) { return unknown }
	if m.Config.Channel == "console" && result.Outcome == "attempted" { _, _ = fmt.Fprintln(os.Stderr, m.Text) }
	return result
}

func (p PluginDispatcher) Dispatch(ctx context.Context, n workflow.Notification) (string, error) {
	result := p.Send(ctx, n)
	if result.Outcome != "attempted" { return "", errors.New("notification attempt failed") }
	return "local-attempt", nil
}
