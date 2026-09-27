package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestVerificationPlanDigestIgnoresCheckAndEnvironmentOrder(t *testing.T) {
	first := validVerificationPlan()
	first.Checks = append(first.Checks, VerificationCheck{
		CheckID:             "task-tests",
		Argv:                []string{"go", "test", "./internal/task"},
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

func TestSelectCompileAdapterPrefersCompileScriptThenGo(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.test/adapter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter, err := SelectCompileAdapter(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Kind != "go" || adapter.Name != "go-build" {
		t.Fatalf("adapter without script = %+v, want Go adapter", adapter)
	}

	if err := os.MkdirAll(filepath.Join(workspace, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "scripts", "compile.py"), []byte("raise SystemExit(0)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter, err = SelectCompileAdapter(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Kind != "script" || adapter.Name != "scripts/compile.py" {
		t.Fatalf("adapter with script = %+v, want compile script", adapter)
	}
}

func TestSelectAndResolveMavenCompileAdapter(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "pom.xml"), []byte("<project/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter, err := SelectCompileAdapter(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Kind != "maven" || adapter.Name != "maven-compile" || adapter.Command != "mvn" {
		t.Fatalf("Maven adapter = %+v", adapter)
	}
	if !reflect.DeepEqual(adapter.Args, []string{"-DskipTests", "compile"}) {
		t.Fatalf("Maven adapter args = %v", adapter.Args)
	}

	resolved, err := ResolveCompileTarget(workspace, CompileTarget{Name: adapter.Name, Kind: adapter.Kind, PathOwners: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != "maven" || resolved.Command != "mvn" || !reflect.DeepEqual(resolved.Args, adapter.Args) {
		t.Fatalf("resolved Maven adapter = %+v", resolved)
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
