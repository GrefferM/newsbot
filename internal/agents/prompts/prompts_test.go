package prompts

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEveryAgentHasAPromptWithItsSharedPartsInPlace(t *testing.T) {
	used := map[string]bool{}

	for _, name := range []string{"binance", "osinter", "risk", "trader", "quick", "funnel-survey", "funnel-select", "funnel-dig", "funnel-rank", "guard"} {
		text := Of(name)

		if len(text) < 100 || strings.Contains(text, "{{") {
			t.Errorf("%s: prompt too short or a shared part not filled in", name)
		}

		for _, m := range part.FindAllStringSubmatch(read(name), -1) {
			used[m[1]] = true
		}
	}

	shared, _ := fs.Glob(files, "_*.md")

	for _, file := range shared {
		if name := strings.TrimSuffix(strings.TrimPrefix(file, "_"), ".md"); !used[name] {
			t.Errorf("shared part %s is used by no prompt", name)
		}
	}
}
