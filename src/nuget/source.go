package nuget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nuget_ku_2_2/src/graph"
)

type Source struct {
	client   *http.Client
	base     string
	filter   string
	versions map[string][]string
}

func NewSource(ctx context.Context, client *http.Client,
	indexURL string, filter string) (*Source, error) {
	base, err := fetchPackageBase(ctx, client, indexURL)
	if err != nil {
		return nil, err
	}
	return &Source{
		client: client, base: base, filter: strings.ToLower(filter),
		versions: make(map[string][]string),
	}, nil
}

func (s *Source) Dependencies(ctx context.Context, node graph.Node) ([]graph.Node, error) {
	result, err := fetchDirectFromBase(ctx, s.client, s.base, node.ID, node.Version)
	if err != nil {
		return nil, err
	}
	items := append([]Dependency(nil), result.Flat...)
	for _, group := range result.Groups {
		items = append(items, group.Dependencies...)
	}
	children := make([]graph.Node, 0, len(items))
	seen := make(map[graph.Node]bool)
	for _, item := range items {
		if s.filter != "" && strings.Contains(strings.ToLower(item.ID), s.filter) {
			continue
		}
		version, err := s.resolve(ctx, item.ID, item.Version)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", item.ID, item.Version, err)
		}
		child := graph.Node{ID: item.ID, Version: version}
		if !seen[child] {
			seen[child] = true
			children = append(children, child)
		}
	}
	return children, nil
}

func (s *Source) resolve(ctx context.Context, id string, rawRange string) (string, error) {
	versionRange, err := parseRange(rawRange)
	if err != nil {
		return "", err
	}
	versions, err := s.listVersions(ctx, id)
	if err != nil {
		return "", err
	}
	return versionRange.lowest(versions)
}

func (s *Source) listVersions(ctx context.Context, id string) ([]string, error) {
	key := strings.ToLower(id)
	if versions, ok := s.versions[key]; ok {
		return versions, nil
	}
	address := strings.TrimRight(s.base, "/") + "/" + key + "/index.json"
	data, err := getBody(ctx, s.client, address)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список версий: %w", err)
	}
	var listing struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, fmt.Errorf("ошибка JSON списка версий: %w", err)
	}
	if len(listing.Versions) == 0 {
		return nil, fmt.Errorf("для пакета %s нет версий", id)
	}
	s.versions[key] = listing.Versions
	return listing.Versions, nil
}
