package store

import (
	"context"
	"database/sql"
	"fmt"
)

// This file holds idempotent, id-keyed upserts for the planning layer
// (plan_capability / plan_deliverable / plan_view + their junctions). Unlike the
// business-key upserts in import.go, these take an explicit primary key: the caller
// derives a deterministic id from the external source id (e.g. a Notion page id) so
// re-import converges instead of duplicating. Each upsert returns whether it inserted
// (true) or updated.

// Capability is the importable subset of a plan_capability row. ID is caller-supplied
// (deterministic from the source id). ParentID is set in a second pass (SetCapabilityParent)
// so a parent need not be inserted first.
type Capability struct {
	ID       string
	Title    string
	Level    string
	DomainID string
}

// UpsertCapability upserts a capability by its (caller-supplied) id.
func UpsertCapability(ctx context.Context, x Execer, c Capability) (bool, error) {
	var existing string
	err := x.QueryRowContext(ctx, "SELECT id FROM `plan_capability` WHERE id=?", c.ID).Scan(&existing)
	switch {
	case err == sql.ErrNoRows:
		_, err = x.ExecContext(ctx,
			"INSERT INTO `plan_capability` (id,title,level,domain_id) VALUES (?,?,?,?)",
			c.ID, nullIfEmpty(c.Title), nullIfEmpty(c.Level), c.DomainID)
		if err != nil {
			return false, fmt.Errorf("insert capability %q: %w", c.Title, err)
		}
		return true, nil
	case err != nil:
		return false, err
	}
	_, err = x.ExecContext(ctx,
		"UPDATE `plan_capability` SET title=?, level=?, domain_id=? WHERE id=?",
		nullIfEmpty(c.Title), nullIfEmpty(c.Level), c.DomainID, c.ID)
	if err != nil {
		return false, fmt.Errorf("update capability %q: %w", c.Title, err)
	}
	return false, nil
}

// SetCapabilityParent sets (or clears, when parentID is "") a capability's parent_id.
// Run after all capabilities exist, so the self-referencing FK always resolves.
func SetCapabilityParent(ctx context.Context, x Execer, id, parentID string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `plan_capability` SET parent_id=? WHERE id=?", nullIfEmpty(parentID), id); err != nil {
		return fmt.Errorf("set capability %s parent: %w", id, err)
	}
	return nil
}

// Deliverable is the importable subset of a plan_deliverable row.
type Deliverable struct {
	ID          string
	Title       string
	Size        string
	Status      string // proposed|specced|wired|built|ship (NOT NULL; defaults to proposed)
	AIReady     string
	MilestoneID string
}

// UpsertDeliverable upserts a deliverable by its (caller-supplied) id.
func UpsertDeliverable(ctx context.Context, x Execer, d Deliverable) (bool, error) {
	if d.Status == "" {
		d.Status = "proposed"
	}
	var existing string
	err := x.QueryRowContext(ctx, "SELECT id FROM `plan_deliverable` WHERE id=?", d.ID).Scan(&existing)
	switch {
	case err == sql.ErrNoRows:
		_, err = x.ExecContext(ctx,
			"INSERT INTO `plan_deliverable` (id,title,size,status,ai_ready,milestone_id) VALUES (?,?,?,?,?,?)",
			d.ID, nullIfEmpty(d.Title), nullIfEmpty(d.Size), d.Status, nullIfEmpty(d.AIReady), nullIfEmpty(d.MilestoneID))
		if err != nil {
			return false, fmt.Errorf("insert deliverable %q: %w", d.Title, err)
		}
		return true, nil
	case err != nil:
		return false, err
	}
	_, err = x.ExecContext(ctx,
		"UPDATE `plan_deliverable` SET title=?, size=?, status=?, ai_ready=?, milestone_id=? WHERE id=?",
		nullIfEmpty(d.Title), nullIfEmpty(d.Size), d.Status, nullIfEmpty(d.AIReady), nullIfEmpty(d.MilestoneID), d.ID)
	if err != nil {
		return false, fmt.Errorf("update deliverable %q: %w", d.Title, err)
	}
	return false, nil
}

// View is the importable subset of a plan_view row.
type View struct {
	ID       string
	Title    string
	Route    string
	SpecID   string
	DomainID string
}

