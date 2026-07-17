package promptkitty

import (
	"testing"
	"testing/fstest"
)

func TestSearchIndexesBodiesAndMetadataDeterministically(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"catalog/manifest.yaml": {Data: []byte(`
version: test
templates:
  examples:
    - path: templates/alpha.md
    - path: templates/beta.md
    - path: templates/gamma.md
`)},
		"catalog/templates/alpha.md": {Data: []byte(`---
name: alpha
description: first candidate
labels:
  intent: spectral analysis
---
Inspect a quartz oscillator.`)},
		"catalog/templates/beta.md": {Data: []byte(`---
name: beta
description: second candidate
labels:
  intent: ordinary review
---
Inspect a ceramic package.`)},
		"catalog/templates/gamma.md": {Data: []byte(`---
name: gamma
description: third candidate
labels:
  intent: ordinary review
---
Inspect another ceramic package.`)},
	}

	library, err := NewFromFS(fsys, "catalog")
	if err != nil {
		t.Fatalf("NewFromFS() error: %v", err)
	}

	for query, want := range map[string]string{
		"quartz oscillator": "alpha",
		"spectral-analysis": "alpha",
	} {
		results := library.Search(query, Filter{Type: ComponentTemplate})
		if len(results) == 0 || results[0].Name != want {
			t.Fatalf("Search(%q) = %#v, want %q first", query, results, want)
		}
	}

	results := library.Search("ceramic", Filter{Type: ComponentTemplate})
	if len(results) != 2 || results[0].Name != "beta" || results[1].Name != "gamma" {
		t.Fatalf("Search(ceramic) order = %#v, want beta then gamma", results)
	}
	for _, result := range results {
		if result.Type != ComponentTemplate {
			t.Fatalf("Search(ceramic) returned type %q, want template", result.Type)
		}
	}
}

func TestTokenizeSearchText(t *testing.T) {
	t.Parallel()

	want := []string{"prompt", "kitty", "поиск", "2026"}
	got := tokenizeSearchText("Prompt_Kitty: поиск—2026")
	if len(got) != len(want) {
		t.Fatalf("tokenizeSearchText() = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("tokenizeSearchText()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
