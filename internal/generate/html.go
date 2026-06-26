package generate

import (
	"bytes"
	"html"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// The HTML renderer is a thin transform over the Markdown renderer (the document family
// shares one pipeline): it takes each rendered Obsidian-Markdown page, rewrites the
// Obsidian-specific bits into CommonMark goldmark understands (wikilinks → relative .html
// links, `^block` anchors → inline <a id> targets), converts to HTML, and wraps it in a
// minimal page. So HTML is "Markdown for the browser" — same structure, links that work in
// a static site instead of in Obsidian.
type htmlRenderer struct{ md goldmark.Markdown }

func newHTMLRenderer(m *Model) Renderer {
	return htmlRenderer{md: goldmark.New(
		goldmark.WithExtensions(extension.GFM),            // tables, etc.
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()), // pass through our own <a id> anchors
	)}
}

func (r htmlRenderer) Render(m *Model) ([]File, error) {
	mdFiles, err := newMarkdownRenderer(m).Render(m)
	if err != nil {
		return nil, err
	}
	out := make([]File, 0, len(mdFiles))
	for _, f := range mdFiles {
		title := docTitle(f.Content)
		gfm := obsidianToGFM(f.Content, f.Path)
		var buf bytes.Buffer
		if err := r.md.Convert([]byte(gfm), &buf); err != nil {
			return nil, err
		}
		out = append(out, File{
			Path:    strings.TrimSuffix(f.Path, ".md") + ".html",
			Content: htmlPage(title, buf.String()),
			Kind:    f.Kind,
		})
	}
	return out, nil
}

// wikilinkRe matches an Obsidian wikilink `[[target|label]]` (the table-cell form escapes
// the pipe as `\|`); label is optional.
var wikilinkRe = regexp.MustCompile(`\[\[([^|\]]+?)(?:\\?\|([^\]]+?))?\]\]`)

// anchorRe matches a trailing `^block` reference on a line (FR list items, glossary meta),
// capturing any list-item prefix so the anchor moves inside the item.
var anchorRe = regexp.MustCompile(`(?m)^(\s*(?:[-*] )?)(.*) \^([A-Za-z0-9_-]+)\s*$`)

// obsidianToGFM rewrites our Obsidian Markdown into CommonMark+GFM:
//   - YAML frontmatter is dropped (the H1 carries the title);
//   - `[[path#^anchor|label]]` → `[label](<rel>path.html#anchor)` (relative to ownerPath),
//     same-file `[[#^anchor|label]]` → `[label](#anchor)`;
//   - a trailing `^anchor` → an inline `<a id="anchor"></a>` link target.
func obsidianToGFM(md, ownerPath string) string {
	md = stripFrontmatter(md)
	rel := relPrefix(ownerPath)
	md = wikilinkRe.ReplaceAllStringFunc(md, func(s string) string {
		mm := wikilinkRe.FindStringSubmatch(s)
		target, label := mm[1], mm[2]
		if label == "" {
			label = target
		}
		return "[" + label + "](" + wikiHref(target, rel) + ")"
	})
	md = anchorRe.ReplaceAllString(md, `$1<a id="$3"></a>$2`)
	return md
}

// wikiHref turns a wikilink target into an href. rel is the `../`-prefix back to the vault
// root for cross-document links.
func wikiHref(target, rel string) string {
	switch {
	case strings.HasPrefix(target, "#^"):
		return "#" + target[2:] // same-file block reference
	case strings.HasPrefix(target, "#"):
		return target // same-file heading
	}
	pathPart, anchor := target, ""
	if i := strings.Index(target, "#^"); i >= 0 {
		pathPart, anchor = target[:i], "#"+target[i+2:]
	} else if i := strings.IndexByte(target, '#'); i >= 0 {
		pathPart, anchor = target[:i], target[i:]
	}
	return rel + pathPart + ".html" + anchor
}

// relPrefix is the `../` sequence from a document back to the vault root.
func relPrefix(ownerPath string) string {
	dir := path.Dir(ownerPath)
	if dir == "." || dir == "" {
		return ""
	}
	return strings.Repeat("../", strings.Count(dir, "/")+1)
}

func stripFrontmatter(md string) string {
	if strings.HasPrefix(md, "---\n") {
		if i := strings.Index(md[4:], "\n---\n"); i >= 0 {
			return md[4+i+5:]
		}
	}
	return md
}

// docTitle is the first H1's text (for the page <title>).
func docTitle(md string) string {
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return "ASDF"
}

func htmlPage(title, body string) string {
	return "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" +
		"<title>" + html.EscapeString(title) + "</title>\n</head>\n<body>\n" +
		body + "</body>\n</html>"
}
