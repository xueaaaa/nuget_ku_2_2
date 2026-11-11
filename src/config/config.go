package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const defaultDepth = 3

var (
	packagePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	testPackagePattern = regexp.MustCompile(`^[A-Z]+$`)
	versionPattern     = regexp.MustCompile(
		`^[0-9]+(\.[0-9]+){0,3}(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`,
	)
)

type Settings struct {
	Package    string
	Repository string
	TestMode   bool
	Version    string
	Output     string
	ASCIITree  bool
	MaxDepth   int
	Filter     string
}

func Parse(args []string) (Settings, error) {
	settings := Settings{}
	flags := newFlagSet(&settings)
	if err := flags.Parse(args); err != nil {
		return Settings{}, err
	}
	if flags.NArg() != 0 {
		return Settings{}, fmt.Errorf("неожиданные аргументы: %s", strings.Join(flags.Args(), " "))
	}
	if err := validate(settings, flags); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func newFlagSet(settings *Settings) *flag.FlagSet {
	flags := flag.NewFlagSet("nuget_ku_2_2", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&settings.Package, "package", "", "имя пакета")
	flags.StringVar(&settings.Repository, "repository", "", "URL или путь к файлу")
	flags.BoolVar(&settings.TestMode, "test-mode", false, "режим тестового файла")
	flags.StringVar(&settings.Version, "version", "", "версия пакета")
	flags.StringVar(&settings.Output, "output", "graph.svg", "имя SVG-файла")
	flags.BoolVar(&settings.ASCIITree, "ascii-tree", false, "вывод ASCII-дерева")
	flags.IntVar(&settings.MaxDepth, "max-depth", defaultDepth, "максимальная глубина")
	flags.StringVar(&settings.Filter, "filter", "", "подстрока фильтра")
	return flags
}

func validate(settings Settings, flags *flag.FlagSet) error {
	if !packagePattern.MatchString(settings.Package) {
		return errors.New("--package: укажите имя из латинских букв, цифр, точек, дефисов или подчёркиваний")
	}
	if settings.TestMode && !testPackagePattern.MatchString(settings.Package) {
		return errors.New("--package: в тестовом режиме имя должно состоять из заглавных букв A-Z")
	}
	if err := validateRepository(settings.Repository, settings.TestMode); err != nil {
		return err
	}
	if !versionPattern.MatchString(settings.Version) {
		return errors.New("--version: укажите версию вида 1, 1.2, 1.2.3 или 1.2.3.4")
	}
	if err := validateOutput(settings.Output); err != nil {
		return err
	}
	if settings.MaxDepth < 0 {
		return errors.New("--max-depth: глубина не может быть отрицательной")
	}
	if filterWasSet(flags) && strings.TrimSpace(settings.Filter) == "" {
		return errors.New("--filter: подстрока не может быть пустой")
	}
	return nil
}

func validateRepository(repository string, testMode bool) error {
	if testMode {
		file, err := os.Open(repository)
		if err != nil {
			return fmt.Errorf("--repository: не удалось открыть тестовый файл: %w", err)
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("--repository: нужен обычный файл")
		}
		return nil
	}
	return validateURL(repository)
}

func validateURL(repository string) error {
	parsed, err := url.Parse(repository)
	if err != nil || strings.ContainsAny(repository, " \t\r\n") {
		return errors.New("--repository: укажите корректный URL HTTP или HTTPS")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return errors.New("--repository: укажите корректный URL HTTP или HTTPS")
	}
	return nil
}

func validateOutput(output string) error {
	if output == "" || strings.ContainsAny(output, `/\`) {
		return errors.New("--output: укажите только имя файла с расширением .svg")
	}
	if strings.EqualFold(output, ".svg") || !strings.HasSuffix(strings.ToLower(output), ".svg") {
		return errors.New("--output: укажите только имя файла с расширением .svg")
	}
	return nil
}

func filterWasSet(flags *flag.FlagSet) bool {
	found := false
	flags.Visit(func(option *flag.Flag) {
		if option.Name == "filter" {
			found = true
		}
	})
	return found
}

func Print(output io.Writer, settings Settings) {
	fmt.Fprintf(output, "package = %s\n", settings.Package)
	fmt.Fprintf(output, "repository = %s\n", settings.Repository)
	fmt.Fprintf(output, "test_mode = %t\n", settings.TestMode)
	fmt.Fprintf(output, "version = %s\n", settings.Version)
	fmt.Fprintf(output, "output = %s\n", settings.Output)
	fmt.Fprintf(output, "ascii_tree = %t\n", settings.ASCIITree)
	fmt.Fprintf(output, "max_depth = %d\n", settings.MaxDepth)
	fmt.Fprintf(output, "filter = %s\n", settings.Filter)
}
