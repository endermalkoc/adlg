package generate

import (
	"encoding/json"
	"strings"
)

// The JSON serializer: Model → structured records (the data family). Unlike the document
// renderers it does NOT resolve inline [[TYPE:key]] refs — prose fields keep their
// canonical tokens, which a consumer resolves via the entity-ref graph. It mirrors the
// document tree (a .json per spec/entity) and adds a root index.json discovery manifest.
type jsonRenderer struct{}

func (jsonRenderer) Render(m *Model) ([]File, error) {
	var files []File
	for _, sp := range m.Specs {
		c, err := toJSON(sp)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: jsonPath(sp.Path), Content: c, Kind: "spec"})
	}
	for _, e := range m.Entities {
		c, err := toJSON(e)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: jsonPath(e.DocPath), Content: c, Kind: "entity"})
	}
	c, err := toJSON(buildManifest(m))
	if err != nil {
		return nil, err
	}
	files = append(files, File{Path: "index.json", Content: c, Kind: "index"})
	if len(m.Terms) > 0 {
		c, err := toJSON(m.Terms)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: "glossary.json", Content: c, Kind: "glossary"})
	}
	return files, nil
}

// Manifest is the root index.json: a discovery listing of every document and where to
// fetch its JSON, so an agent can enumerate the graph then pull individual records.
type Manifest struct {
	Domains  []*Domain     `json:"domains"`
	Specs    []ManifestRef `json:"specs"`
	Entities []ManifestRef `json:"entities"`
	Glossary bool          `json:"glossary"`
}

// ManifestRef is a lightweight pointer to a document's JSON file.
type ManifestRef struct {
	Key    string `json:"key"`             // spec prefix or entity name
	Title  string `json:"title,omitempty"` // spec title
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path"` // path to the document's .json
}

func buildManifest(m *Model) Manifest {
	man := Manifest{Domains: m.Domains, Glossary: len(m.Terms) > 0}
	for _, sp := range m.Specs {
		key := sp.Prefix
		if key == "" {
			key = sp.Slug
		}
		man.Specs = append(man.Specs, ManifestRef{Key: key, Title: sp.Title, Domain: sp.Domain, Path: jsonPath(sp.Path)})
	}
	for _, e := range m.Entities {
		man.Entities = append(man.Entities, ManifestRef{Key: e.Name, Path: jsonPath(e.DocPath)})
	}
	return man
}

// jsonPath swaps a document's .md path for .json.
func jsonPath(docPath string) string {
	return strings.TrimSuffix(docPath, ".md") + ".json"
}

func toJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
