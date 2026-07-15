package promptkitty

import (
	"errors"
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
