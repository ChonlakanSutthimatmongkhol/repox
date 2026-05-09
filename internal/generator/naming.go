// Package generator handles template rendering and file generation.
package generator

import (
	"strings"
	"unicode"
)

// ToSnakeCase converts a string to snake_case.
// Examples: "watchList" → "watch_list", "WatchList" → "watch_list", "watch-list" → "watch_list"
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}
	// First split on non-alphanumeric separators
	words := splitWords(s)
	return strings.ToLower(strings.Join(words, "_"))
}

// ToPascalCase converts a string to PascalCase.
// Examples: "watchlist" → "Watchlist", "watch-list" → "WatchList", "watch_list" → "WatchList"
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}
	words := splitWords(s)
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, "")
}

// ToCamelCase converts a string to camelCase.
// Examples: "watchlist" → "watchlist", "watch-list" → "watchList", "watch_list" → "watchList"
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	pascal := ToPascalCase(s)
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// splitWords breaks s into words by separators (_, -, space) and camelCase boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		if r == '_' || r == '-' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			// Insert boundary before uppercase if previous char was lowercase/digit
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				words = append(words, current.String())
				current.Reset()
			} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) && current.Len() > 1 {
				// Handles "XMLParser" → ["XML", "Parser"]
				words = append(words, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	// lowercase the rest so "WORD" → "Word"
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}
