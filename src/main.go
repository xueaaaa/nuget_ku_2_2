package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"nuget_ku_2_2/src/config"
	"nuget_ku_2_2/src/graph"
	"nuget_ku_2_2/src/nuget"
	"nuget_ku_2_2/src/testrepo"
)

func main() {
	settings, err := config.Parse(os.Args[1:])
	if err != nil {
		exitWithError(err)
	}
	source, err := newSource(settings)
	if err != nil {
		exitWithError(err)
	}
	root := graph.Node{ID: settings.Package, Version: settings.Version}
	result, err := graph.Build(context.Background(), source, root,
		settings.MaxDepth, settings.Filter)
	if err != nil {
		exitWithError(err)
	}
	graph.Print(os.Stdout, result)
}

func newSource(settings config.Settings) (graph.Source, error) {
	if settings.TestMode {
		source, err := testrepo.Load(settings.Repository, settings.Version)
		if err != nil {
			return nil, err
		}
		if !source.Has(settings.Package) {
			return nil, fmt.Errorf("тестовый репозиторий не содержит пакет %s", settings.Package)
		}
		return source, nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return nuget.NewSource(context.Background(), client,
		settings.Repository, settings.Filter)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Ошибка:", err)
	os.Exit(2)
}
