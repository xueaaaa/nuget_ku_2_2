package tests

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"nuget_ku_2_2/src/graph"
)

type graphSource map[graph.Node][]graph.Node

func (source graphSource) Dependencies(_ context.Context, node graph.Node) ([]graph.Node, error) {
	return source[node], nil
}

func cycleSource() (graph.Node, graphSource) {
	root := graph.Node{ID: "A"}
	source := graphSource{
		root:      {{ID: "B"}, {ID: "C"}},
		{ID: "B"}: {{ID: "D"}},
		{ID: "C"}: {{ID: "D"}},
		{ID: "D"}: {root},
	}
	return root, source
}

func TestGraphCycle(t *testing.T) {
	root, source := cycleSource()
	result, err := graph.Build(context.Background(), source, root, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 4 || len(result.Edges) != 5 {
		t.Fatalf("неверный граф: %+v", result)
	}
	cycleCount := 0
	for _, edge := range result.Edges {
		if edge.Cycle {
			cycleCount++
			if edge.From.ID != "D" || edge.To.ID != "A" {
				t.Fatalf("неверное циклическое ребро: %+v", edge)
			}
		}
	}
	if cycleCount != 1 {
		t.Fatalf("ожидался один цикл, получено %d", cycleCount)
	}
	var output bytes.Buffer
	graph.Print(&output, result)
	if !strings.Contains(output.String(), "D -> A [цикл]") {
		t.Fatalf("цикл не показан: %q", output.String())
	}
}

func TestGraphDepthAndFilter(t *testing.T) {
	root, source := cycleSource()
	shallow, err := graph.Build(context.Background(), source, root, 1, "")
	if err != nil || len(shallow.Edges) != 2 || len(shallow.Nodes) != 3 {
		t.Fatalf("неверное ограничение глубины: %+v, %v", shallow, err)
	}
	filtered, err := graph.Build(context.Background(), source, root, 3, "d")
	if err != nil || len(filtered.Nodes) != 3 || len(filtered.Edges) != 2 {
		t.Fatalf("неверная фильтрация: %+v, %v", filtered, err)
	}
	zero, err := graph.Build(context.Background(), source, root, 0, "")
	if err != nil || len(zero.Nodes) != 1 || len(zero.Edges) != 0 {
		t.Fatalf("неверная нулевая глубина: %+v, %v", zero, err)
	}
}

func TestGraphRejectsFilteredRoot(t *testing.T) {
	root, source := cycleSource()
	if _, err := graph.Build(context.Background(), source, root, 3, "a"); err == nil {
		t.Fatal("исходный пакет должен исключаться фильтром")
	}
}

func TestGraphRevisitsNodeAtShorterDepth(t *testing.T) {
	root := graph.Node{ID: "A"}
	source := graphSource{
		root:      {{ID: "X"}, {ID: "B"}},
		{ID: "X"}: {{ID: "B"}},
		{ID: "B"}: {{ID: "C"}},
		{ID: "C"}: {{ID: "D"}},
	}
	result, err := graph.Build(context.Background(), source, root, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		ids = append(ids, node.ID)
	}
	if !reflect.DeepEqual(ids, []string{"A", "X", "B", "C", "D"}) {
		t.Fatalf("узел на коротком пути не раскрыт: %v", ids)
	}
}
