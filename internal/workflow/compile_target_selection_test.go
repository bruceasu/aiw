package workflow

import (
	"reflect"
	"testing"
)

func TestCompileTargetSelection(t *testing.T) {
	plan := CompilePlan{Available: true, Targets: []CompileTarget{
		{Name: "go", Kind: "go", PathOwners: []string{"backend", "shared"}},
		{Name: "web", Kind: "script", Script: "web/compile.sh", PathOwners: []string{"web", "shared"}},
		{Name: "python", Kind: "unavailable", PathOwners: []string{"python", "shared"}},
	}}
	for _, tc := range []struct { name string; paths []string; indexes []int }{
		{"one target", []string{"web/app.ts"}, []int{1}},
		{"multiple languages and duplicate paths", []string{"web/app.ts", "backend/main.go", "web/style.css"}, []int{0, 1}},
		{"shared ownership", []string{"shared/types.json"}, []int{0, 1, 2}},
		{"unmapped shared config", []string{"go.work"}, []int{0, 1, 2}},
		{"known and unknown", []string{"web/app.ts", "README.md"}, []int{0, 1, 2}},
		{"no paths", nil, []int{0, 1, 2}},
		{"directory boundary", []string{"web-other/app.ts"}, []int{0, 1, 2}},
		{"unavailable target", []string{"python/app.py"}, []int{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var want []CompileTarget
			for _, index := range tc.indexes { want = append(want, plan.Targets[index]) }
			if got := SelectCompileTargets(plan, tc.paths); !reflect.DeepEqual(got, want) { t.Fatalf("targets = %+v, want %+v", got, want) }
		})
	}
	plan.Targets[2].PathOwners = []string{"."}
	if got := SelectCompileTargets(plan, []string{"web/app.ts"}); !reflect.DeepEqual(got, plan.Targets) { t.Fatalf("repository-wide owner must select all: %+v", got) }
}
