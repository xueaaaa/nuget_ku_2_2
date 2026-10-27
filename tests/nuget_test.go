package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"nuget_ku_2_2/src/nuget"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (send roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return send(request)
}

func newNuGetClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status := http.StatusOK
		body := ""
		switch request.URL.String() {
		case "https://feed.example/v3/index.json":
			body = `{"version":"3.0.0","resources":[` +
				`{"@id":"https://content.example/flat/","@type":"PackageBaseAddress/3.0.0"}]}`
		case "https://content.example/flat/demo.package/1.2.0/demo.package.nuspec":
			body = groupedManifest
		case "https://content.example/flat/empty.package/1.0.0/empty.package.nuspec":
			body = emptyManifest
		case "https://content.example/flat/flat.package/1.0.0/flat.package.nuspec":
			body = flatManifest
		default:
			status = http.StatusNotFound
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})}
}

const groupedManifest = `<?xml version="1.0"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>Demo.Package</id><version>1.2.0</version>
    <dependencies>
      <group targetFramework="net8.0">
        <dependency id="First.Library" version="[2.0.0,3.0.0)" />
        <dependency id="Second.Library" version="4.1.0" />
      </group>
      <group targetFramework="net6.0" />
    </dependencies>
  </metadata>
</package>`

const emptyManifest = `<package><metadata><id>Empty.Package</id>
<version>1.0.0</version></metadata></package>`

const flatManifest = `<package><metadata><id>Flat.Package</id>
<version>1.0.0</version><dependencies>
<dependency id="Common.Library" version="[1.0.0,)" />
</dependencies></metadata></package>`

func TestFetchDirectDependencies(t *testing.T) {
	result, err := nuget.FetchDirect(context.Background(), newNuGetClient(),
		"https://feed.example/v3/index.json", "Demo.Package", "1.02.0.0+build")
	if err != nil {
		t.Fatal(err)
	}
	expectedGroups := []nuget.Group{
		{TargetFramework: "net8.0", Dependencies: []nuget.Dependency{
			{ID: "First.Library", Version: "[2.0.0,3.0.0)"},
			{ID: "Second.Library", Version: "4.1.0"},
		}},
		{TargetFramework: "net6.0"},
	}
	if !reflect.DeepEqual(result.Groups, expectedGroups) {
		t.Fatalf("неверные группы зависимостей: %+v", result.Groups)
	}
	var output bytes.Buffer
	nuget.PrintDirect(&output, result)
	for _, line := range []string{"[net8.0]", "First.Library [2.0.0,3.0.0)",
		"Second.Library 4.1.0", "[net6.0]"} {
		if !strings.Contains(output.String(), line) {
			t.Errorf("нет строки %q в выводе %q", line, output.String())
		}
	}
}

func TestEmptyPackageAndMissingVersion(t *testing.T) {
	indexURL := "https://feed.example/v3/index.json"
	client := newNuGetClient()
	result, err := nuget.FetchDirect(context.Background(), client,
		indexURL, "Empty.Package", "1")
	if err != nil || len(result.Flat) != 0 || len(result.Groups) != 0 {
		t.Fatalf("пакет без зависимостей обработан неверно: %+v, %v", result, err)
	}
	var output bytes.Buffer
	nuget.PrintDirect(&output, result)
	if !strings.Contains(output.String(), "отсутствуют") {
		t.Fatalf("нет сообщения об отсутствии зависимостей: %q", output.String())
	}
	_, err = nuget.FetchDirect(context.Background(), client,
		indexURL, "Missing.Package", "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("ожидалась ошибка для отсутствующего пакета: %v", err)
	}
}

func TestFlatDependencies(t *testing.T) {
	result, err := nuget.FetchDirect(context.Background(), newNuGetClient(),
		"https://feed.example/v3/index.json", "Flat.Package", "1.0.0")
	expected := []nuget.Dependency{{ID: "Common.Library", Version: "[1.0.0,)"}}
	if err != nil || !reflect.DeepEqual(result.Flat, expected) || len(result.Groups) != 0 {
		t.Fatalf("плоский список зависимостей обработан неверно: %+v, %v", result, err)
	}
	var output bytes.Buffer
	nuget.PrintDirect(&output, result)
	if !strings.Contains(output.String(), "Common.Library [1.0.0,)") {
		t.Fatalf("зависимость не выведена: %q", output.String())
	}
}
