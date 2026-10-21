package main

import (
	"fmt"
	"os"

	"nuget_ku_2_2/src/config"
)

func main() {
	settings, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(2)
	}
	config.Print(os.Stdout, settings)
}
