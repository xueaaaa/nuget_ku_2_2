package visual

import (
	"fmt"
	"io"

	"nuget_ku_2_2/src/graph"
)

type treePrinter struct {
	output   io.Writer
	children map[graph.Node][]graph.Node
	active   map[graph.Node]bool
	seen     map[graph.Node]bool
}

func PrintASCII(output io.Writer, result graph.Graph) {
	printer := treePrinter{
		output: output, children: make(map[graph.Node][]graph.Node),
		active: make(map[graph.Node]bool), seen: make(map[graph.Node]bool),
	}
	for _, edge := range result.Edges {
		printer.children[edge.From] = append(printer.children[edge.From], edge.To)
	}
	fmt.Fprintln(output, "\nASCII-дерево:")
	fmt.Fprintln(output, nodeLabel(result.Root))
	printer.walk(result.Root, "")
}

func (p *treePrinter) walk(node graph.Node, prefix string) {
	p.active[node] = true
	p.seen[node] = true
	defer delete(p.active, node)
	children := p.children[node]
	for index, child := range children {
		branch, next := "+-- ", prefix+"|   "
		if index == len(children)-1 {
			branch, next = "\\-- ", prefix+"    "
		}
		marker := ""
		if p.active[child] {
			marker = " [цикл]"
		} else if p.seen[child] {
			marker = " [повтор]"
		}
		fmt.Fprintln(p.output, prefix+branch+nodeLabel(child)+marker)
		if marker == "" {
			p.walk(child, next)
		}
	}
}
