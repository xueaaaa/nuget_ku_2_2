package nuget

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	packageBaseType      = "PackageBaseAddress/3.0.0"
	maxResponseBytes     = 4 << 20
	minVersionSegments   = 3
	revisionPartSegments = 4
)

type serviceIndex struct {
	Version   string `json:"version"`
	Resources []struct {
		ID   string `json:"@id"`
		Type string `json:"@type"`
	} `json:"resources"`
}

type manifest struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		ID           string       `xml:"id"`
		Version      string       `xml:"version"`
		Dependencies dependencies `xml:"dependencies"`
	} `xml:"metadata"`
}

type dependencies struct {
	Flat   []Dependency `xml:"dependency"`
	Groups []Group      `xml:"group"`
}

type Dependency struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

type Group struct {
	TargetFramework string       `xml:"targetFramework,attr"`
	Dependencies    []Dependency `xml:"dependency"`
}

type Result struct {
	Package string
	Version string
	Flat    []Dependency
	Groups  []Group
}

func FetchDirect(ctx context.Context, client *http.Client, indexURL string,
	packageID string, version string) (Result, error) {
	base, err := fetchPackageBase(ctx, client, indexURL)
	if err != nil {
		return Result{}, err
	}
	return fetchDirectFromBase(ctx, client, base, packageID, version)
}

func fetchDirectFromBase(ctx context.Context, client *http.Client,
	base string, packageID string, version string) (Result, error) {
	manifestURL := buildManifestURL(base, packageID, version)
	data, err := getBody(ctx, client, manifestURL)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось получить .nuspec: %w", err)
	}
	return parseManifest(data, packageID, version)
}

func fetchPackageBase(ctx context.Context, client *http.Client,
	indexURL string) (string, error) {
	data, err := getBody(ctx, client, indexURL)
	if err != nil {
		return "", fmt.Errorf("не удалось получить service index: %w", err)
	}
	var index serviceIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return "", fmt.Errorf("ошибка JSON service index: %w", err)
	}
	if !strings.HasPrefix(index.Version, "3.") {
		return "", errors.New("service index не соответствует NuGet V3")
	}
	for _, resource := range index.Resources {
		if resource.Type == packageBaseType && validHTTPURL(resource.ID) {
			return resource.ID, nil
		}
	}
	return "", errors.New("service index не содержит PackageBaseAddress/3.0.0")
}

func validHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != ""
}

func buildManifestURL(base string, packageID string, version string) string {
	id := strings.ToLower(packageID)
	part := normalizeVersion(version)
	return strings.TrimRight(base, "/") + "/" + id + "/" + part + "/" + id + ".nuspec"
}

func normalizeVersion(version string) string {
	withoutBuild, _, _ := strings.Cut(version, "+")
	numeric, prerelease, hasPrerelease := strings.Cut(withoutBuild, "-")
	numbers := strings.Split(numeric, ".")
	for index, number := range numbers {
		numbers[index] = strings.TrimLeft(number, "0")
		if numbers[index] == "" {
			numbers[index] = "0"
		}
	}
	for len(numbers) < minVersionSegments {
		numbers = append(numbers, "0")
	}
	if len(numbers) == revisionPartSegments && numbers[len(numbers)-1] == "0" {
		numbers = numbers[:len(numbers)-1]
	}
	result := strings.Join(numbers, ".")
	if hasPrerelease {
		result += "-" + strings.ToLower(prerelease)
	}
	return result
}

func getBody(ctx context.Context, client *http.Client, address string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", address, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("ответ репозитория слишком большой")
	}
	return data, nil
}

func parseManifest(data []byte, packageID string, version string) (Result, error) {
	var document manifest
	if err := xml.Unmarshal(data, &document); err != nil {
		return Result{}, fmt.Errorf("ошибка XML .nuspec: %w", err)
	}
	if !strings.EqualFold(document.Metadata.ID, packageID) ||
		normalizeVersion(document.Metadata.Version) != normalizeVersion(version) {
		return Result{}, errors.New(".nuspec содержит другое имя или версию пакета")
	}
	deps := document.Metadata.Dependencies
	if len(deps.Flat) > 0 && len(deps.Groups) > 0 {
		return Result{}, errors.New(".nuspec смешивает зависимости и группы")
	}
	if err := validateDependencies(deps); err != nil {
		return Result{}, err
	}
	return Result{packageID, version, deps.Flat, deps.Groups}, nil
}

func validateDependencies(deps dependencies) error {
	for _, item := range deps.Flat {
		if item.ID == "" || item.Version == "" {
			return errors.New(".nuspec содержит зависимость без имени или версии")
		}
	}
	for _, group := range deps.Groups {
		for _, item := range group.Dependencies {
			if item.ID == "" || item.Version == "" {
				return errors.New(".nuspec содержит зависимость без имени или версии")
			}
		}
	}
	return nil
}

func PrintDirect(output io.Writer, result Result) {
	fmt.Fprintf(output, "Прямые зависимости %s %s:\n", result.Package, result.Version)
	if len(result.Flat) == 0 && len(result.Groups) == 0 {
		fmt.Fprintln(output, "  отсутствуют")
		return
	}
	printDependencies(output, result.Flat)
	for _, group := range result.Groups {
		framework := group.TargetFramework
		if framework == "" {
			framework = "по умолчанию"
		}
		fmt.Fprintf(output, "[%s]\n", framework)
		if len(group.Dependencies) == 0 {
			fmt.Fprintln(output, "  отсутствуют")
		}
		printDependencies(output, group.Dependencies)
	}
}

func printDependencies(output io.Writer, items []Dependency) {
	for _, item := range items {
		fmt.Fprintf(output, "  %s %s\n", item.ID, item.Version)
	}
}
