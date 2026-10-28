package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"nuget_ku_2_2/src/graph"
	"nuget_ku_2_2/src/nuget"
)

const feedURL = "https://feed.example/v3/index.json"
const flatURL = "https://content.example/flat/"

func transitiveResponses() map[string]string {
	return map[string]string{
		feedURL: `{"version":"3.0.0","resources":[` +
			`{"@id":"https://content.example/flat/",` +
			`"@type":"PackageBaseAddress/3.0.0"}]}`,
		flatURL + "root.package/1.0.0/root.package.nuspec": `<package><metadata><id>Root.Package</id>` +
			`<version>1.0.0</version>` +
			`<dependencies><dependency id="Alpha" version="[1.0.0,2.0.0)"/>` +
			`<dependency id="Beta" version="[2.0.0]"/></dependencies>` +
			`</metadata></package>`,
		flatURL + "alpha/index.json": `{"versions":["1.0.0-beta","1.1.0","1.0.0"]}`,
		flatURL + "beta/index.json":  `{"versions":["1.0.0","2.0.0"]}`,
		flatURL + "alpha/1.0.0/alpha.nuspec": `<package><metadata><id>Alpha</id><version>1.0.0</version>` +
			`<dependencies><dependency id="Beta" version="[2.0.0]"/>` +
			`</dependencies></metadata></package>`,
		flatURL + "beta/2.0.0/beta.nuspec": `<package><metadata><id>Beta</id><version>2.0.0</version>` +
			`</metadata></package>`,
	}
}

func transitiveClient(betaRequests *int) *http.Client {
	responses := transitiveResponses()
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.String(), "/beta/index.json") {
			*betaRequests++
		}
		body, found := responses[request.URL.String()]
		if !found {
			return nil, fmt.Errorf("неожиданный запрос: %s", request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)),
			Header: make(http.Header), Request: request,
		}, nil
	})}
}

func TestNuGetTransitiveGraph(t *testing.T) {
	betaRequests := 0
	ctx := context.Background()
	source, err := nuget.NewSource(ctx, transitiveClient(&betaRequests), feedURL, "")
	if err != nil {
		t.Fatal(err)
	}
	root := graph.Node{ID: "Root.Package", Version: "1.0.0"}
	result, err := graph.Build(ctx, source, root, 2, "")
	if err != nil || len(result.Nodes) != 3 || len(result.Edges) != 3 {
		t.Fatalf("неверный граф NuGet: %+v, %v", result, err)
	}
	if result.Nodes[1].Version != "1.0.0" || result.Nodes[2].Version != "2.0.0" {
		t.Fatalf("неверно выбраны версии: %+v", result.Nodes)
	}
	if betaRequests != 1 {
		t.Fatalf("список версий Beta запрошен %d раз", betaRequests)
	}
}

func TestNuGetFilterSkipsRequests(t *testing.T) {
	betaRequests := 0
	ctx := context.Background()
	source, err := nuget.NewSource(ctx, transitiveClient(&betaRequests), feedURL, "beta")
	if err != nil {
		t.Fatal(err)
	}
	root := graph.Node{ID: "Root.Package", Version: "1.0.0"}
	result, err := graph.Build(ctx, source, root, 2, "beta")
	if err != nil || len(result.Nodes) != 2 || len(result.Edges) != 1 {
		t.Fatalf("фильтр NuGet не применён: %+v, %v", result, err)
	}
	if betaRequests != 0 {
		t.Fatalf("исключённый пакет был запрошен: %d", betaRequests)
	}
}
