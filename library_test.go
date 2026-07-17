package promptkitty

import (
	"errors"
	"slices"
	"testing"
)

func TestEmbeddedCatalog(t *testing.T) {
	t.Parallel()

	library, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if got, want := library.Version(), "0.6.1"; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}

	tests := []struct {
		kind ComponentType
		want int
	}{
		{ComponentPersona, 15},
		{ComponentProtocol, 56},
		{ComponentFormat, 24},
		{ComponentTaxonomy, 5},
		{ComponentTemplate, 71},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			t.Parallel()
			if got := len(library.List(Filter{Type: test.kind})); got != test.want {
				t.Fatalf("List(%s) count = %d, want %d", test.kind, got, test.want)
			}
		})
	}

	if got := len(library.Pipelines()); got != 4 {
		t.Fatalf("Pipelines() count = %d, want 4", got)
	}
}

func TestCatalogSearchShowAndCopies(t *testing.T) {
	t.Parallel()

	library, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	results := library.Search("memory safety", Filter{Type: ComponentProtocol})
	if len(results) == 0 {
		t.Fatal("Search() returned no memory-safety protocols")
	}
	for _, component := range results {
		if component.Type != ComponentProtocol {
			t.Fatalf("Search() returned type %q, want protocol", component.Type)
		}
	}

	detail, err := library.Show("review-code")
	if err != nil {
		t.Fatalf("Show(review-code) error: %v", err)
	}
	if detail.Type != ComponentTemplate || detail.Category != "code-analysis" {
		t.Fatalf("Show(review-code) = %#v", detail.Component)
	}
	params := parameterNames(detail.Metadata["params"])
	if !params["code"] || !params["review_focus"] || !params["language"] {
		t.Fatalf("Show(review-code) params = %#v", params)
	}

	protocol, err := library.Show("anti-hallucination")
	if err != nil {
		t.Fatalf("Show(anti-hallucination) error: %v", err)
	}
	if len(protocol.UsedByTemplates) == 0 {
		t.Fatal("Show(anti-hallucination) has no reverse references")
	}

	detail.Metadata["name"] = "mutated"
	again, err := library.Show("review-code")
	if err != nil {
		t.Fatalf("Show(review-code) second error: %v", err)
	}
	if again.Metadata["name"] != "review-code" {
		t.Fatal("Show() returned mutable library metadata")
	}

	_, err = library.Show("not-a-component")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Show(missing) error = %v, want NotFoundError", err)
	}
}

func TestSearchRanksNaturalLanguageQueries(t *testing.T) {
	t.Parallel()

	library, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tests := []struct {
		query string
		want  string
	}{
		{query: "write a requirements document", want: "author-requirements-doc"},
		{query: "review Go code", want: "review-code"},
		{query: "find root cause of a memory leak bug", want: "investigate-bug"},
	}
	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			results := library.Search(test.query, Filter{Type: ComponentTemplate})
			if len(results) == 0 {
				t.Fatalf("Search(%q) returned no templates", test.query)
			}
			if got := results[0].Name; got != test.want {
				t.Fatalf("Search(%q) first result = %q, want %q", test.query, got, test.want)
			}
		})
	}
}

func TestSearchCompatibilityAndCopies(t *testing.T) {
	t.Parallel()

	library, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	allTemplates := library.List(Filter{Type: ComponentTemplate})
	if results := library.Search(" \t\n", Filter{Type: ComponentTemplate}); !slices.EqualFunc(results, allTemplates, func(a, b Component) bool {
		return a.Name == b.Name && a.Type == b.Type
	}) {
		t.Fatal("Search(empty) does not preserve List order")
	}
	if results := library.Search("qzjx7f20e40a", Filter{}); len(results) != 0 {
		t.Fatalf("Search(missing) returned %d results, want none", len(results))
	}

	results := library.Search("memory safety", Filter{Type: ComponentProtocol})
	if len(results) == 0 {
		t.Fatal("Search(memory safety) returned no protocols")
	}
	results[0].Metadata["name"] = "mutated"
	again := library.Search("memory safety", Filter{Type: ComponentProtocol})
	if len(again) == 0 || again[0].Metadata["name"] == "mutated" {
		t.Fatal("Search() returned mutable library metadata")
	}
}
