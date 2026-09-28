package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestImplementationReportOutputSchemaContainsValidatedFields(t *testing.T) {
	encoded, err := json.Marshal(ImplementationReportOutputSchema())
	if err != nil { t.Fatal(err) }
	for _, field := range []string{"report", "schema_version", "request_id", "input_sha256", "coverage", "changes", "references", "history", "additionalProperties"} {
		if !strings.Contains(string(encoded), `"`+field+`"`) { t.Fatalf("report output schema lacks %s", field) }
	}
}
