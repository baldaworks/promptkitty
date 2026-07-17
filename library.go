package promptkitty

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const embeddedRoot = "content/promptkit"

type manifestDocument struct {
	Version    string                      `yaml:"version"`
	Personas   []map[string]any            `yaml:"personas"`
	Protocols  map[string][]map[string]any `yaml:"protocols"`
	Formats    []map[string]any            `yaml:"formats"`
	Taxonomies []map[string]any            `yaml:"taxonomies"`
	Templates  map[string][]map[string]any `yaml:"templates"`
	Pipelines  map[string]manifestPipeline `yaml:"pipelines"`
}

type manifestPipeline struct {
	Description string                  `yaml:"description"`
	Stages      []manifestPipelineStage `yaml:"stages"`
}

type manifestPipelineStage struct {
	Template string `yaml:"template"`
	Consumes any    `yaml:"consumes"`
	Produces any    `yaml:"produces"`
}

// Library is an immutable PromptKit catalog and assembler.
type Library struct {
	version    string
	components []Component
	byName     map[string][]int
	bodies     map[string]string
	pipelines  []Pipeline
	search     searchIndex
}

// New loads the pinned embedded PromptKit component library.
func New() (*Library, error) {
	return NewFromFS(embeddedContent, embeddedRoot)
}

// NewFromFS loads a PromptKit library rooted at root within fsys. It is useful
// for tests and callers that maintain a compatible private catalog.
func NewFromFS(fsys fs.FS, root string) (*Library, error) {
	root = path.Clean(strings.TrimSpace(root))
	if root == "" || root == "/" || !fs.ValidPath(root) {
		return nil, fmt.Errorf("invalid PromptKit filesystem root %q", root)
	}

	manifestPath := path.Join(root, "manifest.yaml")
	rawManifest, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read PromptKit manifest: %w", err)
	}

	var manifest manifestDocument
	if err := yaml.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, fmt.Errorf("parse PromptKit manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return nil, fmt.Errorf("PromptKit manifest version is required")
	}

	library := &Library{
		version: manifest.Version,
		byName:  make(map[string][]int),
		bodies:  make(map[string]string),
	}

	addEntries := func(kind ComponentType, category string, entries []map[string]any) error {
		for _, entry := range entries {
			componentPath, ok := stringValue(entry["path"])
			if !ok || strings.TrimSpace(componentPath) == "" {
				return fmt.Errorf("PromptKit %s entry has no path", kind)
			}

			raw, readErr := fs.ReadFile(fsys, path.Join(root, componentPath))
			if readErr != nil {
				return fmt.Errorf("read PromptKit component %q: %w", componentPath, readErr)
			}

			frontmatter, body, splitErr := splitMarkdownDocument(string(raw))
			if splitErr != nil {
				return fmt.Errorf("parse PromptKit component %q: %w", componentPath, splitErr)
			}

			metadata := cloneMap(entry)
			for key, value := range frontmatter {
				metadata[key] = cloneValue(value)
			}

			name, ok := stringValue(metadata["name"])
			if !ok || strings.TrimSpace(name) == "" {
				return fmt.Errorf("PromptKit component %q has no name", componentPath)
			}

			description, _ := stringValue(metadata["description"])
			language, _ := stringValue(metadata["language"])
			component := Component{
				Name:        name,
				Type:        kind,
				Category:    category,
				Path:        componentPath,
				Description: strings.TrimSpace(description),
				Language:    language,
				Metadata:    metadata,
			}
			library.components = append(library.components, component)
			library.bodies[componentPath] = body
		}

		return nil
	}

	if err := addEntries(ComponentPersona, "", manifest.Personas); err != nil {
		return nil, err
	}
	for _, category := range sortedKeys(manifest.Protocols) {
		if err := addEntries(ComponentProtocol, category, manifest.Protocols[category]); err != nil {
			return nil, err
		}
	}
	if err := addEntries(ComponentFormat, "", manifest.Formats); err != nil {
		return nil, err
	}
	if err := addEntries(ComponentTaxonomy, "", manifest.Taxonomies); err != nil {
		return nil, err
	}
	for _, category := range sortedKeys(manifest.Templates) {
		if err := addEntries(ComponentTemplate, category, manifest.Templates[category]); err != nil {
			return nil, err
		}
	}

	sort.SliceStable(library.components, func(i, j int) bool {
		left, right := library.components[i], library.components[j]
		if componentRank(left.Type) != componentRank(right.Type) {
			return componentRank(left.Type) < componentRank(right.Type)
		}
		if left.Category != right.Category {
			return left.Category < right.Category
		}

		return left.Name < right.Name
	})
	for index, component := range library.components {
		key := strings.ToLower(component.Name)
		library.byName[key] = append(library.byName[key], index)
	}
	library.search, err = newSearchIndex(library.components, library.bodies)
	if err != nil {
		return nil, fmt.Errorf("build PromptKit search index: %w", err)
	}

	for _, name := range sortedKeys(manifest.Pipelines) {
		rawPipeline := manifest.Pipelines[name]
		pipeline := Pipeline{Name: name, Description: strings.TrimSpace(rawPipeline.Description)}
		for _, stage := range rawPipeline.Stages {
			pipeline.Stages = append(pipeline.Stages, PipelineStage{
				Template: stage.Template,
				Consumes: cloneValue(stage.Consumes),
				Produces: cloneValue(stage.Produces),
			})
		}
		library.pipelines = append(library.pipelines, pipeline)
	}

	return library, nil
}

