package sitegen

import (
	"strings"

	markdownpkg "github.com/freeDog-wy/go-backend-template/pkg/markdown"
)

type TOCEntry = markdownpkg.TOCEntry

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func summaryDescription(value string) string {
	value = normalizeText(value)
	runes := []rune(value)
	if len(runes) <= 160 {
		return value
	}
	return string(runes[:157]) + "..."
}
