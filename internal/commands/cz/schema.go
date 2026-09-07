package cz

func czCandidatesSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     map[string]any{"type": "string"},
			"scope":    map[string]any{"type": "string"},
			"subject":  map[string]any{"type": "string"},
			"body":     map[string]any{"type": "string"},
			"breaking": map[string]any{"type": "string"},
			"footer":   map[string]any{"type": "string"},
		},
		"required":             []string{"type", "subject", "scope", "body", "breaking", "footer"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"candidates": map[string]any{"type": "array", "items": item},
		},
		"required":             []string{"candidates"},
		"additionalProperties": false,
	}
}
