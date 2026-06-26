package generate

import (
	"context"

	"github.com/endermalkoc/asdf/internal/refs"
	"github.com/endermalkoc/asdf/internal/store"
)

// Model is the format-agnostic view of the canonical graph that every renderer (Markdown,
// JSON, HTML, …) consumes. It is assembled once by Load, so a renderer never touches the
// store: assembly and formatting are separate concerns. The JSON tags double as the JSON
// serialization shape.
type Model struct {
	Domains    []*Domain      `json:"domains"`
	Specs      []*Spec        `json:"specs"`
	Entities   []*Entity      `json:"entities"`
	Terms      []*Term        `json:"glossary,omitempty"`
	Targets    []refs.Target  `json:"-"` // inline-ref resolution (doc renderers); not serialized
	Priorities map[int]string `json:"-"` // level → label
}

// Domain is a top-level grouping of specs.
type Domain struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
}

// Spec is a feature specification with its prose sections, stories, and requirements.
// Path is the full document path (`<domain>/[dir/]<slug>.md`); renderers swap the
// extension as needed.
type Spec struct {
	Prefix       string         `json:"prefix,omitempty"`
	Slug         string         `json:"slug,omitempty"`
	Title        string         `json:"title,omitempty"`
	Domain       string         `json:"domain"`
	Status       string         `json:"status"`
	Path         string         `json:"path"`
	Sections     []*Section     `json:"sections,omitempty"`
	Stories      []*Story       `json:"user_stories,omitempty"`
	Groups       []*Group       `json:"requirement_groups,omitempty"`
	Requirements []*Requirement `json:"requirements,omitempty"`
}

// Section is one curated prose section (overview, edge_cases, …).
type Section struct {
	Key      string `json:"key"`
	Title    string `json:"title,omitempty"`
	Level    int    `json:"level"`
	Position int    `json:"position"`
	Body     string `json:"body,omitempty"`
}

// Story is a user story plus its acceptance scenarios.
type Story struct {
	Position        int         `json:"position"`
	Title           string      `json:"title,omitempty"`
	Priority        int         `json:"priority"`
	AsA             string      `json:"as_a,omitempty"`
	IWant           string      `json:"i_want,omitempty"`
	SoThat          string      `json:"so_that,omitempty"`
	Narrative       string      `json:"narrative,omitempty"`
	WhyPriority     string      `json:"why_priority,omitempty"`
	IndependentTest string      `json:"independent_test,omitempty"`
	Scenarios       []*Scenario `json:"scenarios,omitempty"`
}

// Scenario is one Given/When/Then acceptance scenario.
type Scenario struct {
	Position int    `json:"position"`
	Given    string `json:"given,omitempty"`
	When     string `json:"when,omitempty"`
	Then     string `json:"then,omitempty"`
}

// Group is an FR group sub-header (title + interspersed note) over a spec's FR list.
type Group struct {
	ID       string `json:"-"`
	Position int    `json:"position"`
	Title    string `json:"title"`
	Notes    string `json:"notes,omitempty"`
}

// Requirement is one functional requirement. GroupID links it to a Group; MD renders only
// FRKey/Statement/GroupID, but the model carries the full row for richer data formats.
type Requirement struct {
	FRKey          string `json:"fr_key"`
	Number         int    `json:"number"`
	Suffix         string `json:"suffix,omitempty"`
	GroupID        string `json:"-"`
	Position       int    `json:"position,omitempty"`
	Statement      string `json:"statement,omitempty"`
	DeliveryStatus string `json:"delivery_status,omitempty"`
	Milestone      string `json:"milestone,omitempty"`
}

// Entity is a first-class shared-entity document with its prose sections.
type Entity struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	DocPath     string     `json:"path"`
	Sections    []*Section `json:"sections,omitempty"`
}

