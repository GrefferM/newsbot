package news

import (
	"regexp"
	"strings"
)

type Symbol string

var (
	symbolRe      = regexp.MustCompile(`^[A-Z0-9]{2,15}$`)
	parenSymbolRe = regexp.MustCompile(`\(([A-Z0-9]{2,15})\)`)
	listSepRe     = regexp.MustCompile(`\s*(?:,|&|\band\b)\s*`)
	usdtPairRe    = regexp.MustCompile(`\b([A-Z0-9]{2,20})USDT\b`)
	scalePrefixes = []string{"1000000", "1000"}
)

func SymbolsFromList(list string) []Symbol {
	var out []Symbol

	for _, part := range listSepRe.Split(list, -1) {
		part = strings.TrimSpace(part)

		if m := parenSymbolRe.FindStringSubmatch(part); m != nil {
			part = m[1]
		}

		if symbolRe.MatchString(part) {
			out = append(out, Symbol(part))
		}
	}

	return out
}

func SymbolsFromUSDTPairs(text string) []Symbol {
	var out []Symbol

	for _, m := range usdtPairRe.FindAllStringSubmatch(text, -1) {
		out = append(out, Symbol(withoutScale(m[1])))
	}

	return out
}

func withoutScale(s string) string {
	for _, prefix := range scalePrefixes {
		s = strings.TrimPrefix(s, prefix)
	}

	return s
}
