package openspecgen

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"aiw/internal/requirement"
)

const qualityGuidance = "\nQuality evidence: Keep each business statement from the approved Plan (outside OpenSpec Targets) and each active confirmed quote in the mapped requirement body or scenario. You may add detail, but keep the source sentence as evidence. Do not use headings, comments, code blocks or sibling scenarios as evidence. Map every scenario and numbered task. Use the Plan path or active fact ID as source_id; captured but unconfirmed text is not approved scope. Use kind engineering only for a non-blocking engineering decision. Record its detail in design.md with %% and add TODO and Verification sections to tasks.md. Checks preserve source statements; they do not prove semantic correctness, design readiness or implementation acceptance.\n"

// Quality is deliberately conservative: it verifies retained source statements,
// not arbitrary semantic equivalence. Paraphrases without evidence fail closed.
// Freshness and approval are checked by the caller against trusted live input.
func candidateQuality(request requirement.GenerationRequest, candidate requirement.GenerationCandidate) (requirement.GenerationReport, error) {
	report := requirement.GenerationReport{CoverageMethod: "source-statement-preservation-v1", DesignReadiness: "not-assessed", ImplementationAcceptance: "not-assessed"}
	issue := func(source, kind, detail string) {
		report.Issues = append(report.Issues, requirement.GenerationIssue{SourceID: source, Kind: kind, Detail: detail, Blocking: true})
	}
	if _, err := RenderCandidate(request, candidate); err != nil {
		issue("", "structure", err.Error())
		return report, err
	}
	report.StructureValid = true
	files := map[string]string{}
	for _, artifact := range candidate.Artifacts { files[artifact.Path] = qualityText(artifact.Content) }
	for _, required := range []struct { path, heading string }{
		{"proposal.md", "## Why"}, {"proposal.md", "## What Changes"}, {"proposal.md", "## Capabilities"},
		{"design.md", "## Context"}, {"design.md", "## Goals / Non-Goals"}, {"design.md", "## Decisions"},
	} {
		if section(files[required.path], required.heading, "## ") == "" { issue("", "empty-section", required.path+": "+required.heading) }
	}
	active := map[string]bool{}
	for _, id := range request.ActiveFacts { active[id] = true }
	type sourceUnit struct { id, text string }
	var units []sourceUnit
	var retired []string
	for _, fact := range request.Confirmation.Facts {
		if active[fact.ID] {
			units = append(units, sourceUnit{fact.ID, fact.Reference.Quote})
		} else {
			retired = append(retired, fact.Reference.Quote)
		}
	}
	for _, source := range request.Sources {
		if source.Kind != "requirement-plan" { continue }
		body := source.Content
		// Only an explicitly confirmed replacement removes old approved text.
		for _, old := range retired { body = strings.ReplaceAll(body, old, "") }
		inTargets := false
		for _, line := range strings.Split(qualityText(body), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				if strings.HasPrefix(line, "## ") { inTargets = line == "## OpenSpec Targets" }
				continue
			}
			if inTargets { continue }
			line = qualityListPrefix.ReplaceAllString(line, "")
			if qualityNormalize(line) != "" { units = append(units, sourceUnit{source.Path, line}) }
		}
	}
	if len(units) == 0 { issue("", "missing-approved-scope", "No approved business statements are available") }
	covered := make([]bool, len(units))
	validMappings := map[string]bool{}
	validTasks := map[string]bool{}
	approved := ""
	for _, unit := range units { approved += "\n" + unit.text }
	for _, mapping := range candidate.Coverage {
		body := section(files[mapping.Path], "### Requirement: "+mapping.Requirement, "### ")
		scenario := section(body, "#### Scenario: "+mapping.Scenario, "#### ")
		intro := strings.SplitN(body, "#### ", 2)[0]
		if !strings.Contains(scenario, "**WHEN**") || !strings.Contains(scenario, "**THEN**") {
			issue(mapping.SourceID, "scenario", "Mapped scenario needs WHEN and THEN: "+mapping.Scenario)
			continue
		}
		matched := false
		for index, unit := range units {
			if unit.id != mapping.SourceID { continue }
			// Sibling scenarios and headings cannot provide evidence for this one.
			if strings.Contains(qualityNormalize(intro+"\n"+scenario), qualityNormalize(unit.text)) && qualityRelated(unit.text, scenario) {
				covered[index], matched = true, true
			}
		}
		if !matched {
			issue(mapping.SourceID, "unsupported-mapping", "Mapped body does not retain an approved statement: "+mapping.Path+" / "+mapping.Scenario)
			continue
		}
		report.VerifiedCoverage = append(report.VerifiedCoverage, mapping)
		validMappings[mapping.Path+"\n"+mapping.Requirement+"\n"+mapping.Scenario] = true
		if mapping.Task != "" { validTasks[mapping.Task] = true }
	}
	for index, unit := range units {
		if !covered[index] { issue(unit.id, "missing-source-statement", unit.text) }
	}
	for _, artifact := range candidate.Artifacts {
		body := files[artifact.Path]
		if !qualityRelated(approved, body) { issue("", "off-topic", "No business evidence in "+artifact.Path) }
		for _, placeholder := range []string{"The system MUST generate the required OpenSpec artifacts from an approved Requirement", "it produces proposal, design, delta spec, and tasks files", "[insert", "<business rule>", "lorem ipsum"} {
			if strings.Contains(strings.ToLower(body), strings.ToLower(placeholder)) { issue("", "generic-template", "Placeholder or generator template in "+artifact.Path) }
		}
		if strings.HasPrefix(artifact.Path, "specs/") {
			current := ""
			checkRequirement := func() {
				if current == "" { return }
				text := section(body, "### Requirement: "+current, "### ")
				intro := strings.SplitN(text, "#### ", 2)[0]
				if !strings.Contains(text, "#### Scenario: ") || (!strings.Contains(intro, " MUST ") && !strings.Contains(intro, " SHALL ")) {
					issue("", "empty-requirement", artifact.Path+": "+current)
				}
			}
			seen := map[string]bool{}
			for _, line := range strings.Split(body, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "## ") { checkRequirement(); current = "" }
				if strings.HasPrefix(line, "### Requirement: ") {
					checkRequirement()
					current = strings.TrimPrefix(line, "### Requirement: ")
					if seen["requirement\n"+current] { issue("", "ambiguous-heading", artifact.Path+": "+current) }
					seen["requirement\n"+current] = true
				}
				if strings.HasPrefix(line, "#### Scenario: ") {
					key := artifact.Path+"\n"+current+"\n"+strings.TrimPrefix(line, "#### Scenario: ")
					if seen[key] { issue("", "ambiguous-heading", key) }
					seen[key] = true
					if !validMappings[key] { issue("", "unmapped-scenario", key) }
				}
			}
			checkRequirement()
			for _, old := range retired {
				if qualityNormalize(old) != "" && !strings.Contains(qualityNormalize(approved), qualityNormalize(old)) && strings.Contains(qualityNormalize(body), qualityNormalize(old)) {
					issue("", "superseded-source", "Retired statement remains in "+artifact.Path)
				}
			}
		}
	}
	for _, match := range qualityTask.FindAllStringSubmatch(files["tasks.md"], -1) {
		if !validTasks[match[1]] || !qualityRelated(approved, match[2]) { issue("", "unmapped-task", "Task lacks supported business mapping: "+match[1]) }
	}
	for _, unresolved := range candidate.Unresolved {
		// Only explicit engineering deferrals may remain non-blocking.
		if unresolved.Kind != "engineering" || strings.TrimSpace(unresolved.Detail) == "" { unresolved.Blocking = true }
		report.Issues = append(report.Issues, unresolved)
		if !unresolved.Blocking {
			if !strings.Contains(files["design.md"], "%%") || !strings.Contains(files["design.md"], unresolved.Detail) ||
				section(files["tasks.md"], "## TODO", "## ") == "" || section(files["tasks.md"], "## Verification", "## ") == "" {
				issue(unresolved.SourceID, "undocumented-deferral", "Engineering deferral requires design %% and task TODO/Verification: "+unresolved.Detail)
			}
		}
	}
	report.CoverageValid = true
	for _, finding := range report.Issues { if finding.Blocking { report.CoverageValid = false } }
	if !report.CoverageValid { return report, fmt.Errorf("candidate quality rejected; inspect coverage issues in result.json") }
	return report, nil
}

