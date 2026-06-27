package generate

import (
	"fmt"
	"html"
	"strings"
)

// Rich HTML rendering for the planning pages. Unlike specs/entities (Markdown → GFM), the
// planning pages are rendered straight from the Model so they can use enum pills, type icons,
// and a real capability hierarchy tree — things Markdown tables can't express. The Markdown
// planning pages stay plain (nested lists + tables) for Obsidian.

// planningHTMLBodies returns the rich HTML <article> body for each planning page, keyed by the
// page's Markdown path (so the HTML renderer can substitute it for the GFM-converted body).
func planningHTMLBodies(m *Model) map[string]string {
	bodies := map[string]string{}
	if len(m.Capabilities) == 0 && len(m.Deliverables) == 0 && len(m.Views) == 0 {
		return bodies
	}
	bodies["planning/index.md"] = planningIndexHTML(m)
	if len(m.Capabilities) > 0 {
		bodies["planning/capabilities.md"] = capabilitiesHTML(m.Capabilities)
	}
	if len(m.Deliverables) > 0 {
		bodies["planning/deliverables.md"] = deliverablesHTML(m.Deliverables)
	}
	if len(m.Views) > 0 {
		bodies["planning/views.md"] = viewsHTML(m.Views)
	}
	return bodies
}

// ---- capability hierarchy --------------------------------------------------

// capTreeNode is one node in the capability forest (a capability + its sub-capabilities).
type capTreeNode struct {
	Cap      *Capability
	Children []*capTreeNode
}

// buildCapForest assembles the capability hierarchy (via parent_id) and returns the root
// nodes grouped by domain slug, plus the domain slugs in sorted order. Input order (caps is
// sorted by title) is preserved among siblings, so output is deterministic.
func buildCapForest(caps []*Capability) (map[string][]*capTreeNode, []string) {
	node := make(map[string]*capTreeNode, len(caps))
	for _, c := range caps {
		node[c.ID] = &capTreeNode{Cap: c}
	}
	rootsByDomain := map[string][]*capTreeNode{}
	for _, c := range caps {
		n := node[c.ID]
		if c.ParentID != "" {
			if p := node[c.ParentID]; p != nil {
				p.Children = append(p.Children, n)
				continue
			}
		}
		d := c.DomainSlug
		if d == "" {
			d = "unassigned"
		}
		rootsByDomain[d] = append(rootsByDomain[d], n)
	}
	return rootsByDomain, sortedKeys(rootsByDomain)
}

func capabilitiesHTML(caps []*Capability) string {
	rootsByDomain, domains := buildCapForest(caps)
	var b strings.Builder
	b.WriteString("<h1>Capabilities</h1>\n")
	b.WriteString(`<p class="lede">Product capabilities as a hierarchy — domain › epic › capability.</p>` + "\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "<h2>%s%s</h2>\n", icon("domain"), html.EscapeString(humanizeSegment(d)))
		b.WriteString(`<ul class="cap-tree">`)
		for _, n := range rootsByDomain[d] {
			writeCapNodeHTML(&b, n)
		}
		b.WriteString("</ul>\n")
	}
	return b.String()
}

