package store

import "context"

// Read queries for the planning layer (plan_capability / plan_deliverable / plan_view
// + their junctions), used by the generator to render the planning roll-up pages. Plain
// reads, no transaction — mirroring read.go.

// CapabilityRow is a capability joined to its domain slug.
type CapabilityRow struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Level      string `json:"level,omitempty"`
	DomainSlug string `json:"domain,omitempty"`
	ParentID   string `json:"-"`
}

// ListCapabilities returns every capability with its domain slug, by title.
func ListCapabilities(ctx context.Context, x Execer) ([]CapabilityRow, error) {
	rows, err := x.QueryContext(ctx, `
		SELECT c.id, COALESCE(c.title,''), COALESCE(c.level,''), COALESCE(d.slug,''), COALESCE(c.parent_id,'')
		FROM `+"`plan_capability`"+` c LEFT JOIN `+"`req_domain`"+` d ON c.domain_id = d.id
		ORDER BY c.title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CapabilityRow
	for rows.Next() {
		var c CapabilityRow
		if err := rows.Scan(&c.ID, &c.Title, &c.Level, &c.DomainSlug, &c.ParentID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeliverableRow is a deliverable joined to its milestone slug.
type DeliverableRow struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Size          string `json:"size,omitempty"`
	Status        string `json:"status,omitempty"`
	AIReady       string `json:"ai_ready,omitempty"`
	MilestoneSlug string `json:"milestone,omitempty"`
}

// ListDeliverables returns every deliverable with its milestone slug, by title.
func ListDeliverables(ctx context.Context, x Execer) ([]DeliverableRow, error) {
	rows, err := x.QueryContext(ctx, `
		SELECT dl.id, COALESCE(dl.title,''), COALESCE(dl.size,''), COALESCE(dl.status,''), COALESCE(dl.ai_ready,''), COALESCE(m.slug,'')
		FROM `+"`plan_deliverable`"+` dl LEFT JOIN `+"`plan_milestone`"+` m ON dl.milestone_id = m.id
		ORDER BY dl.title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeliverableRow
	for rows.Next() {
		var d DeliverableRow
		if err := rows.Scan(&d.ID, &d.Title, &d.Size, &d.Status, &d.AIReady, &d.MilestoneSlug); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// PlanViewRow is a view joined to its domain slug and (optionally) its backing spec's
// location, so the generator can link the view to that spec's document.
type PlanViewRow struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Route      string `json:"route,omitempty"`
	DomainSlug string `json:"domain,omitempty"`
	// Backing spec (empty when spec_id is null). SpecDomain/SpecPath/SpecSlug feed SpecDocPath.
	SpecDomain string `json:"-"`
	SpecPath   string `json:"-"`
	SpecSlug   string `json:"-"`
	SpecTitle  string `json:"spec_title,omitempty"`
}

// ListPlanViews returns every view with its domain slug and backing-spec location, by title.
func ListPlanViews(ctx context.Context, x Execer) ([]PlanViewRow, error) {
	rows, err := x.QueryContext(ctx, `
		SELECT v.id, COALESCE(v.title,''), COALESCE(v.route,''), COALESCE(vd.slug,''),
		       COALESCE(sd.slug,''), COALESCE(s.path,''), COALESCE(s.slug,''), COALESCE(s.title,'')
		FROM `+"`plan_view`"+` v
		LEFT JOIN `+"`req_domain`"+` vd ON v.domain_id = vd.id
		LEFT JOIN `+"`req_spec`"+` s ON v.spec_id = s.id
		LEFT JOIN `+"`req_domain`"+` sd ON s.domain_id = sd.id
		ORDER BY v.title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlanViewRow
	for rows.Next() {
		var v PlanViewRow
		if err := rows.Scan(&v.ID, &v.Title, &v.Route, &v.DomainSlug, &v.SpecDomain, &v.SpecPath, &v.SpecSlug, &v.SpecTitle); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// IDPair is one (owner, other) row from a planning junction table.
type IDPair struct{ A, B string }

func listPairs(ctx context.Context, x Execer, query string) ([]IDPair, error) {
	rows, err := x.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IDPair
	for rows.Next() {
		var p IDPair
		if err := rows.Scan(&p.A, &p.B); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListCapabilityMilestonePairs returns (capability_id, milestone_slug) links.
func ListCapabilityMilestonePairs(ctx context.Context, x Execer) ([]IDPair, error) {
	return listPairs(ctx, x, "SELECT cm.capability_id, m.slug FROM `plan_capability_milestone` cm JOIN `plan_milestone` m ON cm.milestone_id = m.id")
}

// ListCapabilityDeliverablePairs returns (capability_id, deliverable_id) links.
func ListCapabilityDeliverablePairs(ctx context.Context, x Execer) ([]IDPair, error) {
	return listPairs(ctx, x, "SELECT capability_id, deliverable_id FROM `plan_capability_deliverable`")
}

// ListDeliverableViewPairs returns (deliverable_id, view_id) links.
func ListDeliverableViewPairs(ctx context.Context, x Execer) ([]IDPair, error) {
	return listPairs(ctx, x, "SELECT deliverable_id, view_id FROM `plan_deliverable_view`")
}

// ListDeliverableDependencyPairs returns (deliverable_id, blocked_by_id) links.
func ListDeliverableDependencyPairs(ctx context.Context, x Execer) ([]IDPair, error) {
	return listPairs(ctx, x, "SELECT deliverable_id, blocked_by_id FROM `plan_deliverable_dependency`")
}

// ExternalRefRow is one external_ref row (the source/tracker pointer on a subject).
type ExternalRefView struct {
	SubjectType string
	SubjectID   string
	System      string
	ExternalID  string
	URL         string
}

// ListExternalRefsForSubjects returns external refs whose subject_type is one of the
// planning subjects (capability, deliverable, view), so the generator can surface the
// Notion source link and any Bead IDs.
func ListExternalRefsForSubjects(ctx context.Context, x Execer) ([]ExternalRefView, error) {
	rows, err := x.QueryContext(ctx, `
		SELECT subject_type, subject_id, `+"`system`"+`, external_id, COALESCE(url,'')
		FROM `+"`pub_external_ref`"+`
		WHERE subject_type IN ('capability','deliverable','view')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExternalRefView
	for rows.Next() {
		var r ExternalRefView
		if err := rows.Scan(&r.SubjectType, &r.SubjectID, &r.System, &r.ExternalID, &r.URL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// PlanningCounts reports how many capabilities / deliverables / views exist (cheap
// existence check for the nav, which decides whether to show the Planning section).
func PlanningCounts(ctx context.Context, x Execer) (caps, delivs, views int, err error) {
	q := func(table string) (int, error) {
		var n int
		e := x.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+table+"`").Scan(&n)
		return n, e
	}
	if caps, err = q("plan_capability"); err != nil {
		return
	}
	if delivs, err = q("plan_deliverable"); err != nil {
		return
	}
	views, err = q("plan_view")
	return
}
