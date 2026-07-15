package promptkitty

import (
	"fmt"
	"strings"
)

// NotFoundError reports a requested component that is absent from the catalog.
type NotFoundError struct {
	Type ComponentType
	Name string
}

func (e *NotFoundError) Error() string {
	if e.Type == "" {
		return fmt.Sprintf("PromptKit component %q not found", e.Name)
	}

	return fmt.Sprintf("PromptKit %s %q not found", e.Type, e.Name)
}

// AmbiguousError reports a name shared by more than one component type.
type AmbiguousError struct {
	Name  string
	Types []ComponentType
}

func (e *AmbiguousError) Error() string {
	values := make([]string, 0, len(e.Types))
	for _, kind := range e.Types {
		values = append(values, string(kind))
	}

	return fmt.Sprintf("PromptKit component %q is ambiguous across: %s", e.Name, strings.Join(values, ", "))
}

// ParametersError reports missing or unknown template parameters.
type ParametersError struct {
	Template string
	Missing  []string
	Unknown  []string
}

func (e *ParametersError) Error() string {
	parts := make([]string, 0, 2)
	if len(e.Missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(e.Missing, ", "))
	}
	if len(e.Unknown) > 0 {
		parts = append(parts, "unknown: "+strings.Join(e.Unknown, ", "))
	}

	return fmt.Sprintf("PromptKit template %q parameters are invalid (%s)", e.Template, strings.Join(parts, "; "))
}
