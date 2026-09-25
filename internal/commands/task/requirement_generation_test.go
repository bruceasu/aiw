package task

import "testing"

func TestParsePrepareSpecArgs(t *testing.T) {
	for _, args := range [][]string{{"orders"}, {"orders", "--candidate", "candidate.json"}, {"orders", "--regenerate"}} {
		id, options, err := parsePrepareSpecArgs(args)
		if err != nil || id != "orders" { t.Fatalf("%v: %v", args, err) }
		if len(args) == 3 && options.CandidatePath != "candidate.json" { t.Fatal("missing candidate path") }
		if len(args) == 2 && !options.Regenerate { t.Fatal("missing regenerate flag") }
	}
	for _, args := range [][]string{nil, {".."}, {"orders", "--candidate"}, {"orders", "--candidate", ""}, {"orders", "--regenerate", "--candidate", "candidate.json"}, {"orders", "--unknown"}} {
		if _, _, err := parsePrepareSpecArgs(args); err == nil { t.Fatalf("accepted %v", args) }
	}
}
