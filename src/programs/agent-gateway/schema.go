package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"
)

const maxSchemaBytes = 16 << 10
const maxSchemaDepth = 16
const maxSchemaNodes = 512

type schemaNode struct {
	typeName string
	properties map[string]*schemaNode
	required map[string]bool
	items *schemaNode
	additional bool
	minimum *float64
	maximum *float64
	minLength *int
	maxLength *int
	minItems *int
	maxItems *int
	enum []any
	constant any
	hasConstant bool
}

func validateFunctionSchema(data []byte) error {
	if len(data) == 0 || len(data) > maxSchemaBytes { return errors.New("schema byte limit exceeded") }
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data)); decoder.UseNumber()
	if decoder.Decode(&raw) != nil { return errors.New("invalid schema JSON") }
	count := 0
	root, err := parseSchema(raw, 0, &count)
	if err != nil || root.typeName != "object" { return errors.New("unsupported function schema") }
	return nil
}

func parseSchema(raw any, depth int, count *int) (*schemaNode, error) {
	*count++
	if depth > maxSchemaDepth || *count > maxSchemaNodes { return nil, errors.New("schema complexity limit exceeded") }
	object, ok := raw.(map[string]any); if !ok { return nil, errors.New("schema node must be an object") }
	node := &schemaNode{properties: map[string]*schemaNode{}, required: map[string]bool{}}
	seenProperties, seenRequired, seenAdditional, seenItems := false, false, false, false
	seenNumberBounds, seenStringBounds, seenArrayBounds := false, false, false
	for key, value := range object {
		switch key {
		case "type":
			node.typeName, ok = value.(string); if !ok { return nil, errors.New("invalid schema type") }
			if node.typeName != "object" && node.typeName != "array" && node.typeName != "string" && node.typeName != "number" && node.typeName != "integer" && node.typeName != "boolean" && node.typeName != "null" { return nil, errors.New("unsupported schema type") }
		case "properties":
			seenProperties = true
			properties, valid := value.(map[string]any); if !valid || len(properties) > maxSchemaNodes { return nil, errors.New("invalid schema properties") }
			for name, child := range properties { parsed, err := parseSchema(child, depth+1, count); if err != nil { return nil, err }; node.properties[name] = parsed }
		case "required":
			seenRequired = true
			items, valid := value.([]any); if !valid { return nil, errors.New("invalid schema required list") }
			for _, item := range items { name, valid := item.(string); if !valid || node.required[name] { return nil, errors.New("invalid schema required name") }; node.required[name] = true }
		case "additionalProperties":
			seenAdditional = true
			allow, valid := value.(bool); if !valid || allow { return nil, errors.New("additionalProperties must be false") }; node.additional = false
		case "items":
			seenItems = true
			parsed, err := parseSchema(value, depth+1, count); if err != nil { return nil, err }; node.items = parsed
		case "minimum", "maximum":
			seenNumberBounds = true
			number, valid := schemaNumber(value); if !valid { return nil, errors.New("invalid numeric schema bound") }
			if key == "minimum" { node.minimum = &number } else { node.maximum = &number }
		case "minLength", "maxLength", "minItems", "maxItems":
			number, valid := schemaInteger(value); if !valid || number < 0 || number > maxText { return nil, errors.New("invalid schema size bound") }
		switch key { case "minLength":node.minLength=&number;seenStringBounds=true;case "maxLength":node.maxLength=&number;seenStringBounds=true;case "minItems":node.minItems=&number;seenArrayBounds=true;case "maxItems":node.maxItems=&number;seenArrayBounds=true }
		case "enum":
			values, valid := value.([]any); if !valid || len(values) == 0 || len(values) > 128 { return nil, errors.New("invalid schema enum") }; node.enum = values
		case "const":node.constant=value;node.hasConstant=true
		default:
			return nil, errors.New("unsupported schema keyword")
		}
	}
	if node.typeName == "" { return nil, errors.New("schema type is required") }
	for name := range node.required { if _, exists := node.properties[name]; !exists { return nil, errors.New("required property is undeclared") } }
	if node.typeName == "object" && !seenAdditional || node.typeName != "object" && (seenAdditional || seenProperties || seenRequired) { return nil, errors.New("object keywords require an object schema") }
	if node.typeName == "array" && (!seenItems || node.items == nil) || node.typeName != "array" && (seenItems || seenArrayBounds) { return nil, errors.New("array keywords require an array schema") }
	if node.typeName != "string" && seenStringBounds || node.typeName != "number" && node.typeName != "integer" && seenNumberBounds { return nil, errors.New("schema bounds do not match schema type") }
	return node, nil
}

