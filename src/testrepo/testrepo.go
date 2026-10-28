package testrepo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"nuget_ku_2_2/src/graph"
)

var packageName = regexp.MustCompile(`^[A-Z]+$`)

type Source struct {
	version string
	items   map[string][]string
}

func Load(path string, version string) (*Source, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	source := &Source{version: version, items: make(map[string][]string)}
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		if err := source.addLine(scanner.Text(), line); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *Source) addLine(text string, line int) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("строка %d: ожидается формат A: B C", line)
	}
	id := strings.TrimSpace(parts[0])
	if !packageName.MatchString(id) {
		return fmt.Errorf("строка %d: имя пакета должно состоять из A-Z", line)
	}
	if _, exists := s.items[id]; exists {
		return fmt.Errorf("строка %d: пакет %s указан повторно", line, id)
	}
	children := strings.Fields(parts[1])
	for _, child := range children {
		if !packageName.MatchString(child) {
			return fmt.Errorf("строка %d: имя зависимости должно состоять из A-Z", line)
		}
	}
	s.items[id] = children
	return nil
}

func (s *Source) Has(id string) bool {
	_, exists := s.items[id]
	return exists
}

func (s *Source) Dependencies(ctx context.Context, node graph.Node) ([]graph.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	children := s.items[node.ID]
	result := make([]graph.Node, 0, len(children))
	for _, id := range children {
		result = append(result, graph.Node{ID: id, Version: s.version})
	}
	return result, nil
}
