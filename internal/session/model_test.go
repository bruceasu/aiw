package session

import (
	"encoding/json"
	"testing"
)

func TestStatusWithoutManagedReferenceRemainsStandalone(t *testing.T) {
	status := Status{SchemaVersion: 1, Session: SessionInfo{ID: "standalone"}}
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Status
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Task != nil {
		t.Fatalf("standalone Session unexpectedly has Task reference: %#v", decoded.Task)
	}
}
