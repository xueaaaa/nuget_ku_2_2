package graph

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Node struct {
	ID      string
	Version string
}

type Edge struct {
	From  Node
	To    Node
	Cycle bool
}

type Graph struct {
	Root  Node
	Nodes []Node
	Edges []Edge
}

type Source interface {
	Dependencies(context.Context, Node) ([]Node, error)
}

type builder struct {
	source     Source
	graph      Graph
	maxDepth   int
	filter     string
	nodes      map[Node]bool
	edges      map[[2]Node]int
	active     map[Node]bool
	expandedAt map[Node]int
}

func Build(ctx context.Context, source Source, root Node,
	maxDepth int, filter string) (Graph, error) {
	if maxDepth < 0 {
		return Graph{}, errors.New("максимальная глубина не может быть отрицательной")
	}
	filter = strings.ToLower(filter)
	if filter != "" && strings.Contains(strings.ToLower(root.ID), filter) {
		return Graph{}, errors.New("исходный пакет исключён фильтром")
	}
	b := builder{
		source: source, graph: Graph{Root: root}, maxDepth: maxDepth,
		filter: filter, nodes: make(map[Node]bool),
		edges: make(map[[2]Node]int), active: make(map[Node]bool),
		expandedAt: make(map[Node]int),
	}
	b.addNode(root)
	if err := b.walk(ctx, root, 0); err != nil {
		return Graph{}, err
	}
	return b.graph, nil
}

func (b *builder) walk(ctx context.Context, node Node, depth int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if depth >= b.maxDepth {
		return nil
	}
	if b.alreadyExpanded(node, depth) {
		return nil
	}
	b.expandedAt[node] = depth
	b.active[node] = true
	defer delete(b.active, node)
	children, err := b.source.Dependencies(ctx, node)
	if err != nil {
		return fmt.Errorf("зависимости %s: %w", label(node), err)
	}
	for _, child := range children {
		if b.excluded(child.ID) {
			continue
		}
		b.addNode(child)
		b.addEdge(node, child, b.active[child])
		if !b.active[child] {
			if err := b.walk(ctx, child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *builder) alreadyExpanded(node Node, depth int) bool {
	previous, ok := b.expandedAt[node]
	return ok && previous <= depth
}

func (b *builder) excluded(id string) bool {
	return b.filter != "" && strings.Contains(strings.ToLower(id), b.filter)
}

func (b *builder) addNode(node Node) {
	if !b.nodes[node] {
		b.nodes[node] = true
		b.graph.Nodes = append(b.graph.Nodes, node)
	}
}

func (b *builder) addEdge(from Node, to Node, cycle bool) {
	key := [2]Node{from, to}
	if index, ok := b.edges[key]; ok {
		b.graph.Edges[index].Cycle = b.graph.Edges[index].Cycle || cycle
		return
	}
	b.edges[key] = len(b.graph.Edges)
	b.graph.Edges = append(b.graph.Edges, Edge{From: from, To: to, Cycle: cycle})
}

func Print(output io.Writer, result Graph) {
	fmt.Fprintf(output, "Граф зависимостей %s:\n", label(result.Root))
	fmt.Fprintln(output, "Пакеты:")
	for _, node := range result.Nodes {
		fmt.Fprintf(output, "  %s\n", label(node))
	}
	fmt.Fprintln(output, "Связи:")
	if len(result.Edges) == 0 {
		fmt.Fprintln(output, "  отсутствуют")
	}
	for _, edge := range result.Edges {
		mark := ""
		if edge.Cycle {
			mark = " [цикл]"
		}
		fmt.Fprintf(output, "  %s -> %s%s\n", label(edge.From), label(edge.To), mark)
	}
}

func label(node Node) string {
	if node.Version == "" {
		return node.ID
	}
	return node.ID + "@" + node.Version
}
