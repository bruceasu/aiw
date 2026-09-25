package notification

import (
 "context"
 "crypto/sha256"
 "fmt"
 "os"
 "path/filepath"
 "testing"
 "aiw/internal/workflow"
)

func processFixture(t *testing.T) PluginDispatcher {
 t.Helper()
 root := t.TempDir()
 if err := os.Mkdir(filepath.Join(root, "plugins"), 0700); err != nil { t.Fatal(err) }
 for _, name := range []string{"aiw-notify.py", "send_teams_msg.py"} {
  body, err := os.ReadFile(filepath.Join("..", "..", "plugins", name)); if err != nil { t.Fatal(err) }
  if err := os.WriteFile(filepath.Join(root, "plugins", name), body, 0600); err != nil { t.Fatal(err) }
 }
 writeConsoleConfig(t, root, "fixture")
 return PluginDispatcher{ProjectRoot: root}
}

func writeConsoleConfig(t *testing.T, root, target string) {
 t.Helper()
 body := fmt.Sprintf("[notifications]\nschema_version = 1\nenabled = true\nchannel = \"console\"\ntarget_id = %q\n", target)
 if err := os.WriteFile(filepath.Join(root, "aiw.toml"), []byte(body), 0600); err != nil { t.Fatal(err) }
}

func TestNotificationPythonConsoleRoundTrip(t *testing.T) {
 p := processFixture(t)
 config, err := p.Preflight(context.Background(), workflow.Notification{}); if err != nil { t.Fatal(err) }
 if !config.Enabled || !config.Ready || config.Channel != "console" { t.Fatalf("unexpected config: %+v", config) }
 text := "Local notification fixture"
 n := workflow.Notification{ID: "fixture", Managed: &workflow.NotificationMessage{Version: 2, Text: text, ContentDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(text))), Config: config, Attempts: []workflow.NotificationAttempt{{ID: "attempt-fixture"}}}}
 result := p.Send(context.Background(), n)
 if !workflow.ValidNotificationResult(n, result) || result.Outcome != "attempted" || !result.Attempted { t.Fatalf("unexpected result: %+v", result) }
 writeConsoleConfig(t, p.ProjectRoot, "changed")
 result = p.Send(context.Background(), n)
 if result.Outcome != "configuration_failure" || result.Attempted { t.Fatalf("changed target accepted: %+v", result) }
}

func TestNotificationPythonRejectsChangedScript(t *testing.T) {
 for _, name := range []string{"aiw-notify.py", "send_teams_msg.py"} {
  t.Run(name, func(t *testing.T) {
   p := processFixture(t)
   f, err := os.OpenFile(filepath.Join(p.ProjectRoot, "plugins", name), os.O_APPEND|os.O_WRONLY, 0600); if err != nil { t.Fatal(err) }
   _, err = f.WriteString("\n# changed fixture\n"); closeErr := f.Close(); if err != nil || closeErr != nil { t.Fatal("fixture write failed") }
   if _, err := p.Preflight(context.Background(), workflow.Notification{}); err == nil || err.Error() != "unsupported notification script version" { t.Fatalf("unexpected identity result: %v", err) }
  })
 }
}

func TestNotificationPythonCancelledContext(t *testing.T) {
 p := processFixture(t)
 ctx, cancel := context.WithCancel(context.Background()); cancel()
 if _, err := p.Preflight(ctx, workflow.Notification{}); err == nil || err.Error() != "notification process result unknown" { t.Fatalf("unexpected cancelled result: %v", err) }
}
