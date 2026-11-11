package visual

import (
	"fmt"
	"html"
	"io"

	"nuget_ku_2_2/src/graph"
)

func nodeLabel(node graph.Node) string {
	return node.ID + "@" + node.Version
}

func PrintMermaid(output io.Writer, result graph.Graph) {
	indices := make(map[graph.Node]int, len(result.Nodes))
	fmt.Fprintln(output, "flowchart LR")
	for index, node := range result.Nodes {
		indices[node] = index
		fmt.Fprintf(output, "  n%d[\"%s\"]\n", index, html.EscapeString(nodeLabel(node)))
	}
	for _, edge := range result.Edges {
		fmt.Fprintf(output, "  n%d --> n%d\n", indices[edge.From], indices[edge.To])
	}
}
