package prompts

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed *.md
var files embed.FS

var part = regexp.MustCompile(`(?m)^\{\{(\w+)\}\}\n`)

func Of(name string) string {
	return part.ReplaceAllStringFunc(read(name), func(line string) string {
		return read("_" + part.FindStringSubmatch(line)[1])
	})
}

func read(name string) string {
	data, err := files.ReadFile(name + ".md")
	if err != nil {
		panic(fmt.Sprintf("prompt %s: %v", name, err))
	}

	return strings.TrimRight(string(data), "\n") + "\n"
}
