package promptkitty

// ComponentType identifies one PromptKit component layer.
type ComponentType string

const (
	ComponentPersona  ComponentType = "persona"
	ComponentProtocol ComponentType = "protocol"
	ComponentFormat   ComponentType = "format"
	ComponentTaxonomy ComponentType = "taxonomy"
	ComponentTemplate ComponentType = "template"
)

// Filter selects catalog components. Empty fields match every component.
type Filter struct {
	Type     ComponentType
	Category string
	Language string
}

// Component is a catalog entry enriched with its Markdown frontmatter.
// Metadata is a defensive copy and may be inspected by future transports.
type Component struct {
	Name        string         `json:"name"`
	Type        ComponentType  `json:"type"`
	Category    string         `json:"category,omitempty"`
	Path        string         `json:"path,omitempty"`
	Description string         `json:"description,omitempty"`
	Language    string         `json:"language,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ComponentDetail adds reverse template references to a component.
type ComponentDetail struct {
	Component

	UsedByTemplates []string `json:"usedByTemplates,omitempty"`
}

// Pipeline is one named sequence from the embedded PromptKit manifest.
type Pipeline struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Stages      []PipelineStage `json:"stages"`
}

// PipelineStage describes one template and its artifact contract.
type PipelineStage struct {
	Template string `json:"template"`
	Consumes any    `json:"consumes,omitempty"`
	Produces any    `json:"produces,omitempty"`
}

// AssembleRequest selects a template, supplies every declared parameter, and
// optionally adjusts its component composition.
type AssembleRequest struct {
	Template             string
	Params               map[string]string
	Persona              string
	AdditionalProtocols  []string
	AdditionalTaxonomies []string
	// Format leaves the template default when nil, omits the format when it
	// points to an empty string, and otherwise replaces the default.
	Format *string
}

// ComponentRef records one component included in an assembled prompt.
type ComponentRef struct {
	Name string        `json:"name"`
	Type ComponentType `json:"type"`
	Path string        `json:"path"`
}

// AssembleResult is a completely rendered PromptKit prompt. All declared
// template parameters have been substituted.
type AssembleResult struct {
	Markdown   string         `json:"markdown"`
	Template   Component      `json:"template"`
	Components []ComponentRef `json:"components"`
}
