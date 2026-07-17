package promptkitty

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	vecbm25 "github.com/hupe1980/vecgo/lexical/bm25"
	"github.com/hupe1980/vecgo/model"
)

const (
	commonTermPercent = 80
	searchFieldCount  = 4
)

var searchFieldWeights = [searchFieldCount]float64{4, 2, 1, 1}

var searchStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "for": true, "in": true,
	"of": true, "on": true, "or": true, "the": true, "to": true,
}

var searchTermExpansions = map[string][]string{
	"write": {"author"},
}

type searchIndex struct {
	fields            [searchFieldCount]*vecbm25.MemoryIndex
	documentFrequency map[string]int
	documentCount     int
}

type searchResult struct {
	index int
	score float64
}

func newSearchIndex(components []Component, bodies map[string]string) (searchIndex, error) {
	index := searchIndex{
		documentFrequency: make(map[string]int),
		documentCount:     len(components),
	}
	for fieldIndex := range index.fields {
		index.fields[fieldIndex] = vecbm25.New()
	}

	for componentIndex, component := range components {
		fields := [searchFieldCount]string{
			component.Name,
			component.Description,
			searchMetadataText(component),
			bodies[component.Path],
		}
		seen := make(map[string]bool)
		for fieldIndex, text := range fields {
			terms := tokenizeSearchText(text)
			for _, term := range terms {
				seen[term] = true
			}
			if len(terms) == 0 {
				continue
			}
			if err := index.fields[fieldIndex].Add(model.ID(componentIndex+1), strings.Join(terms, " ")); err != nil {
				return searchIndex{}, fmt.Errorf("index PromptKit component %q: %w", component.Name, err)
			}
		}
		for term := range seen {
			index.documentFrequency[term]++
		}
	}

	return index, nil
}

func (index searchIndex) rank(query string, components []Component, filter Filter) []int {
	terms := uniqueSearchTerms(query)
	terms = index.removeCommonTerms(terms)
	if len(terms) == 0 || index.documentCount == 0 {
		return nil
	}

	totalScores := make([]float64, index.documentCount)
	normalizedQuery := strings.Join(terms, " ")
	for fieldIndex, field := range index.fields {
		candidates, err := field.Search(context.Background(), normalizedQuery, index.documentCount)
		if err != nil {
			return nil
		}
		for _, candidate := range candidates {
			documentIndex := int(candidate.ID) - 1
			if documentIndex < 0 || documentIndex >= len(totalScores) {
				continue
			}
			totalScores[documentIndex] += searchFieldWeights[fieldIndex] * float64(candidate.Score)
		}
	}

	results := make([]searchResult, 0)
	for documentIndex, score := range totalScores {
		if score <= 0 || !matchesFilter(components[documentIndex], filter) {
			continue
		}
		results = append(results, searchResult{index: documentIndex, score: score})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}

		return results[i].index < results[j].index
	})

	indices := make([]int, len(results))
	for resultIndex, result := range results {
		indices[resultIndex] = result.index
	}

	return indices
}

func (index searchIndex) removeCommonTerms(terms []string) []string {
	filtered := terms[:0]
	for _, term := range terms {
		frequency := index.documentFrequency[term]
		if frequency == 0 || frequency*100 > index.documentCount*commonTermPercent {
			continue
		}
		filtered = append(filtered, term)
	}

	return filtered
}

func searchMetadataText(component Component) string {
	values := []string{string(component.Type), component.Category}
	for _, key := range sortedKeys(component.Metadata) {
		switch key {
		case "name", "description", "path":
			continue
		}
		values = append(values, key)
		appendSearchValue(&values, component.Metadata[key])
	}

	return strings.Join(values, " ")
}

func appendSearchValue(values *[]string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range sortedKeys(typed) {
			*values = append(*values, key)
			appendSearchValue(values, typed[key])
		}
	case []any:
		for _, item := range typed {
			appendSearchValue(values, item)
		}
	case nil:
		return
	default:
		*values = append(*values, fmt.Sprint(typed))
	}
}

func uniqueSearchTerms(query string) []string {
	seen := make(map[string]bool)
	for _, term := range tokenizeSearchText(query) {
		if searchStopWords[term] {
			continue
		}
		seen[term] = true
		for _, expansion := range searchTermExpansions[term] {
			seen[expansion] = true
		}
	}

	terms := make([]string, 0, len(seen))
	for term := range seen {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	return terms
}

func tokenizeSearchText(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
