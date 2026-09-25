package notification

import (
 "context"
 "crypto/sha256"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "aiw/internal/workflow"
)

func TestNotificationOriginalSnapshotBinding(t *testing.T) {
 p := processFixture(t)
 path := filepath.Join(p.ProjectRoot, "aiw.toml")
 body, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
 n := workflow.Notification{Managed: &workflow.NotificationMessage{ConfigSnapshot: fmt.Sprintf("%x", sha256.Sum256(body))}}
 if _, err := p.Preflight(context.Background(), n); err != nil { t.Fatal(err) }
 // Same TOML meaning, different original bytes: the original snapshot must reject it.
 if err := os.WriteFile(path, append(body, []byte("\n# changed bytes\n")...), 0600); err != nil { t.Fatal(err) }
 if _, err := p.Preflight(context.Background(), n); err == nil || err.Error() != "notification original configuration changed" { t.Fatalf("snapshot change not rejected: %v", err) }
}

func TestNotificationDisabledProcessConfiguration(t *testing.T) {
 p := processFixture(t)
 path := filepath.Join(p.ProjectRoot, "aiw.toml")
 if err := os.Remove(path); err != nil { t.Fatal(err) }
 config, err := p.Preflight(context.Background(), workflow.Notification{})
 if err != nil || config.Enabled || config.Ready { t.Fatalf("missing config: %+v %v", config, err) }
 body := "[notifications]\nschema_version = 1\nenabled = false\n"
 if err := os.WriteFile(path, []byte(body), 0600); err != nil { t.Fatal(err) }
 config, err = p.Preflight(context.Background(), workflow.Notification{})
 if err != nil || config.Enabled || config.Ready { t.Fatalf("disabled config: %+v %v", config, err) }
}

func TestNotificationRequestRejectedBeforeProcess(t *testing.T) {
 p := processFixture(t)
 // Cancellation would return process-result-unknown if request validation were bypassed.
 ctx, cancel := context.WithCancel(context.Background()); cancel()
 for _, request := range []any{map[string]string{"text": strings.Repeat("x", 64*1024)}, make(chan int)} {
  var config workflow.NotificationConfig
  if err := p.invoke(ctx, "preflight", request, &config); err == nil || err.Error() != "notification request limit" { t.Fatalf("request was not rejected before process: %v", err) }
 }
}
