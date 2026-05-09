package retriever

import (
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ScoreSimilarity returns a 0.0–1.0 score for how similar target is to example.
//
// Weights:
//   - 0.2 name keyword overlap
//   - 0.3 component structure match
//   - 0.2 import similarity
//   - 0.3 pattern match
func ScoreSimilarity(target string, example models.Example) float64 {
	return nameScore(target, example)*0.2 +
		structureScore(example)*0.3 +
		importScore(target, example)*0.2 +
		patternScore(target, example)*0.3
}

// FindSimilar returns the top N examples ranked by similarity to target.
func FindSimilar(target string, examples []models.Example, topN int) []models.Example {
	type scored struct {
		ex    models.Example
		score float64
	}
	ranked := make([]scored, len(examples))
	for i, ex := range examples {
		ranked[i] = scored{ex, ScoreSimilarity(target, ex)}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if topN > len(ranked) {
		topN = len(ranked)
	}
	result := make([]models.Example, topN)
	for i := range result {
		result[i] = ranked[i].ex
	}
	return result
}

// nameScore measures keyword overlap between target words and example name.
func nameScore(target string, example models.Example) float64 {
	targetWords := splitWords(target)
	if len(targetWords) == 0 {
		return 0
	}
	exName := strings.ToLower(example.Name)
	matches := 0
	for _, w := range targetWords {
		if strings.Contains(exName, w) {
			matches++
		}
	}
	return float64(matches) / float64(len(targetWords))
}

// structureScore measures how many components the example has (max 1.0).
// Counts Flutter fields (bloc, screen, usecase) and Go fields (handler, service)
// plus common fields (repository, test). Caps at 1.0.
func structureScore(example models.Example) float64 {
	m := example.Metadata
	count := 0
	if m.HasBloc {
		count++
	}
	if m.HasScreen {
		count++
	}
	if m.HasHandler {
		count++
	}
	if m.HasService {
		count++
	}
	if m.HasRepository {
		count++
	}
	if m.HasUseCase {
		count++
	}
	if m.HasTest {
		count++
	}
	if count == 0 {
		return 0
	}
	score := float64(count) / 5.0
	if score > 1.0 {
		return 1.0
	}
	return score
}

// importScore measures shared import fraction.
func importScore(_ string, example models.Example) float64 {
	total := len(example.Metadata.Imports)
	if total == 0 {
		return 0
	}
	// Score proportional to number of imports (richer = higher baseline).
	if total > 5 {
		return 1.0
	}
	return float64(total) / 5.0
}

// patternScore measures shared patterns.
func patternScore(target string, example models.Example) float64 {
	if len(example.Patterns) == 0 {
		return 0
	}
	targetLower := strings.ToLower(target)
	matches := 0
	for _, p := range example.Patterns {
		if strings.Contains(p, "bloc") && strings.Contains(targetLower, "bloc") {
			matches++
		} else if strings.Contains(p, "router") && strings.Contains(targetLower, "router") {
			matches++
		} else {
			matches++ // any pattern is a positive signal
		}
	}
	total := len(example.Patterns)
	if matches > total {
		matches = total
	}
	return float64(matches) / float64(total)
}

// splitWords splits a snake_case or camelCase name into lowercase words.
func splitWords(s string) []string {
	s = strings.ToLower(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var words []string
	for _, p := range parts {
		if p != "" {
			words = append(words, p)
		}
	}
	return words
}