// Term is a glossary term.
type Term struct {
	Slug       string   `json:"slug"`
	Term       string   `json:"term,omitempty"`
	Definition string   `json:"definition,omitempty"`
	DomainSlug string   `json:"domain,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// Load assembles the whole graph from the store into a Model. This is the single place
// that queries; every renderer reads the returned structs.
func Load(ctx context.Context, x store.Execer) (*Model, error) {
	m := &Model{}

	domains, err := store.ListDomains(ctx, x)
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		m.Domains = append(m.Domains, &Domain{Slug: d.Slug, Name: d.Name, Description: d.Description, Status: d.Status})
	}

	specRows, err := store.ListSpecs(ctx, x)
	if err != nil {
		return nil, err
	}
	for _, sr := range specRows {
		sp := &Spec{
			Prefix: sr.Prefix, Slug: sr.Slug, Title: sr.Title, Domain: sr.DomainSlug, Status: sr.Status,
			Path: store.SpecDocPath(sr.DomainSlug, sr.Path, sr.Slug),
		}
		secRows, err := store.ListSpecSections(ctx, x, sr.ID)
		if err != nil {
			return nil, err
		}
		sp.Sections = toSections(secRows)
		storyRows, err := store.ListStoriesBySpec(ctx, x, sr.ID)
		if err != nil {
			return nil, err
		}
		for _, st := range storyRows {
			s := &Story{
				Position: st.Position, Title: st.Title, Priority: st.Priority, AsA: st.AsA,
				IWant: st.IWant, SoThat: st.SoThat, Narrative: st.Narrative,
				WhyPriority: st.WhyPriority, IndependentTest: st.IndependentTest,
			}
			scnRows, err := store.ListScenariosByStory(ctx, x, st.ID)
			if err != nil {
				return nil, err
			}
			for _, sc := range scnRows {
				s.Scenarios = append(s.Scenarios, &Scenario{Position: sc.Position, Given: sc.Given, When: sc.When, Then: sc.Then})
			}
			sp.Stories = append(sp.Stories, s)
		}
		groupRows, err := store.ListReqGroups(ctx, x, sr.ID)
		if err != nil {
			return nil, err
		}
		for _, g := range groupRows {
			sp.Groups = append(sp.Groups, &Group{ID: g.ID, Position: g.Position, Title: g.Title, Notes: g.Notes})
		}
		reqRows, err := store.ListReqsBySpecID(ctx, x, sr.ID)
		if err != nil {
			return nil, err
		}
		for _, r := range reqRows {
			sp.Requirements = append(sp.Requirements, &Requirement{
				FRKey: r.FRKey, Number: r.Number, Suffix: r.Suffix, GroupID: r.GroupID,
				Position: r.Position, Statement: r.Statement, DeliveryStatus: r.DeliveryStatus, Milestone: r.Milestone,
			})
		}
		m.Specs = append(m.Specs, sp)
	}

	entityRows, err := store.ListEntities(ctx, x)
	if err != nil {
		return nil, err
	}
	for _, er := range entityRows {
		e := &Entity{Name: er.Name, Description: er.Description, Status: er.Status, DocPath: er.DocPath}
		secRows, err := store.ListEntitySections(ctx, x, er.ID)
		if err != nil {
			return nil, err
		}
		e.Sections = toSections(secRows)
		m.Entities = append(m.Entities, e)
	}

	termRows, err := store.ListGlossaryTerms(ctx, x)
	if err != nil {
		return nil, err
	}
	for _, t := range termRows {
		m.Terms = append(m.Terms, &Term{Slug: t.Slug, Term: t.Term, Definition: t.Definition, DomainSlug: t.DomainSlug, Aliases: t.Aliases})
	}

	targets, err := store.ListRefTargets(ctx, x)
	if err != nil {
		return nil, err
	}
	m.Targets = toTargets(targets)

	prio, err := loadPriorityLabels(ctx, x)
	if err != nil {
		return nil, err
	}
	m.Priorities = prio

	return m, nil
}

func toSections(rows []store.SectionRow) []*Section {
	out := make([]*Section, len(rows))
	for i, r := range rows {
		out[i] = &Section{Key: r.Key, Title: r.Title, Level: r.Level, Position: r.Position, Body: r.Body}
	}
	return out
}

// toTargets adapts store ref-target rows to the refs resolver's Target shape.
func toTargets(rows []store.RefTargetRow) []refs.Target {
	out := make([]refs.Target, len(rows))
	for i, r := range rows {
		out[i] = refs.Target{Type: r.Type, Key: r.Key, ID: r.ID, DocPath: r.DocPath, Anchor: r.Anchor}
	}
	return out
}

// loadPriorityLabels loads the level→label map for the 0–4 priority taxonomy.
func loadPriorityLabels(ctx context.Context, x store.Execer) (map[int]string, error) {
	rows, err := store.ListPriorities(ctx, x)
	if err != nil {
		return nil, err
	}
	m := make(map[int]string, len(rows))
	for _, p := range rows {
		m[p.Level] = p.Label
	}
	return m, nil
}
