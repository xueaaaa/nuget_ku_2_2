package graph

import (
	"fmt"
	"io"
	"strings"
)

type visitState uint8

const (
	unseen visitState = iota
	visiting
	finished
)

type orderWalker struct {
	children map[Node][]Node
	states   map[Node]visitState
	path     []Node
	order    []Node
}

func LoadOrder(result Graph) ([]Node, error) {
	walker := orderWalker{
		children: make(map[Node][]Node),
		states:   make(map[Node]visitState),
	}
	for _, edge := range result.Edges {
		walker.children[edge.From] = append(walker.children[edge.From], edge.To)
	}
	if err := walker.walk(result.Root); err != nil {
		return nil, err
	}
	return walker.order, nil
}

func (w *orderWalker) walk(node Node) error {
	switch w.states[node] {
	case visiting:
		return fmt.Errorf("невозможно определить порядок загрузки: цикл %s", w.cycle(node))
	case finished:
		return nil
	}
	w.states[node] = visiting
	w.path = append(w.path, node)
	defer func() { w.path = w.path[:len(w.path)-1] }()
	for _, child := range w.children[node] {
		if err := w.walk(child); err != nil {
			return err
		}
	}
	w.states[node] = finished
	w.order = append(w.order, node)
	return nil
}

func (w *orderWalker) cycle(repeated Node) string {
	labels := make([]string, 0, len(w.path)+1)
	found := false
	for _, node := range w.path {
		if node == repeated {
			found = true
		}
		if found {
			labels = append(labels, label(node))
		}
	}
	labels = append(labels, label(repeated))
	return strings.Join(labels, " -> ")
}

func PrintLoadOrder(output io.Writer, root Node, order []Node) {
	fmt.Fprintf(output, "Порядок загрузки для %s:\n", label(root))
	for index, node := range order {
		fmt.Fprintf(output, "  %d. %s\n", index+1, label(node))
	}
}
