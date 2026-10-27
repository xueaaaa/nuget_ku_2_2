package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"nuget_ku_2_2/src/config"
	"nuget_ku_2_2/src/nuget"
)

func main() {
	settings, err := config.Parse(os.Args[1:])
	if err != nil {
		exitWithError(err)
	}
	if settings.TestMode {
		exitWithError(fmt.Errorf("тестовый репозиторий будет доступен на следующем этапе"))
	}
	client := &http.Client{Timeout: 15 * time.Second}
	result, err := nuget.FetchDirect(
		context.Background(), client, settings.Repository,
		settings.Package, settings.Version,
	)
	if err != nil {
		exitWithError(err)
	}
	nuget.PrintDirect(os.Stdout, result)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Ошибка:", err)
	os.Exit(2)
}