var qualityListPrefix = regexp.MustCompile(`^(?:[-*+]\s+(?:\[[ xX]\]\s+)?|[0-9]+(?:\.[0-9]+)*[.)]?\s+)`)
var qualityTask = regexp.MustCompile(`(?m)^- \[[ xX]\] ([0-9]+(?:\.[0-9]+)+)\s+([^\r\n]+)$`)

// Remove comments and fenced examples so quoted examples cannot count as rules.
func qualityText(text string) string {
	var lines []string
	fence, comment := "", false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "<!--") { comment = true }
		if comment { if strings.Contains(trimmed, "-->") { comment = false }; continue }
		if fence != "" { if strings.HasPrefix(trimmed, fence) { fence = "" }; continue }
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") { fence = trimmed[:3]; continue }
		if strings.HasPrefix(trimmed, ">") { continue }
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func qualityNormalize(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("<>+=-%", r) { return r }; return ' '
	}, text))), " ")
}

func qualityRelated(source, body string) bool {
	words, bodyWords := qualityWords(source), qualityWords(body)
	count := 0
	for word := range words { if bodyWords[word] { count++ } }
	return count >= 2 || (len(words) == 1 && count == 1)
}

func qualityWords(text string) map[string]bool {
	words := map[string]bool{}
	for _, word := range strings.Fields(qualityNormalize(text)) {
		runes := []rune(word)
		// Chinese text has no space-delimited words. Use adjacent Han pairs for
		// the relatedness hint only; source-statement matching remains exact.
		for index := 1; index < len(runes); index++ {
			if unicode.Is(unicode.Han, runes[index-1]) && unicode.Is(unicode.Han, runes[index]) { words[string(runes[index-1:index+1])] = true }
		}
		if len(runes) >= 3 && !strings.Contains(" the and must shall when then system with from this that ", " "+word+" ") {
			if len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") { word = strings.TrimSuffix(word, "s") }
			words[word] = true
		}
	}
	return words
}
