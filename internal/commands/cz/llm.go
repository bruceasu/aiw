package cz

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

func ParseLLMCandidates(out string) ([]Draft, error) {
	if cands, ok := parseCandidatesJSON(strings.TrimSpace(out)); ok {
		return cands, nil
	}

	chunks := strings.Split(out, "```")
	for i := 1; i < len(chunks); i += 2 {
		block := strings.TrimSpace(chunks[i])
		block = strings.TrimPrefix(block, "json")
		if cands, ok := parseCandidatesJSON(strings.TrimSpace(block)); ok {
			return cands, nil
		}
	}

	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start >= 0 && end > start {
		candidate := out[start : end+1]
		if cands, ok := parseCandidatesJSON(candidate); ok {
			return cands, nil
		}
	}

	lines := strings.Split(out, "\n")
	var cands []Draft
	for _, ln := range lines {
		ln = cleanCandidateLine(ln)
		if ln == "" {
			continue
		}
		if d, ok := ParseConventionalHeader(ln); ok {
			cands = append(cands, d)
		}
	}
	if len(cands) == 0 {
		return nil, errors.New("invalid output")
	}
	return cands, nil
}

func parseCandidatesJSON(raw string) ([]Draft, bool) {
	if raw == "" {
		return nil, false
	}
	var resp LLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil && resp.Candidates != nil {
		return resp.Candidates, true
	}

	var list []Draft
	if err := json.Unmarshal([]byte(raw), &list); err == nil && list != nil {
		return list, true
	}
	return nil, false
}

func cleanCandidateLine(line string) string {
	v := strings.TrimSpace(line)
	v = strings.TrimPrefix(v, "-")
	v = strings.TrimPrefix(v, "*")
	v = strings.TrimSpace(v)

	if idx := strings.Index(v, ")"); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	if idx := strings.Index(v, "."); idx > 0 {
		prefix := strings.TrimSpace(v[:idx])
		if _, err := strconv.Atoi(prefix); err == nil {
			v = strings.TrimSpace(v[idx+1:])
		}
	}
	return v
}

func ParseConventionalHeader(line string) (Draft, bool) {
	i := strings.Index(line, ":")
	if i <= 0 || i+1 >= len(line) {
		return Draft{}, false
	}

	left := strings.TrimSpace(line[:i])
	subject := strings.TrimSpace(line[i+1:])
	if left == "" || subject == "" {
		return Draft{}, false
	}

	left = strings.TrimSuffix(left, "!")
	typePart := left
	scope := ""
	if l := strings.Index(left, "("); l >= 0 {
		r := strings.LastIndex(left, ")")
		if r <= l || r != len(left)-1 {
			return Draft{}, false
		}
		typePart = strings.TrimSpace(left[:l])
		scope = strings.TrimSpace(left[l+1 : r])
		if scope == "" {
			return Draft{}, false
		}
	}

	typePart = strings.ToLower(strings.TrimSpace(typePart))
	if typePart == "" || strings.ContainsAny(typePart, " `\t") {
		return Draft{}, false
	}
	if _, ok := conventionalTypeSet()[typePart]; !ok {
		return Draft{}, false
	}

	return Draft{Type: typePart, Scope: scope, Subject: subject}, true
}

func conventionalTypeSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, t := range DefaultConfig().Types {
		set[t.Value] = struct{}{}
	}
	return set
}
