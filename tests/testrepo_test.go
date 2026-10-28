package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nuget_ku_2_2/src/graph"
	"nuget_ku_2_2/src/testrepo"
)

func TestTestRepositoryCases(t *testing.T) {
	source, err := testrepo.Load("data/graph.txt", "1.0")
	if err != nil || !source.Has("A") {
		t.Fatalf("не удалось загрузить тестовый граф: %v", err)
	}
	root := graph.Node{ID: "A", Version: "1.0"}
	result, err := graph.Build(context.Background(), source, root, 3, "")
	if err != nil || len(result.Nodes) != 5 || len(result.Edges) != 6 {
		t.Fatalf("неверный тестовый граф: %+v, %v", result, err)
	}
	filtered, err := graph.Build(context.Background(), source, root, 3, "D")
	if err != nil || len(filtered.Nodes) != 4 || len(filtered.Edges) != 3 {
		t.Fatalf("фильтр не применён: %+v, %v", filtered, err)
	}
}

func TestTestRepositoryRejectsInvalidLines(t *testing.T) {
	cases := []string{"A B\n", "A: bad\n", "A: B\nA: C\n", "a: B\n"}
	for _, content := range cases {
		path := filepath.Join(t.TempDir(), "graph.txt")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := testrepo.Load(path, "1.0")
		if err == nil || !strings.Contains(err.Error(), "строка") {
			t.Fatalf("неверная строка принята: %q, %v", content, err)
		}
	}
}
