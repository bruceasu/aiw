package workflow

import "testing"

func TestVerificationPlanDigestIgnoresCheckAndEnvironmentOrder(t *testing.T) {
	first := validVerificationPlan()
	first.Checks = append(first.Checks, VerificationCheck{
		CheckID:             "taskx-tests",
		Argv:                []string{"go", "test", "./internal/taskx"},
		WorkingDirectory:    ".",
		TimeoutSeconds:      60,
		ExpectedExitCode:    0,
		EvidenceDestination: "workflow",
		NetworkPolicy:       NetworkPolicyDeny,
		Profile:             FocusedTestProfile,
		AllowedEnvironment:  []string{"GOFLAGS", "GOCACHE"},
	})

	second := first
	second.Checks = append([]VerificationCheck(nil), first.Checks...)
	second.Checks[0].AllowedEnvironment = []string{"GOCACHE", "GOFLAGS"}
	second.Checks[0], second.Checks[1] = second.Checks[1], second.Checks[0]

	firstDigest, err := first.Digest()
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := second.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("digest changed with authored ordering: %s != %s", firstDigest, secondDigest)
	}
}

func TestVerificationPlanRejectsUnsafeChecks(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*VerificationPlan)
	}{
		{
			name: "shell syntax",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].Argv = []string{"go", "test", "./internal/workflow; whoami"}
			},
		},
		{
			name: "shell interpreter",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].Argv = []string{"bash", "-c", "go test ./internal/workflow"}
			},
		},
		{
			name: "custom script",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].Argv = []string{"./scripts/check.sh"}
			},
		},
		{
			name: "duplicate check ID",
			mutate: func(plan *VerificationPlan) {
				duplicate := plan.Checks[0]
				plan.Checks = append(plan.Checks, duplicate)
			},
		},
		{
			name: "absolute working directory",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].WorkingDirectory = "/tmp"
			},
		},
		{
			name: "escaping working directory",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].WorkingDirectory = "../outside"
			},
		},
		{
			name: "zero timeout",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].TimeoutSeconds = 0
			},
		},
		{
			name: "timeout over maximum",
			mutate: func(plan *VerificationPlan) {
				plan.Checks[0].TimeoutSeconds = MaxFocusedTestTimeoutSeconds + 1
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validVerificationPlan()
			test.mutate(&plan)
			if err := plan.Validate(); err == nil {
				t.Fatal("Validate() succeeded for an unsafe check")
			}
		})
	}
}

func validVerificationPlan() VerificationPlan {
	return VerificationPlan{
		SchemaVersion: VerificationPlanSchemaVersion,
		TaskID:        "aiw-safe-validation-recovery",
		Checks: []VerificationCheck{{
			CheckID:             "workflow-tests",
			Argv:                []string{"go", "test", "./internal/workflow"},
			WorkingDirectory:    ".",
			TimeoutSeconds:      60,
			ExpectedExitCode:    0,
			EvidenceDestination: "workflow",
			NetworkPolicy:       NetworkPolicyDeny,
			Profile:             FocusedTestProfile,
			AllowedEnvironment:  []string{"GOCACHE", "GOFLAGS"},
		}},
	}
}