func schemaNumber(value any) (float64, bool) {
	number, ok := value.(json.Number); if !ok { return 0, false }; parsed, err := number.Float64(); return parsed, err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed)
}

func schemaInteger(value any) (int, bool) {
	number, ok := value.(json.Number); if !ok { return 0, false }; parsed, err := number.Int64(); if err != nil || parsed > int64(maxText) { return 0, false }; return int(parsed), true
}

func validateSchemaValue(schema []byte, value []byte) bool {
	var rawSchema any
	schemaDecoder := json.NewDecoder(bytes.NewReader(schema)); schemaDecoder.UseNumber()
	if schemaDecoder.Decode(&rawSchema) != nil { return false }
	count := 0; parsed, err := parseSchema(rawSchema, 0, &count); if err != nil { return false }
	var data any
	valueDecoder := json.NewDecoder(bytes.NewReader(value)); valueDecoder.UseNumber()
	if valueDecoder.Decode(&data) != nil { return false }
	return validateValue(parsed, data)
}

func validateValue(schema *schemaNode, value any) bool {
	switch schema.typeName {
	case "object":
		object, ok := value.(map[string]any); if !ok { return false }
		for key := range schema.required { if _, exists := object[key]; !exists { return false } }
		for key, child := range object { property, exists := schema.properties[key]; if !exists || !validateValue(property, child) { return false } }
	case "array":
		array, ok := value.([]any); if !ok || schema.minItems != nil && len(array) < *schema.minItems || schema.maxItems != nil && len(array) > *schema.maxItems { return false }
		for _, item := range array { if !validateValue(schema.items, item) { return false } }
	case "string":
		text, ok := value.(string); if !ok { return false }; length := utf8.RuneCountInString(text)
		if schema.minLength != nil && length < *schema.minLength || schema.maxLength != nil && length > *schema.maxLength { return false }
	case "number", "integer":
		number, ok := value.(json.Number); if !ok { return false }; parsed, err := number.Float64(); if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || schema.typeName == "integer" && math.Trunc(parsed) != parsed { return false }
		if schema.minimum != nil && parsed < *schema.minimum || schema.maximum != nil && parsed > *schema.maximum { return false }
	case "boolean":
		if _, ok := value.(bool); !ok { return false }
	case "null":
		if value != nil { return false }
	}
	if schema.hasConstant && !reflect.DeepEqual(schema.constant, value) { return false }
	if len(schema.enum) > 0 { found := false; for _, candidate := range schema.enum { if reflect.DeepEqual(candidate, value) { found = true; break } }; if !found { return false } }
	return true
}

func validateToolArguments(tool FunctionTool, arguments string) bool {
	if !validJSONObject([]byte(arguments)) { return false }
	if tool.Strict == nil || !*tool.Strict { return true }
	return validateValueFromBytes(tool.Parameters, []byte(arguments))
}

func validateValueFromBytes(schema, value []byte) bool { return validateSchemaValue(schema, value) }

func supportedSchemaKeyword(key string) bool {
	return strings.Contains(" type properties required additionalProperties items enum const minimum maximum minLength maxLength minItems maxItems ", " "+key+" ")
}
