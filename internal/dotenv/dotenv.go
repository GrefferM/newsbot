package dotenv

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func Load(path string) (int, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	defer f.Close()

	applied := 0
	scanner := bufio.NewScanner(f)

	for line := 1; scanner.Scan(); line++ {
		key, value, ok, err := parseLine(scanner.Text())
		if err != nil {
			return applied, fmt.Errorf("%s:%d: %w", path, line, err)
		}

		if !ok || isSet(key) {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return applied, err
		}

		applied++
	}

	return applied, scanner.Err()
}

func parseLine(raw string) (key, value string, ok bool, err error) {
	line := strings.TrimSpace(raw)

	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}

	key, value, found := strings.Cut(strings.TrimPrefix(line, "export "), "=")
	key = strings.TrimSpace(key)

	if !found || key == "" {
		return "", "", false, errors.New("expected KEY=VALUE")
	}

	return key, unquote(strings.TrimSpace(value)), true, nil
}

func isSet(key string) bool {
	_, set := os.LookupEnv(key)

	return set
}

func unquote(v string) string {
	if isQuoted(v, '"') || isQuoted(v, '\'') {
		return v[1 : len(v)-1]
	}

	if i := strings.Index(v, " #"); i >= 0 {
		return strings.TrimSpace(v[:i])
	}

	return v
}

func isQuoted(v string, q byte) bool {
	return len(v) >= 2 && v[0] == q && v[len(v)-1] == q
}
