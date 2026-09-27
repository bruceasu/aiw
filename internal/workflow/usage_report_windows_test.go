//go:build windows

package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTaskUsageReportFiltersAndKnownUnknownTotals(t *testing.T) {
	store := usageBudgetFixture(t)
	events := []UsageEvent{
		{
			TaskID: "task-1", WorkItemID: "wi-0001", AttemptID: "attempt-1",
			SessionID: "session-1", SessionTurn: 1, Provider: "codex", Model: "model-a",
			Profile: "fast", Outcome: "completed", RecordedAt: "2026-01-01T00:00:00Z",
			Usage: json.RawMessage(`{"availability":"known","input_tokens":{"state":"known","value":4},"output_tokens":{"state":"known","value":6},"total_tokens":{"state":"known","value":10},"cost_amount":{"state":"known","value":1},"cost_currency":{"state":"known","value":"USD"},"raw_response":"private evidence"}`),
		},
		{
			TaskID: "task-1", WorkItemID: "wi-0002", AttemptID: "attempt-2",
			SessionID: "session-2", SessionTurn: 1, Provider: "copilot", Model: "model-b",
			Profile: "balanced", Outcome: "failed", RecordedAt: "2026-01-02T00:00:00Z",
			Usage: json.RawMessage(`{"availability":"unknown","input_tokens":{"state":"known","value":3},"output_tokens":{"state":"unknown"},"total_tokens":{"state":"unknown"},"cost_amount":{"state":"unknown"},"cost_currency":{"state":"unknown"}}`),
		},
		{
			TaskID: "task-1", WorkItemID: "wi-0002", AttemptID: "attempt-3",
			SessionID: "session-3", SessionTurn: 1, Provider: "codex", Model: "model-c",
			Profile: "balanced", Outcome: "completed", RecordedAt: "2026-01-03T00:00:00Z",
			Usage: json.RawMessage(`{"availability":"known","input_tokens":{"state":"known","value":5},"output_tokens":{"state":"known","value":5},"total_tokens":{"state":"known","value":11},"cost_amount":{"state":"known","value":2},"cost_currency":{"state":"known","value":"EUR"}}`),
		},
	}
	for _, event := range events {
		if _, err := store.RecordUsageEvent("task-1", event); err != nil {
			t.Fatalf("record usage event %s: %v", event.AttemptID, err)
		}
	}

	from, _ := time.Parse(time.RFC3339, "2026-01-02T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-01-03T00:00:00Z")
	report, err := GetUsageReport(context.Background(), store, "task-1", UsageReportQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.Calls != 3 || report.Totals.UsageUnknown != 1 ||
		report.Totals.KnownInputTokens != 12 || report.Totals.KnownOutputTokens != 11 ||
		report.Totals.KnownTotalTokens != 21 || report.Totals.KnownCostByCurrency["USD"] != "1" ||
		report.Totals.KnownCostByCurrency["EUR"] != "2" {
		t.Fatalf("known and unknown usage totals were not kept separate: %+v", report.Totals)
	}
	if report.Outcomes["completed"] != 2 || report.Outcomes["failed"] != 1 {
		t.Fatalf("outcome counts do not match the reported calls: %+v", report.Outcomes)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private evidence") || strings.Contains(string(encoded), "raw_response") {
		t.Fatal("usage report exposed Provider raw evidence")
	}

	tests := []struct {
		name  string
		query UsageReportQuery
		want  []string
	}{
		{name: "work item", query: UsageReportQuery{WorkItemID: "wi-0002"}, want: []string{"attempt-2", "attempt-3"}},
		{name: "attempt", query: UsageReportQuery{AttemptID: "attempt-2"}, want: []string{"attempt-2"}},
		{name: "provider", query: UsageReportQuery{Provider: "codex"}, want: []string{"attempt-1", "attempt-3"}},
		{name: "profile", query: UsageReportQuery{Profile: "fast"}, want: []string{"attempt-1"}},
		{name: "time range", query: UsageReportQuery{From: &from, To: &to}, want: []string{"attempt-2", "attempt-3"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filtered, err := store.GetUsageReport(context.Background(), "task-1", test.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(filtered.Calls) != len(test.want) {
				t.Fatalf("got %d calls, want %d: %+v", len(filtered.Calls), len(test.want), filtered.Calls)
			}
			for i, call := range filtered.Calls {
				if string(call.AttemptID) != test.want[i] {
					t.Fatalf("call %d has Attempt %q, want %q", i, call.AttemptID, test.want[i])
				}
			}
		})
	}
}
