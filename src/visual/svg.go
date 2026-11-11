package visual

import (
	"bufio"
	"fmt"
	"html"
	"os"

	"nuget_ku_2_2/src/graph"
)

const (
	nodeWidth  = 220
	nodeHeight = 44
	columnStep = 300
	rowStep    = 76
	margin     = 32
)

type point struct{ x, y int }

func WriteSVG(path string, result graph.Graph) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("не удалось создать SVG: %w", err)
	}
	defer file.Close()
	output := bufio.NewWriter(file)
	positions, width, height := layout(result)
	fmt.Fprintf(output, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %d %d\">\n", width, height)
	fmt.Fprintln(output, `<defs><marker id="arrow" markerWidth="10" markerHeight="10"`+
		` refX="9" refY="3" orient="auto"><path d="M0,0 L0,6 L9,3 z"`+
		` fill="#62758c"/></marker></defs>`)
	fmt.Fprintln(output, "<rect width=\"100%\" height=\"100%\" fill=\"#f8fafc\"/>")
	writeEdges(output, result, positions)
	writeNodes(output, result, positions)
	fmt.Fprintln(output, "</svg>")
	return output.Flush()
}

func layout(result graph.Graph) (map[graph.Node]point, int, int) {
	depths := map[graph.Node]int{result.Root: 0}
	queue := []graph.Node{result.Root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, edge := range result.Edges {
			if edge.From == node {
				if _, found := depths[edge.To]; !found {
					depths[edge.To] = depths[node] + 1
					queue = append(queue, edge.To)
				}
			}
		}
	}
	positions := make(map[graph.Node]point, len(result.Nodes))
	maxDepth := 0
	for index, node := range result.Nodes {
		depth := depths[node]
		if depth > maxDepth {
			maxDepth = depth
		}
		positions[node] = point{margin + depth*columnStep, margin + index*rowStep}
	}
	return positions, maxDepth*columnStep + nodeWidth + margin*2, len(result.Nodes)*rowStep + margin*2
}

func writeEdges(output *bufio.Writer, result graph.Graph, positions map[graph.Node]point) {
	for _, edge := range result.Edges {
		from, to := positions[edge.From], positions[edge.To]
		x1, y1 := from.x+nodeWidth, from.y+nodeHeight/2
		x2, y2 := to.x, to.y+nodeHeight/2
		middle := (x1 + x2) / 2
		fmt.Fprintf(output, `<path d="M%d %d C%d %d %d %d %d %d"`+
			` fill="none" stroke="#62758c" stroke-width="2"`+
			` marker-end="url(#arrow)"/>`+"\n",
			x1, y1, middle, y1, middle, y2, x2, y2)
	}
}

func writeNodes(output *bufio.Writer, result graph.Graph, positions map[graph.Node]point) {
	for _, node := range result.Nodes {
		position := positions[node]
		fmt.Fprintf(output, `<rect x="%d" y="%d" width="%d" height="%d"`+
			` rx="8" fill="#e8f0ff" stroke="#466ca8"/>`+"\n",
			position.x, position.y, nodeWidth, nodeHeight)
		fmt.Fprintf(output, `<text x="%d" y="%d" font-family="sans-serif"`+
			` font-size="14" fill="#17243b">%s</text>`+"\n",
			position.x+12, position.y+28, html.EscapeString(nodeLabel(node)))
	}
}
