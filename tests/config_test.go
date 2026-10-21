package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"nuget_ku_2_2/src/config"
)

func baseArgs() []string {
	return []string{
		"--package", "Newtonsoft.Json",
		"--repository", "https://api.nuget.org/v3/index.json",
		"--version", "13.0.3",
	}
}

func TestPrintsAllEightParameters(t *testing.T) {
	args := append(baseArgs(), "--output", "deps.svg", "--ascii-tree",
		"--max-depth", "2", "--filter", "System")
	settings, err := config.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	config.Print(&output, settings)
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	expected := []string{
		"package = Newtonsoft.Json",
		"repository = https://api.nuget.org/v3/index.json",
		"test_mode = false",
		"version = 13.0.3",
		"output = deps.svg",
		"ascii_tree = true",
		"max_depth = 2",
		"filter = System",
	}
	if !slices.Equal(lines, expected) {
		t.Fatalf("неверный вывод параметров:\nожидалось: %q\nполучено: %q", expected, lines)
	}
}

func TestInvalidOptions(t *testing.T) {
	cases := []struct {
		name    string
		changes []string
		want    string
	}{
		{"имя пакета", []string{"--package", "bad name"}, "--package"},
		{"URL", []string{"--repository", "ftp://example.org"}, "--repository"},
		{"режим файла", []string{"--test-mode=maybe"}, "test-mode"},
		{"версия", []string{"--version", "latest"}, "--version"},
		{"имя изображения", []string{"--output", "graph.png"}, "--output"},
		{"режим дерева", []string{"--ascii-tree=maybe"}, "ascii-tree"},
		{"глубина", []string{"--max-depth", "-1"}, "--max-depth"},
		{"фильтр", []string{"--filter", "  "}, "--filter"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := config.Parse(append(baseArgs(), test.changes...))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ожидалась ошибка %q, получено %v", test.want, err)
			}
		})
	}
}

func TestTestRepository(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.txt")
	if err := os.WriteFile(path, []byte("A: B\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--package", "A", "--repository", path,
		"--test-mode", "--version", "1.0"}
	settings, err := config.Parse(args)
	if err != nil || !settings.TestMode {
		t.Fatalf("корректный тестовый файл отклонён: %v", err)
	}
	_, err = config.Parse(append(args, "--repository", path+".missing"))
	if err == nil || !strings.Contains(err.Error(), "--repository") {
		t.Fatalf("отсутствующий файл принят: %v", err)
	}
}
