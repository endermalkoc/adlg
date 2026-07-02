package generate

import (
	"context"
	"fmt"
	"strings"

	"github.com/endermalkoc/cusp/internal/store"
)

// RenderSpecDoc renders a single spec's document to a self-contained string, on demand from the
// database — the review surface's render chokepoint: no files are written and the output always
// reflects the current DB (on whatever branch x reads). format "html" returns an embedded,
// inline-CSS page suitable for a webview; "md"/"markdown" returns the raw Markdown page. docPath
// is the spec's canonical .md path (store.SpecDocPath).
//
// It reuses the same pipeline as the bulk `generate` — LoadDocs assembles a Model for just this
// spec (plus the shared cross-reference targets and nav, so inline links still resolve), and the
// standard renderers produce the page — so a rendered document is byte-identical to its bulk
// build, minus the site chrome.
func RenderSpecDoc(ctx context.Context, x store.Execer, specID, docPath, format string) (string, error) {
	m, err := LoadDocs(ctx, x, []string{specID}, nil)
	if err != nil {
		return "", err
	}
	return renderLoadedDoc(m, docPath, format)
}

// RenderEntityDoc is the entity analog of RenderSpecDoc: it renders a single entity's document
// on demand. docPath is the entity's canonical .md path (store.EntityDocPath).
func RenderEntityDoc(ctx context.Context, x store.Execer, entityID, docPath, format string) (string, error) {
	m, err := LoadDocs(ctx, x, nil, []string{entityID})
	if err != nil {
		return "", err
	}
	return renderLoadedDoc(m, docPath, format)
}

// renderLoadedDoc picks the document at docPath out of the Model's rendered files and returns it
// in the requested format ("md"/"markdown" → raw Markdown; "html" → the embedded, inline-CSS
// page). Shared by RenderSpecDoc and RenderEntityDoc.
func renderLoadedDoc(m *Model, docPath, format string) (string, error) {
	mdFiles, err := newMarkdownRenderer(m).Render(m)
	if err != nil {
		return "", err
	}
	var target *File
	for i := range mdFiles {
		if mdFiles[i].Path == docPath {
			target = &mdFiles[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("no rendered document at %s", docPath)
	}

	switch strings.ToLower(format) {
	case "", "md", "markdown":
		return target.Content, nil
	case "html":
		r := htmlRenderer{md: newDocMarkdown()}
		body, err := r.docBody(m, *target, m.Nav, planningHTMLBodies(m))
		if err != nil {
			return "", err
		}
		return htmlPageEmbedded(docTitle(target.Content), body), nil
	default:
		return "", fmt.Errorf("unknown format %q (want: html, md)", format)
	}
}
