package ai

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"time"
)

func unknownUsage(provider, model string, started, completed time.Time) *UsageEnvelope {
	unknownInt := func() UsageField[int64] { return UsageField[int64]{State: UsageFieldUnknown} }
	unknownNumber := func() UsageField[json.Number] { return UsageField[json.Number]{State: UsageFieldUnknown} }
	unknownString := func() UsageField[string] { return UsageField[string]{State: UsageFieldUnknown} }
	return &UsageEnvelope{
		Version: UsageEnvelopeVersion, Availability: UsageFieldUnknown,
		Provider: provider, Model: model, StartedAt: started, CompletedAt: completed,
		InputTokens: unknownInt(), OutputTokens: unknownInt(), TotalTokens: unknownInt(),
		CostAmount: unknownNumber(), CostCurrency: unknownString(),
	}
}

// usageFromRaw maps only fields present in the Provider response. It accepts
// common field names used by the supported Provider APIs while preserving the
// exact usage object as bounded raw evidence.
func usageFromRaw(provider, model string, started, completed time.Time, raw []byte) *UsageEnvelope {
	envelope := unknownUsage(provider, model, started, completed)
	usage, evidence, ok := findUsage(raw)
	if !ok {
		return envelope
	}
	envelope.Availability = UsageFieldKnown
	envelope.RawResponse = sanitizeRawUsage(evidence)
	PrepareUsageEvidence(envelope, completed)
	setTokenField(&envelope.InputTokens, firstField(usage, "input_tokens", "prompt_tokens", "promptTokenCount"))
	setTokenField(&envelope.OutputTokens, firstField(usage, "output_tokens", "completion_tokens", "candidatesTokenCount"))
	setTokenField(&envelope.TotalTokens, firstField(usage, "total_tokens", "totalTokenCount"))
	amount := firstField(usage, "cost_amount", "cost", "total_cost")
	currency := firstField(usage, "cost_currency", "currency")
	amountValue, amountState := parseCost(amount)
	currencyValue, currencyState := parseCurrency(currency)
	if amountState == UsageFieldKnown && currencyState != UsageFieldKnown {
		amountState = UsageFieldUnknown
	}
	if currencyState == UsageFieldKnown && amountState != UsageFieldKnown {
		currencyState = UsageFieldUnknown
	}
	if amountState != UsageFieldKnown {
		amountValue = nil
	}
	if currencyState != UsageFieldKnown {
		currencyValue = nil
	}
	envelope.CostAmount = UsageField[json.Number]{State: amountState, Value: amountValue}
	envelope.CostCurrency = UsageField[string]{State: currencyState, Value: currencyValue}
	return envelope
}

var rawUsageFieldAllowlist = map[string]struct{}{
	"input_tokens": {}, "prompt_tokens": {}, "promptTokenCount": {},
	"output_tokens": {}, "completion_tokens": {}, "candidatesTokenCount": {},
	"total_tokens": {}, "totalTokenCount": {},
	"cost_amount": {}, "cost": {}, "total_cost": {},
	"cost_currency": {}, "currency": {},
}

// sanitizeRawUsage keeps only recognized scalar usage fields and enforces a
// strict serialized size bound. Normalized field states preserve malformed or
// omitted values even when a raw value is too large to retain.
func sanitizeRawUsage(raw []byte) json.RawMessage {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil { return nil }
	clean := make(map[string]json.RawMessage)
	for name, value := range fields {
		if _, ok := rawUsageFieldAllowlist[name]; !ok || len(value) > 256 { continue }
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	var scalar any
	if decoder.Decode(&scalar) != nil { continue }
		switch scalar.(type) {
		case nil, bool, string, json.Number:
			clean[name] = append(json.RawMessage(nil), value...)
		}
	}
	if len(clean) == 0 { return nil }
	encoded, err := json.Marshal(clean)
	if err != nil { return nil }
	return encoded
}

func findUsage(raw []byte) (map[string]json.RawMessage, []byte, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return nil, nil, false
	}
	var search func(any) (map[string]json.RawMessage, []byte, bool)
	search = func(node any) (map[string]json.RawMessage, []byte, bool) {
		switch typed := node.(type) {
		case map[string]any:
			usageNode, exists := typed["last_token_usage"]
			if !exists {
				usageNode, exists = typed["usage"]
			}
			if !exists {
				usageNode, exists = typed["usageMetadata"]
			}
			if exists {
				encoded, err := json.Marshal(usageNode)
				if err == nil {
					var usage map[string]json.RawMessage
					if json.Unmarshal(encoded, &usage) == nil && usage != nil {
						return usage, encoded, true
					}
				}
			}
			for _, child := range typed {
				if usage, evidence, ok := search(child); ok {
					return usage, evidence, true
				}
			}
		case []any:
			for _, child := range typed {
				if usage, evidence, ok := search(child); ok {
					return usage, evidence, true
				}
			}
		}
		return nil, nil, false
	}
	if usage, evidence, ok := search(value); ok {
		return usage, evidence, true
	}
	// CLI providers emit one JSON object per line. Inspect each line without
	// retaining unrelated event or model-output content in the envelope.
	if bytes.Contains(raw, []byte("\n")) {
		for _, line := range bytes.Split(raw, []byte("\n")) {
			if usage, evidence, ok := findUsage(line); ok {
				return usage, evidence, true
			}
		}
	}
	return nil, nil, false
}

func firstField(fields map[string]json.RawMessage, names ...string) json.RawMessage {
	for _, name := range names {
		if value, ok := fields[name]; ok {
			return value
		}
	}
	return nil
}

func setTokenField(target *UsageField[int64], raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		target.State = UsageFieldInvalid
		return
	}
	value, err := number.Int64()
	if err != nil || value < 0 {
		target.State = UsageFieldInvalid
		return
	}
	target.State = UsageFieldKnown
	target.Value = &value
}

func parseCost(raw json.RawMessage) (*json.Number, UsageFieldState) {
	if len(raw) == 0 {
		return nil, UsageFieldUnknown
	}
	var value json.Number
	if json.Unmarshal(raw, &value) != nil {
		return nil, UsageFieldInvalid
	}
	rational, ok := new(big.Rat).SetString(value.String())
	if !ok || rational.Sign() < 0 {
		return nil, UsageFieldInvalid
	}
	return &value, UsageFieldKnown
}

func parseCurrency(raw json.RawMessage) (*string, UsageFieldState) {
	if len(raw) == 0 {
		return nil, UsageFieldUnknown
	}
	var value string
	if json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" {
		return nil, UsageFieldInvalid
	}
	value = strings.ToUpper(strings.TrimSpace(value))
	return &value, UsageFieldKnown
}
