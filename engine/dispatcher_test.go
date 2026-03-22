package engine

import (
	"testing"

	"github.com/hghukasyan/flowcore"
)

func TestPlanParallelMatchesWorkflowLayers(t *testing.T) {
	w := flowcore.New()
	w.Step("a", func(*flowcore.Context) error { return nil })
	w.Step("b", func(*flowcore.Context) error { return nil }, flowcore.DependsOn("a"))
	got, err := PlanParallel(w)
	if err != nil {
		t.Fatal(err)
	}
	want, err := w.ExecutionLayers()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d", len(got), len(want))
	}
	for i := range got {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("layer %d len mismatch", i)
		}
		for j := range got[i] {
			if got[i][j].Name != want[i][j].Name {
				t.Fatalf("layer %d step %d: %s vs %s", i, j, got[i][j].Name, want[i][j].Name)
			}
		}
	}
}
