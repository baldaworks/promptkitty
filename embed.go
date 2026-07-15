package promptkitty

import "embed"

//go:generate go run ./internal/tools/syncpromptkit -lock content/upstream.json -dest content/promptkit

// embeddedContent contains the pinned PromptKit component library. The
// patterns are intentionally explicit so documentation and repository-only
// files cannot enter the runtime artifact by accident.
//
//go:embed content/promptkit/manifest.yaml
//go:embed content/promptkit/personas/*.md
//go:embed content/promptkit/protocols/*/*.md
//go:embed content/promptkit/formats/*.md
//go:embed content/promptkit/taxonomies/*.md
//go:embed content/promptkit/templates/*.md
var embeddedContent embed.FS