// Version returns the version declared by the embedded PromptKit manifest.
func (l *Library) Version() string {
	return l.version
}

// List returns matching components in stable type/category/name order.
func (l *Library) List(filter Filter) []Component {
	result := make([]Component, 0, len(l.components))
	for _, component := range l.components {
		if !matchesFilter(component, filter) {
			continue
		}
		result = append(result, cloneComponent(component))
	}

	return result
}

// Search ranks catalog components by their relevance to query, then applies
// filter. Search is deterministic and uses only the loaded catalog.
func (l *Library) Search(query string, filter Filter) []Component {
	if strings.TrimSpace(query) == "" {
		return l.List(filter)
	}

	indices := l.search.rank(query, l.components, filter)
	result := make([]Component, 0, len(indices))
	for _, index := range indices {
		result = append(result, cloneComponent(l.components[index]))
	}

	return result
}

// Show returns one component and the templates that reference it.
func (l *Library) Show(name string) (ComponentDetail, error) {
	indices := l.byName[strings.ToLower(strings.TrimSpace(name))]
	if len(indices) == 0 {
		return ComponentDetail{}, &NotFoundError{Name: name}
	}
	if len(indices) > 1 {
		types := make([]ComponentType, 0, len(indices))
		for _, index := range indices {
			types = append(types, l.components[index].Type)
		}
		return ComponentDetail{}, &AmbiguousError{Name: name, Types: types}
	}

	component := l.components[indices[0]]
	detail := ComponentDetail{Component: cloneComponent(component)}
	if component.Type != ComponentTemplate {
		for _, candidate := range l.components {
			if candidate.Type == ComponentTemplate && templateReferences(candidate, component) {
				detail.UsedByTemplates = append(detail.UsedByTemplates, candidate.Name)
			}
		}
		sort.Strings(detail.UsedByTemplates)
	}

	return detail, nil
}

// Pipelines returns the manifest's artifact pipelines in stable name order.
func (l *Library) Pipelines() []Pipeline {
	result := make([]Pipeline, 0, len(l.pipelines))
	for _, pipeline := range l.pipelines {
		copyPipeline := Pipeline{Name: pipeline.Name, Description: pipeline.Description}
		for _, stage := range pipeline.Stages {
			copyPipeline.Stages = append(copyPipeline.Stages, PipelineStage{
				Template: stage.Template,
				Consumes: cloneValue(stage.Consumes),
				Produces: cloneValue(stage.Produces),
			})
		}
		result = append(result, copyPipeline)
	}

	return result
}

func (l *Library) find(kind ComponentType, name string) (Component, error) {
	for _, index := range l.byName[strings.ToLower(strings.TrimSpace(name))] {
		if l.components[index].Type == kind {
			return l.components[index], nil
		}
	}

	return Component{}, &NotFoundError{Type: kind, Name: name}
}

func matchesFilter(component Component, filter Filter) bool {
	if filter.Type != "" && component.Type != filter.Type {
		return false
	}
	if filter.Category != "" && !strings.EqualFold(component.Category, filter.Category) {
		return false
	}
	if filter.Language != "" && !strings.EqualFold(component.Language, filter.Language) {
		return false
	}

	return true
}

func templateReferences(template, target Component) bool {
	switch target.Type {
	case ComponentPersona:
		persona, _ := stringValue(template.Metadata["persona"])
		return strings.EqualFold(componentShortName(persona), target.Name)
	case ComponentProtocol:
		return containsFold(shortNames(stringSlice(template.Metadata["protocols"])), target.Name)
	case ComponentFormat:
		format, _ := stringValue(template.Metadata["format"])
		return strings.EqualFold(format, target.Name)
	case ComponentTaxonomy:
		return containsFold(shortNames(stringSlice(template.Metadata["taxonomies"])), target.Name)
	default:
		return false
	}
}

func cloneComponent(component Component) Component {
	component.Metadata = cloneMap(component.Metadata)

	return component
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneValue(value)
	}

	return result
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		result := make([]any, len(typed))
		for index := range typed {
			result[index] = cloneValue(typed[index])
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}

func stringValue(value any) (string, bool) {
	text, ok := value.(string)

	return text, ok
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func componentShortName(name string) string {
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		return name[slash+1:]
	}

	return name
}

func shortNames(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, componentShortName(value))
	}

	return result
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}

	return false
}

func sortedKeys[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)

	return result
}

func componentRank(kind ComponentType) int {
	switch kind {
	case ComponentPersona:
		return 0
	case ComponentProtocol:
		return 1
	case ComponentFormat:
		return 2
	case ComponentTaxonomy:
		return 3
	case ComponentTemplate:
		return 4
	default:
		return 5
	}
}
