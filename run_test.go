package flowcore

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

func TestComputeLayersSequentialDeps(t *testing.T) {
	w := New()
	w.Step("a", func(*Context) error { return nil })
	w.Step("b", func(*Context) error { return nil }, DependsOn("a"))
	w.Step("c", func(*Context) error { return nil }, DependsOn("b"))
	ls, err := computeLayers(w.steps)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"a"}, {"b"}, {"c"}}
	got := layerNames(ls)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("layers: want %v got %v", want, got)
	}
}

func TestRunRespectsDeps(t *testing.T) {
	var order []string
	var mu sync.Mutex
	w := New()
	w.Step("a", func(*Context) error {
		mu.Lock()
		order = append(order, "a")
		mu.Unlock()
		return nil
	})
	w.Step("b", func(*Context) error {
		mu.Lock()
		order = append(order, "b")
		mu.Unlock()
		return nil
	}, DependsOn("a"))
	w.Step("c", func(*Context) error {
		mu.Lock()
		order = append(order, "c")
		mu.Unlock()
		return nil
	}, DependsOn("b"))
	err := RunWithConfig(context.Background(), w, RunConfig{Logger: nil, Store: nil})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("order = %v", order)
	}
	// Sequential layers still run steps in parallel goroutines; order must respect dependencies (a before b before c).
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("order = %v", order)
	}
}

func layerNames(ls [][]*Step) [][]string {
	out := make([][]string, len(ls))
	for i, l := range ls {
		for _, s := range l {
			out[i] = append(out[i], s.Name)
		}
	}
	return out
}

func TestComputeLayersParallelIndependent(t *testing.T) {
	w := New()
	w.Step("x", func(*Context) error { return nil })
	w.Step("y", func(*Context) error { return nil })
	w.Step("z", func(*Context) error { return nil }, DependsOn("x", "y"))
	ls, err := computeLayers(w.steps)
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 2 {
		t.Fatalf("want 2 layers got %v", layerNames(ls))
	}
	if len(ls[0]) != 2 || len(ls[1]) != 1 {
		t.Fatalf("want [2,1] steps per layer got %v", layerNames(ls))
	}
}
