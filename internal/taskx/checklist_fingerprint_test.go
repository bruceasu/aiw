package taskx

import "testing"

func TestChecklistFingerprintIgnoresSourcePosition(t *testing.T) {
	first := []ChecklistItem{{Number: "1.1", Title: "first", Line: 1}, {Number: "1.2", Title: "second", Line: 2}}
	second := []ChecklistItem{{Number: "1.1", Title: "first", Line: 30}, {Number: "1.2", Title: "second", Line: 40}}
	if ChecklistFingerprint(first) != ChecklistFingerprint(second) {
		t.Fatal("fingerprint changed only because source lines moved")
	}
}