// UpsertView upserts a view by its (caller-supplied) id.
func UpsertView(ctx context.Context, x Execer, v View) (bool, error) {
	var existing string
	err := x.QueryRowContext(ctx, "SELECT id FROM `plan_view` WHERE id=?", v.ID).Scan(&existing)
	switch {
	case err == sql.ErrNoRows:
		_, err = x.ExecContext(ctx,
			"INSERT INTO `plan_view` (id,title,route,spec_id,domain_id) VALUES (?,?,?,?,?)",
			v.ID, nullIfEmpty(v.Title), nullIfEmpty(v.Route), nullIfEmpty(v.SpecID), v.DomainID)
		if err != nil {
			return false, fmt.Errorf("insert view %q: %w", v.Title, err)
		}
		return true, nil
	case err != nil:
		return false, err
	}
	_, err = x.ExecContext(ctx,
		"UPDATE `plan_view` SET title=?, route=?, spec_id=?, domain_id=? WHERE id=?",
		nullIfEmpty(v.Title), nullIfEmpty(v.Route), nullIfEmpty(v.SpecID), v.DomainID, v.ID)
	if err != nil {
		return false, fmt.Errorf("update view %q: %w", v.Title, err)
	}
	return false, nil
}

// FindSpecIDBySlug returns the id of the unique spec with this slug. found is false
// when zero — or more than one — spec matches (ambiguous), so a caller can leave a
// soft FK (view.spec_id) null rather than guess.
func FindSpecIDBySlug(ctx context.Context, x Execer, slug string) (string, bool, error) {
	rows, err := x.QueryContext(ctx, "SELECT id FROM `req_spec` WHERE slug=? LIMIT 2", slug)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", false, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", false, err
	}
	if len(ids) == 1 {
		return ids[0], true, nil
	}
	return "", false, nil
}

// ---- planning junctions (composite-PK, merge-safe via INSERT IGNORE) --------

func linkJunction(ctx context.Context, x Execer, table, colA, colB, a, b string) error {
	if _, err := x.ExecContext(ctx,
		"INSERT IGNORE INTO `"+table+"` (`"+colA+"`,`"+colB+"`) VALUES (?,?)", a, b); err != nil {
		return fmt.Errorf("link %s (%s,%s): %w", table, a, b, err)
	}
	return nil
}

func clearJunctionByOwner(ctx context.Context, x Execer, table, ownerCol, ownerID string) error {
	if _, err := x.ExecContext(ctx,
		"DELETE FROM `"+table+"` WHERE `"+ownerCol+"`=?", ownerID); err != nil {
		return fmt.Errorf("clear %s for %s: %w", table, ownerID, err)
	}
	return nil
}

// LinkCapabilityMilestone / ClearCapabilityMilestones — plan_capability_milestone.
func LinkCapabilityMilestone(ctx context.Context, x Execer, capID, milestoneID string) error {
	return linkJunction(ctx, x, "plan_capability_milestone", "capability_id", "milestone_id", capID, milestoneID)
}
func ClearCapabilityMilestones(ctx context.Context, x Execer, capID string) error {
	return clearJunctionByOwner(ctx, x, "plan_capability_milestone", "capability_id", capID)
}

// LinkCapabilityDeliverable / ClearCapabilityDeliverables — plan_capability_deliverable.
func LinkCapabilityDeliverable(ctx context.Context, x Execer, capID, deliverableID string) error {
	return linkJunction(ctx, x, "plan_capability_deliverable", "capability_id", "deliverable_id", capID, deliverableID)
}
func ClearCapabilityDeliverables(ctx context.Context, x Execer, capID string) error {
	return clearJunctionByOwner(ctx, x, "plan_capability_deliverable", "capability_id", capID)
}

// LinkDeliverableView / ClearDeliverableViews — plan_deliverable_view.
func LinkDeliverableView(ctx context.Context, x Execer, deliverableID, viewID string) error {
	return linkJunction(ctx, x, "plan_deliverable_view", "deliverable_id", "view_id", deliverableID, viewID)
}
func ClearDeliverableViews(ctx context.Context, x Execer, deliverableID string) error {
	return clearJunctionByOwner(ctx, x, "plan_deliverable_view", "deliverable_id", deliverableID)
}

// LinkDeliverableDependency / ClearDeliverableDependencies — plan_deliverable_dependency
// (deliverable_id is blocked_by blocked_by_id; both → plan_deliverable).
func LinkDeliverableDependency(ctx context.Context, x Execer, deliverableID, blockedByID string) error {
	return linkJunction(ctx, x, "plan_deliverable_dependency", "deliverable_id", "blocked_by_id", deliverableID, blockedByID)
}
func ClearDeliverableDependencies(ctx context.Context, x Execer, deliverableID string) error {
	return clearJunctionByOwner(ctx, x, "plan_deliverable_dependency", "deliverable_id", deliverableID)
}
