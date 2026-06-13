package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiCandidatesSchema(t *testing.T) {
	schema := geminiCandidatesSchema()
	if schema["type"] != "object" {
		t.Fatalf("expected object type, got %v", schema["type"])
	}
	properties := schema["properties"].(map[string]any)
	if _, ok := properties["candidates"]; !ok {
		t.Fatal("expected candidates property")
	}
}

func TestParseGeminiResponse(t *testing.T) {
	raw := `{
		"candidates": [
			{
				"content": {
					"parts": [
						{
							"text": "{\"candidates\":[{\"type\":\"feat\",\"scope\":\"api\",\"subject\":\"add gemini support\",\"body\":\"\",\"breaking\":\"\",\"footer\":\"\"}]}"
						}
					]
				},
				"finishReason": "STOP"
			}
		]
	}`

	var resp geminiResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal gemini response: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		t.Fatal("expected content in response")
	}

	content := resp.Candidates[0].Content.Parts[0].Text
	if !strings.Contains(content, "add gemini support") {
		t.Fatalf("unexpected content: %s", content)
	}
}