func writeCapNodeHTML(b *strings.Builder, n *capTreeNode) {
	c := n.Cap
	b.WriteString(`<li><div class="cap-node">`)
	b.WriteString(icon("capability"))
	b.WriteString(`<span class="cap-title">` + html.EscapeString(c.Title) + `</span>`)
	b.WriteString(levelPill(c.Level))
	for _, ms := range c.Milestones {
		b.WriteString(pill("pill-ms", "flag", ms))
	}
	if n := len(c.Deliverables); n > 0 {
		b.WriteString(countPill("deliverable", n, "deliverable", c.Deliverables))
	}
	b.WriteString(`</div>`)
	if len(n.Children) > 0 {
		b.WriteString(`<ul>`)
		for _, ch := range n.Children {
			writeCapNodeHTML(b, ch)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</li>`)
}

// ---- deliverables ----------------------------------------------------------

func deliverablesHTML(delivs []*Deliverable) string {
	byMilestone := map[string][]*Deliverable{}
	for _, d := range delivs {
		ms := d.Milestone
		if ms == "" {
			ms = "Unscheduled"
		}
		byMilestone[ms] = append(byMilestone[ms], d)
	}
	var b strings.Builder
	b.WriteString("<h1>Deliverables</h1>\n")
	b.WriteString(`<p class="lede">Units of work, grouped by milestone.</p>` + "\n")
	for _, ms := range sortedKeys(byMilestone) {
		fmt.Fprintf(&b, "<h2>%s%s</h2>\n", icon("flag"), html.EscapeString(ms))
		b.WriteString(`<ul class="plan-list">`)
		for _, d := range byMilestone[ms] {
			b.WriteString(`<li class="plan-item"><div class="plan-item-head">`)
			b.WriteString(icon("deliverable"))
			b.WriteString(`<span class="plan-title">` + html.EscapeString(d.Title) + `</span>`)
			if d.Size != "" {
				b.WriteString(pill("pill-size", "", d.Size))
			}
			if d.Status != "" {
				b.WriteString(pill("pill-status pill-status-"+d.Status, "", titleStatus(d.Status)))
			}
			b.WriteString(aiPill(d.AIReady))
			b.WriteString(`</div>`)
			rels := relGroups(
				relGroup("capability", "Capabilities", d.Capabilities),
				relGroup("view", "Views", d.Views),
				relGroup("blocked", "Blocked by", d.BlockedBy),
			)
			b.WriteString(rels)
			b.WriteString(`</li>`)
		}
		b.WriteString("</ul>\n")
	}
	return b.String()
}

// ---- views -----------------------------------------------------------------

func viewsHTML(views []*View) string {
	rel := relPrefix("planning/views.md") // ../ back to the site root for spec links
	byDomain := map[string][]*View{}
	for _, v := range views {
		d := v.DomainSlug
		if d == "" {
			d = "unassigned"
		}
		byDomain[d] = append(byDomain[d], v)
	}
	var b strings.Builder
	b.WriteString("<h1>Views</h1>\n")
	b.WriteString(`<p class="lede">UI surfaces, grouped by domain, with their route and backing spec.</p>` + "\n")
	for _, d := range sortedKeys(byDomain) {
		fmt.Fprintf(&b, "<h2>%s%s</h2>\n", icon("domain"), html.EscapeString(humanizeSegment(d)))
		b.WriteString(`<ul class="plan-list">`)
		for _, v := range byDomain[d] {
			b.WriteString(`<li class="plan-item"><div class="plan-item-head">`)
			b.WriteString(icon("view"))
			b.WriteString(`<span class="plan-title">` + html.EscapeString(v.Title) + `</span>`)
			if v.SpecPath != "" {
				label := v.SpecTitle
				if label == "" {
					label = strings.TrimSuffix(v.SpecPath, ".md")
				}
				href := rel + strings.TrimSuffix(v.SpecPath, ".md") + ".html"
				b.WriteString(`<a class="xref" href="` + href + `">` + icon("spec") + html.EscapeString(label) + `</a>`)
			}
			b.WriteString(`</div>`)
			if v.Route != "" {
				b.WriteString(`<div class="plan-item-sub"><code>` + html.EscapeString(v.Route) + `</code></div>`)
			}
			b.WriteString(relGroups(relGroup("deliverable", "Deliverables", v.Deliverables)))
			b.WriteString(`</li>`)
		}
		b.WriteString("</ul>\n")
	}
	return b.String()
}

// ---- index -----------------------------------------------------------------

func planningIndexHTML(m *Model) string {
	var b strings.Builder
	b.WriteString("<h1>Planning</h1>\n")
	b.WriteString(`<p class="lede">What to build — capabilities, deliverables, and views, and how they relate.</p>` + "\n")
	b.WriteString(`<div class="plan-cards">`)
	card := func(href, kind string, n int, label string) {
		fmt.Fprintf(&b, `<a class="plan-card" href="%s"><span class="plan-card-icon">%s</span><span class="plan-card-n">%d</span><span class="plan-card-label">%s</span></a>`,
			href, icon(kind), n, label)
	}
	if n := len(m.Capabilities); n > 0 {
		card("capabilities.html", "capability", n, "Capabilities")
	}
	if n := len(m.Deliverables); n > 0 {
		card("deliverables.html", "deliverable", n, "Deliverables")
	}
	if n := len(m.Views); n > 0 {
		card("views.html", "view", n, "Views")
	}
	b.WriteString("</div>\n")
	return b.String()
}

// ---- pills & relationship groups -------------------------------------------

// pill renders a small rounded label with an optional leading icon.
func pill(class, iconName, text string) string {
	ic := ""
	if iconName != "" {
		ic = icon(iconName)
	}
	return `<span class="pill ` + class + `">` + ic + html.EscapeString(text) + `</span>`
}

// levelPill renders a capability's level (domain|epic|capability) as a graded pill.
func levelPill(level string) string {
	if level == "" {
		return ""
	}
	return `<span class="pill pill-level pill-level-` + html.EscapeString(level) + `">` + html.EscapeString(level) + `</span>`
}

// aiPill renders the ai_ready enum as a yes/no/na pill (with a check / x / dash icon).
func aiPill(ai string) string {
	switch ai {
	case "yes":
		return pill("pill-ai-yes", "check", "AI-ready")
	case "no":
		return pill("pill-ai-no", "x", "Not AI-ready")
	case "na":
		return pill("pill-ai-na", "minus", "AI N/A")
	}
	return ""
}

// countPill renders a count with a type icon, titled with the full list for hover detail.
func countPill(iconName string, n int, noun string, items []string) string {
	title := strings.Join(items, ", ")
	plural := noun + "s"
	if n == 1 {
		plural = noun
	}
	return `<span class="pill pill-count" title="` + html.EscapeString(title) + `">` +
		icon(iconName) + fmt.Sprintf("%d %s", n, plural) + `</span>`
}

// relGroup renders one labeled relationship group (icon + comma-joined titles), or "" when empty.
func relGroup(iconName, label string, items []string) string {
	if len(items) == 0 {
		return ""
	}
	return `<span class="rel-group">` + icon(iconName) +
		`<span><span class="rel-label">` + html.EscapeString(label) + `:</span> ` +
		html.EscapeString(strings.Join(items, ", ")) + `</span></span>`
}

// relGroups wraps non-empty relationship groups in a row (or "" when all are empty).
func relGroups(groups ...string) string {
	var nonEmpty []string
	for _, g := range groups {
		if g != "" {
			nonEmpty = append(nonEmpty, g)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return `<div class="plan-item-rels">` + strings.Join(nonEmpty, "") + `</div>`
}
