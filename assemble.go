package promptkitty

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var promptKitExpression = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_-]+)\s*\}\}`)

// Assemble resolves a template and returns a fully rendered prompt. Mustache
// examples that are not declared template parameters are ordinary Markdown
// literals and are preserved verbatim.
func (l *Library) Assemble(request AssembleRequest) (AssembleResult, error) {
	template, err := l.find(ComponentTemplate, request.Template)
	if err != nil {
		return AssembleResult{}, err
	}

	declared := parameterNames(template.Metadata["params"])
	if err := validateParameters(template.Name, declared, request.Params); err != nil {
		return AssembleResult{}, err
	}

	personaName, _ := stringValue(template.Metadata["persona"])
	if request.Persona != "" {
		personaName = request.Persona
	} else if personaName == "configurable" || promptKitExpression.MatchString(personaName) {
		personaName = request.Params["persona"]
	}
	if strings.TrimSpace(personaName) == "" || personaName == "configurable" || promptKitExpression.MatchString(personaName) {
		return AssembleResult{}, fmt.Errorf("PromptKit template %q requires an embedded persona override", template.Name)
	}

	persona, err := l.find(ComponentPersona, componentShortName(personaName))
	if err != nil {
		return AssembleResult{}, fmt.Errorf("resolve persona for template %q: %w", template.Name, err)
	}

	protocolNames := shortNames(stringSlice(template.Metadata["protocols"]))
	protocolNames = stableUnique(append(protocolNames, shortNames(request.AdditionalProtocols)...))
	protocols := make([]Component, 0, len(protocolNames))
	for _, name := range protocolNames {
		protocol, findErr := l.find(ComponentProtocol, name)
		if findErr != nil {
			return AssembleResult{}, fmt.Errorf("resolve protocol for template %q: %w", template.Name, findErr)
		}
		protocols = append(protocols, protocol)
	}

	taxonomyNames := shortNames(stringSlice(template.Metadata["taxonomies"]))
	taxonomyNames = stableUnique(append(taxonomyNames, shortNames(request.AdditionalTaxonomies)...))
	taxonomies := make([]Component, 0, len(taxonomyNames))
	for _, name := range taxonomyNames {
		taxonomy, findErr := l.find(ComponentTaxonomy, name)
		if findErr != nil {
			return AssembleResult{}, fmt.Errorf("resolve taxonomy for template %q: %w", template.Name, findErr)
		}
		taxonomies = append(taxonomies, taxonomy)
	}

	formatName, _ := stringValue(template.Metadata["format"])
	if request.Format != nil {
		formatName = *request.Format
	}
	var format *Component
	if strings.TrimSpace(formatName) != "" {
		resolved, findErr := l.find(ComponentFormat, componentShortName(formatName))
		if findErr != nil {
			return AssembleResult{}, fmt.Errorf("resolve format for template %q: %w", template.Name, findErr)
		}
		format = &resolved
	}

	sections := make([]string, 0, 5)
	components := make([]ComponentRef, 0, 2+len(protocols)+len(taxonomies))
	appendSection := func(header string, component Component, body string) {
		sections = append(sections, header+"\n\n"+body)
		components = append(components, componentReference(component))
	}

	appendSection("# Identity", persona, l.bodies[persona.Path])
	if len(protocols) > 0 {
		bodies := make([]string, 0, len(protocols))
		for _, protocol := range protocols {
			bodies = append(bodies, l.bodies[protocol.Path])
			components = append(components, componentReference(protocol))
		}
		sections = append(sections, "# Reasoning Protocols\n\n"+strings.Join(bodies, "\n\n---\n\n"))
	}
	if len(taxonomies) > 0 {
		bodies := make([]string, 0, len(taxonomies))
		for _, taxonomy := range taxonomies {
			bodies = append(bodies, l.bodies[taxonomy.Path])
			components = append(components, componentReference(taxonomy))
		}
		sections = append(sections, "# Classification Taxonomy\n\n"+strings.Join(bodies, "\n\n---\n\n"))
	}
	if format != nil {
		appendSection("# Output Format", *format, l.bodies[format.Path])
	}

	taskBody := renderParameters(l.bodies[template.Path], declared, request.Params)
	appendSection("# Task", template, taskBody)

	return AssembleResult{
		Markdown:   strings.Join(sections, "\n\n---\n\n"),
		Template:   cloneComponent(template),
		Components: components,
	}, nil
}

func parameterNames(value any) map[string]bool {
	result := make(map[string]bool)
	if values, ok := value.(map[string]any); ok {
		for name := range values {
			result[name] = true
		}

		return result
	}

	return result
}

func validateParameters(template string, declared map[string]bool, provided map[string]string) error {
	missing := make([]string, 0)
	for name := range declared {
		if _, ok := provided[name]; !ok {
			missing = append(missing, name)
		}
	}
	unknown := make([]string, 0)
	for name := range provided {
		if !declared[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(unknown)
	if len(missing) > 0 || len(unknown) > 0 {
		return &ParametersError{Template: template, Missing: missing, Unknown: unknown}
	}

	return nil
}

func renderParameters(body string, declared map[string]bool, params map[string]string) string {
	return promptKitExpression.ReplaceAllStringFunc(body, func(expression string) string {
		match := promptKitExpression.FindStringSubmatch(expression)
		if len(match) != 2 || !declared[match[1]] {
			return expression
		}

		return params[match[1]]
	})
}

func stableUnique(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}

	return result
}

func componentReference(component Component) ComponentRef {
	return ComponentRef{Name: component.Name, Type: component.Type, Path: component.Path}
}
