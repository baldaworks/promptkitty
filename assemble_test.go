package promptkitty

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssembleStaticTemplate(t *testing.T) {
	t.Parallel()

	library := mustLibrary(t)
	result, err := library.Assemble(AssembleRequest{
		Template: "investigate-bug",
		Params: map[string]string{
			"problem_description": "Parser crashes on empty input",
			"code_context":        "src/parser.c",
			"environment":         "Linux amd64",
		},
	})
	if err != nil {
		t.Fatalf("Assemble() error: %v", err)
	}

	for _, want := range []string{
		"# Identity", "# Reasoning Protocols", "# Classification Taxonomy",
		"# Output Format", "# Task", "Parser crashes on empty input",
	} {
		if !strings.Contains(result.Markdown, want) {
			t.Errorf("assembled Markdown is missing %q", want)
		}
	}
	for _, unwanted := range []string{"{{problem_description}}", "{{code_context}}", "{{environment}}"} {
		if strings.Contains(result.Markdown, unwanted) {
			t.Errorf("assembled Markdown contains active placeholder %q", unwanted)
		}
	}
	if len(result.Components) < 5 || result.Components[len(result.Components)-1].Name != "investigate-bug" {
		t.Fatalf("assembled components = %#v", result.Components)
	}
}

func TestAssembleConfigurablePersonaNullFormatAndOverrides(t *testing.T) {
	t.Parallel()

	library := mustLibrary(t)
	result, err := library.Assemble(AssembleRequest{
		Template: "engineering-workflow",
		Params: map[string]string{
			"persona":            "systems-engineer",
			"project_name":       "Promptkitty",
			"change_description": "Add deterministic assembly",
			"existing_artifacts": "None",
			"context":            "Go library",
		},
		AdditionalProtocols: []string{"memory-safety-c", "memory-safety-c"},
	})
	if err != nil {
		t.Fatalf("Assemble() error: %v", err)
	}
	for _, component := range result.Components {
		if component.Type == ComponentFormat {
			t.Fatalf("format:null template contains format component %#v", component)
		}
	}
	if got := strings.Count(result.Markdown, "# Protocol: Memory Safety Analysis (C)"); got != 1 {
		t.Fatalf("additional protocol occurrences = %d, want 1", got)
	}
}

func TestAssembleOmitFormatAndPreserveLiteralMustache(t *testing.T) {
	t.Parallel()

	fixture := fstest.MapFS{
		"manifest.yaml": {Data: []byte(`version: "test"
personas:
  - name: reviewer
    path: personas/reviewer.md
protocols: {}
formats:
  - name: report
    path: formats/report.md
taxonomies: []
templates:
  tests:
    - name: review
      path: templates/review.md
      persona: reviewer
      protocols: []
      format: report
`)},
		"personas/reviewer.md": {Data: []byte("---\nname: reviewer\n---\nYou review things.")},
		"formats/report.md":    {Data: []byte("---\nname: report\ntype: format\n---\nShow examples like `{{literal}}`.")},
		"templates/review.md":  {Data: []byte("---\nname: review\npersona: reviewer\nprotocols: []\nformat: report\nparams:\n  task: Task\n---\nReview {{task}} and preserve {{example}}.")},
	}
	library, err := NewFromFS(fixture, ".")
	if err != nil {
		t.Fatalf("NewFromFS() error: %v", err)
	}

	omit := ""
	result, err := library.Assemble(AssembleRequest{
		Template: "review",
		Params:   map[string]string{"task": "{{ prompt }}"},
		Format:   &omit,
	})
	if err != nil {
		t.Fatalf("Assemble() error: %v", err)
	}
	if strings.Contains(result.Markdown, "# Output Format") {
		t.Fatal("explicit empty format did not omit the format")
	}
	if !strings.Contains(result.Markdown, "Review {{ prompt }} and preserve {{example}}.") {
		t.Fatalf("one-pass render changed literals: %s", result.Markdown)
	}
}

func TestAssembleParameterErrors(t *testing.T) {
	t.Parallel()

	library := mustLibrary(t)
	_, err := library.Assemble(AssembleRequest{
		Template: "investigate-bug",
		Params: map[string]string{
			"problem_description": "broken",
			"extra":               "unknown",
		},
	})
	var parameterError *ParametersError
	if !errors.As(err, &parameterError) {
		t.Fatalf("Assemble() error = %v, want ParametersError", err)
	}
	if len(parameterError.Missing) != 2 || len(parameterError.Unknown) != 1 {
		t.Fatalf("ParametersError = %#v", parameterError)
	}
}

func TestEveryEmbeddedTemplateAssembles(t *testing.T) {
	t.Parallel()

	library := mustLibrary(t)
	for _, template := range library.List(Filter{Type: ComponentTemplate}) {
		t.Run(template.Name, func(t *testing.T) {
			t.Parallel()

			params := make(map[string]string)
			for name := range parameterNames(template.Metadata["params"]) {
				params[name] = "value for " + name
			}
			if _, ok := params["persona"]; ok {
				params["persona"] = "systems-engineer"
			}

			request := AssembleRequest{Template: template.Name, Params: params}
			persona, _ := stringValue(template.Metadata["persona"])
			if persona == "configurable" || promptKitExpression.MatchString(persona) {
				request.Persona = "systems-engineer"
			}

			result, err := library.Assemble(request)
			if err != nil {
				t.Fatalf("Assemble(%s) error: %v", template.Name, err)
			}
			if strings.TrimSpace(result.Markdown) == "" {
				t.Fatal("Assemble() returned empty Markdown")
			}
			for name := range params {
				if strings.Contains(result.Markdown, "{{"+name+"}}") {
					t.Errorf("Assemble() left declared parameter %q", name)
				}
			}
		})
	}
}

func mustLibrary(t *testing.T) *Library {
	t.Helper()

	library, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return library
}
