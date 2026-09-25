package notification

import (
	"testing"
	"bytes"

	"aiw/internal/workflow"
)

func TestStrictNotificationProtocolRejectsAmbiguousResults(t *testing.T) {
	for _, raw := range []string{
		`{"notification_id":"n","status":"accepted","receipt":"local"}`,
		`{"schema_version":2,"notification_id":"n","attempt_id":"a","content_digest":"d","attempted":true,"outcome":"attempted","error_code":"","http_status":null,"extra":1}`,
		`{"schema_version":2,"notification_id":"n","attempt_id":"a","content_digest":"d","attempted":true,"outcome":"attempted","outcome":"network_failure","error_code":"","http_status":null}`,
		`{"schema_version":2,"notification_id":"n","attempt_id":"a","content_digest":"d","attempted":null,"outcome":"attempted","error_code":"","http_status":null}`,
		`{"schema_version":2,"notification_id":"n","attempt_id":"a","content_digest":"d","attempted":true,"outcome":"attempted","error_code":"","http_status":null} {}`,
	} {
		var result workflow.NotificationResult
		if strictResult([]byte(raw), &result) == nil { t.Fatalf("accepted ambiguous protocol: %s", raw) }
	}
}

func TestStrictNotificationProtocolAcceptsAttemptOnly(t *testing.T) {
	var result workflow.NotificationResult
	raw := `{"schema_version":2,"notification_id":"n","attempt_id":"a","content_digest":"d","attempted":true,"outcome":"attempted","error_code":"","http_status":200}`
	if err := strictResult([]byte(raw), &result); err != nil { t.Fatal(err) }
	if result.Outcome != "attempted" || !result.Attempted { t.Fatal("lost attempt observation") }
}

func TestStrictNotificationConfigProtocol(t *testing.T) {
 valid := `{"enabled":true,"ready":true,"channel":"console","config_reference":"config","target_reference":"target"}`
 var config workflow.NotificationConfig
 if err := strictResult([]byte(valid), &config); err != nil { t.Fatal(err) }
 if !config.Enabled || !config.Ready || config.Channel != "console" { t.Fatal("lost config fields") }
 for _, raw := range []string{
  `{"enabled":true,"ready":true,"channel":"console","config_reference":"config"}`,
  `{"enabled":true,"ready":null,"channel":"console","config_reference":"config","target_reference":"target"}`,
  `{"enabled":true,"ready":true,"channel":"console","config_reference":"config","target_reference":"target","extra":1}`,
  `{"enabled":true,"enabled":false,"channel":"console","config_reference":"config","target_reference":"target"}`,
  valid + `{}`,
 } {
  var rejected workflow.NotificationConfig
  if strictResult([]byte(raw), &rejected) == nil { t.Fatalf("accepted invalid configuration: %s", raw) }
 }
}

func TestNotificationOutputBounds(t *testing.T) {
 var output boundedOutput
 exact := bytes.Repeat([]byte{' '}, 64*1024)
 if n, err := output.Write(exact); n != len(exact) || err != nil || output.overflow { t.Fatal("exact output limit rejected") }
 if n, err := output.Write([]byte("overflow")); n != 8 || err != nil || !output.overflow || output.Len() != 64*1024 { t.Fatal("overflow was not bounded") }
 var result workflow.NotificationResult
 if strictResult(append(exact, ' '), &result) == nil { t.Fatal("oversized result accepted") }
 if strictResult([]byte{0xff}, &result) == nil { t.Fatal("invalid UTF-8 accepted") }
}
