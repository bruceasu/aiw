package workflow

import (
	"errors"
	"testing"
)

func TestActorWriteScopesRejectUnauthorizedPaths(t *testing.T) {
	contract := ImplementationContract{
		Status:                 ImplementationContractReady,
		AllowedProductionPaths: []string{"internal/workflow/**"},
		AllowedTestPaths:       []string{"internal/workflow/*_test.go"},
	}
	if err := ValidateCoderChangedPaths(contract, []string{"internal/workflow/compile.go"}); err != nil {
		t.Fatalf("allowed Coder path rejected: %v", err)
	}
	if err := ValidateTesterChangedPaths(contract, []string{"internal/workflow/compile_test.go"}); err != nil {
		t.Fatalf("allowed Tester path rejected: %v", err)
	}

	for _, check := range []struct {
		name string
		run  func() error
		role ActorKind
	}{
		{name: "coder", run: func() error { return ValidateCoderChangedPaths(contract, []string{"cmd/aiw/main.go"}) }, role: ActorCoder},
		{name: "tester", run: func() error { return ValidateTesterChangedPaths(contract, []string{"internal/workflow/compile.go"}) }, role: ActorTester},
	} {
		t.Run(check.name, func(t *testing.T) {
			var violation *ActorWriteScopeViolation
			if err := check.run(); !errors.As(err, &violation) {
				t.Fatalf("error = %v, want actor write scope violation", err)
			}
			if violation.Actor != check.role {
				t.Fatalf("violation actor = %q, want %q", violation.Actor, check.role)
			}
		})
	}
}
