package requirement

import "strings"

// PlanReview is a model's evidence-linked reading of the captured Plan. It is
// optional for old coverage records; absence can never recommend approval.
type PlanReview struct {
	Sections []PlanSection `json:"sections"`
	DeferredDesign []DeferredDesign `json:"deferred_design"`
}

type PlanSection struct {
	Section string `json:"section"`
	Explanation string `json:"explanation"`
	Sources []CoverageReference `json:"sources"`
}

// DeferredDesign quotes the entire proposed postponement and its reason.
// There is no model-controlled confirmed flag: the host supplies confirmation.
type DeferredDesign struct {
	Question string `json:"question"`
	Reason string `json:"reason"`
	Decision CoverageReference `json:"decision"`
}

// ReadinessReport separates conversational advice from lifecycle authority.
// Even Ready requires a separate displayed human approval checkpoint.
type ReadinessReport struct {
	Ready bool `json:"ready"`
	Confirmed []CoverageReference `json:"confirmed"`
	BusinessBlockers []string `json:"business_blockers"`
	PlanGaps []string `json:"plan_gaps"`
	Deferred []DeferredDesign `json:"deferred"`
	UnconfirmedDeferrals []DeferredDesign `json:"unconfirmed_deferrals"`
}

var planSections = [...]string{"facts", "assumptions", "goals", "scope", "non_goals", "rules", "acceptance", "sources", "remaining_decisions"}

// AssessReadiness reparses raw coverage rather than trusting saved report flags.
// Quote validation is not proof of semantic completeness; the human reviews the
// evidence and remains responsible for approval and design/business boundaries.
func AssessReadiness(raw string, snapshot ConversationContext, facts ConfirmedFacts) ReadinessReport {
	report := ReadinessReport{Confirmed: facts.Current(snapshot)}
	coverage, err := ParseCoverage(raw, snapshot, report.Confirmed)
	if err != nil {
		report.BusinessBlockers = append(report.BusinessBlockers, err.Error())
		return report
	}
	trusted := make(map[CoverageReference]bool)
	for _, ref := range report.Confirmed { trusted[ref] = true }
	for _, item := range coverage.Assessment.Items {
		if item.Status == "needs_input" || item.Status == "conflict" {
			report.BusinessBlockers = append(report.BusinessBlockers, item.Dimension+": "+item.Question)
		}
		if item.Status == "not_applicable" {
			confirmed := false
			for _, ref := range item.Sources { confirmed = confirmed || trusted[ref] }
			if !confirmed { report.BusinessBlockers = append(report.BusinessBlockers, item.Dimension+": applicability needs human confirmation") }
		}
	}
	var plan *ConversationSource
	for i := range snapshot.Sources {
		if snapshot.Sources[i].Kind == "requirement-plan" && snapshot.Sources[i].Status == "loaded" { plan = &snapshot.Sources[i]; break }
	}
	if plan == nil {
		report.PlanGaps = append(report.PlanGaps, "No captured Requirement Plan; a draft may still be saved.")
		return report
	}
	for _, marker := range []string{"%% NEEDS_INPUT", "TODO", "TBD"} {
		if strings.Contains(strings.ToUpper(plan.Content), marker) {
			report.PlanGaps = append(report.PlanGaps, "Plan still contains an unresolved placeholder: "+marker)
		}
	}
	review := coverage.Assessment.PlanReview
	if review == nil {
		report.PlanGaps = append(report.PlanGaps, "No evidence-linked Plan review; reassess the captured content.")
		return report
	}
	validPlanRef := func(ref CoverageReference) bool {
		return ref.Source == plan.Path && snapshot.validCoverageReference(ref) && substantivePlanQuote(ref.Quote)
	}
	seen := make(map[string]bool)
	for _, section := range review.Sections {
		known := false
		for _, key := range planSections { known = known || section.Section == key }
		if !known || seen[section.Section] {
			report.PlanGaps = append(report.PlanGaps, "Unknown or duplicate Plan section: "+section.Section)
			continue
		}
		seen[section.Section] = true
		valid := strings.TrimSpace(section.Explanation) != "" && len(section.Sources) > 0 && len(section.Sources) <= 8
		confirmed := false
		for _, ref := range section.Sources {
			valid = valid && validPlanRef(ref)
			confirmed = confirmed || trusted[ref]
		}
		if !valid { report.PlanGaps = append(report.PlanGaps, section.Section+": missing valid Plan body evidence") }
		switch section.Section {
		case "facts", "goals", "scope", "rules", "acceptance":
			if !confirmed { report.PlanGaps = append(report.PlanGaps, section.Section+": Plan statement needs explicit confirmation") }
		}
	}
	for _, key := range planSections {
		if !seen[key] { report.PlanGaps = append(report.PlanGaps, "Missing Plan content: "+key) }
	}
	for _, item := range review.DeferredDesign {
		valid := strings.TrimSpace(item.Question) != "" && strings.TrimSpace(item.Reason) != "" && validPlanRef(item.Decision) &&
			strings.Contains(item.Decision.Quote, item.Question) && strings.Contains(item.Decision.Quote, item.Reason)
		if !valid {
			report.PlanGaps = append(report.PlanGaps, "Invalid design postponement evidence: "+item.Question)
		} else if !trusted[item.Decision] {
			report.UnconfirmedDeferrals = append(report.UnconfirmedDeferrals, item)
		} else {
			report.Deferred = append(report.Deferred, item)
		}
	}
	report.Ready = snapshot.Requirement != nil && len(report.BusinessBlockers) == 0 && len(report.PlanGaps) == 0 && len(report.UnconfirmedDeferrals) == 0
	return report
}

// Reject headings and obvious placeholders. Semantic relevance is assessed by
// the model and human, not by guessing meaning from Markdown headings.
func substantivePlanQuote(quote string) bool {
	upper := strings.ToUpper(quote)
	for _, marker := range []string{"%% NEEDS_INPUT", "TODO", "TBD"} {
		if strings.Contains(upper, marker) { return false }
	}
	for _, line := range strings.Split(quote, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && strings.Trim(line, "-* _|:") != "" { return true }
	}
	return false
}
