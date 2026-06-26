package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// This file holds the show/edit/delete primitives (the Add/List ones live in store.go).
// Deletes must also remove POLYMORPHIC references — entity_refs, edges, external_refs are
// keyed by (type,id) strings, not FKs, so the database can't cascade them; DeleteNodeRefs
// does, and is reused by every entity's delete.

// DeleteNodeRefs removes every polymorphic reference TO or FROM a node — its entity_refs,
// edges, and external_refs — so deleting the node leaves no dangling reference behind.
// FK-constrained children (e.g. a spec's requirements/sections) are handled by ON DELETE
// CASCADE or an explicit sweep by the caller, not here.
func DeleteNodeRefs(ctx context.Context, x Execer, nodeType, nodeID string) error {
	for _, s := range []struct {
		q    string
		args []any
	}{
		{"DELETE FROM `req_entity_ref` WHERE (owner_type=? AND owner_id=?) OR (target_type=? AND target_id=?)",
			[]any{nodeType, nodeID, nodeType, nodeID}},
		{"DELETE FROM `req_edge` WHERE (from_type=? AND from_id=?) OR (to_type=? AND to_id=?)",
			[]any{nodeType, nodeID, nodeType, nodeID}},
		{"DELETE FROM `pub_external_ref` WHERE subject_type=? AND subject_id=?",
			[]any{nodeType, nodeID}},
	} {
		if _, err := x.ExecContext(ctx, s.q, s.args...); err != nil {
			return fmt.Errorf("clean references for %s %s: %w", nodeType, nodeID, err)
		}
	}
	return nil
}

// ---- Requirement show/edit/delete -----------------------------------------

// GetRequirement fetches one requirement by its fr_key. ok is false when none exists on
// the current branch.
func GetRequirement(ctx context.Context, x Execer, frKey string) (Requirement, bool, error) {
	var r Requirement
	var priority sql.NullInt64
	err := x.QueryRowContext(ctx,
		"SELECT id,spec_id,number,COALESCE(suffix,''),fr_key,COALESCE(statement,''),content_status,"+
			"COALESCE(delivery_status,''),COALESCE(milestone_id,''),COALESCE(notes,''),priority "+
			"FROM `req_requirement` WHERE fr_key=?", frKey).
		Scan(&r.ID, &r.SpecID, &r.Number, &r.Suffix, &r.FRKey, &r.Statement, &r.ContentStatus,
			&r.DeliveryStatus, &r.MilestoneID, &r.Notes, &priority)
	if err == sql.ErrNoRows {
		return Requirement{}, false, nil
	}
	if err != nil {
		return Requirement{}, false, err
	}
	if priority.Valid {
		p := int(priority.Int64)
		r.Priority = &p
	}
	return r, true, nil
}

// UpdateRequirement writes the editable fields of an existing requirement (by its id) and
// bumps updated_at. Identity columns (spec_id/number/suffix/fr_key) are NOT touched —
// renumbering is a separate operation.
func UpdateRequirement(ctx context.Context, x Execer, r Requirement) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `req_requirement` SET statement=?,content_status=?,delivery_status=?,milestone_id=?,notes=?,priority=?,updated_at=? WHERE id=?",
		nullIfEmpty(r.Statement), r.ContentStatus, nullIfEmpty(r.DeliveryStatus), nullIfEmpty(r.MilestoneID),
		nullIfEmpty(r.Notes), nullIfNil(r.Priority), time.Now().UTC(), r.ID); err != nil {
		return fmt.Errorf("update requirement %s: %w", r.FRKey, err)
	}
	return nil
}

// DeleteRequirement removes a requirement and every polymorphic reference touching it.
// Sub-requirements keep existing (their parent_id is nulled by the FK). Returns the
// deleted row for output; ok is false when none exists.
func DeleteRequirement(ctx context.Context, x Execer, frKey string) (Requirement, bool, error) {
	r, ok, err := GetRequirement(ctx, x, frKey)
	if err != nil || !ok {
		return Requirement{}, ok, err
	}
	if err := DeleteNodeRefs(ctx, x, "requirement", r.ID); err != nil {
		return Requirement{}, false, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `req_requirement` WHERE id=?", r.ID); err != nil {
		return Requirement{}, false, fmt.Errorf("delete requirement %s: %w", frKey, err)
	}
	return r, true, nil
}

// ---- Spec show/edit/delete ------------------------------------------------

// specFullPath reconstructs a spec's full docs path in SQL: <domain>/[path/]<slug>.md.
const specFullPath = "CONCAT(d.slug,'/',IF(COALESCE(s.path,'')='','',CONCAT(s.path,'/')),COALESCE(s.slug,''),'.md')"

// GetSpec fetches one spec by its prefix or its full docs path (with or without the
// trailing .md). ok is false when none matches on the current branch.
func GetSpec(ctx context.Context, x Execer, key string) (SpecRow, bool, error) {
	var s SpecRow
	err := x.QueryRowContext(ctx,
		"SELECT s.id, d.slug, COALESCE(s.prefix,''), COALESCE(s.path,''), COALESCE(s.slug,''), COALESCE(s.title,''), s.status "+
			"FROM `req_spec` s JOIN `req_domain` d ON s.domain_id=d.id "+
			"WHERE s.prefix=? OR "+specFullPath+"=? OR CONCAT("+specFullPath+",'')=CONCAT(?, '.md')",
		key, key, key).
		Scan(&s.ID, &s.DomainSlug, &s.Prefix, &s.Path, &s.Slug, &s.Title, &s.Status)
	if err == sql.ErrNoRows {
		return SpecRow{}, false, nil
	}
	if err != nil {
		return SpecRow{}, false, err
	}
	return s, true, nil
}

// UpdateSpec writes a spec's editable fields (title, status) by id, bumping updated_at.
// Identity (prefix/slug/path/domain) is not touched here.
func UpdateSpec(ctx context.Context, x Execer, id, title, status string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `req_spec` SET title=?, status=?, updated_at=? WHERE id=?",
		nullIfEmpty(title), status, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("update spec %s: %w", id, err)
	}
	return nil
}

// DeleteSpec removes a spec and everything under it. The database cascades its
// requirements, sections, requirement-groups, user-stories and scenarios (ON DELETE
// CASCADE); this first cleans the POLYMORPHIC references (entity_refs/edges/external_refs)
// for the spec and each cascaded requirement/user-story, which FKs can't reach. Returns
// the deleted spec; ok is false when none matches.
func DeleteSpec(ctx context.Context, x Execer, key string) (SpecRow, bool, error) {
	sp, ok, err := GetSpec(ctx, x, key)
	if err != nil || !ok {
		return SpecRow{}, ok, err
	}
	reqIDs, err := childIDs(ctx, x, "SELECT id FROM `req_requirement` WHERE spec_id=?", sp.ID)
	if err != nil {
		return SpecRow{}, false, err
	}
	storyIDs, err := childIDs(ctx, x, "SELECT id FROM `req_user_story` WHERE spec_id=?", sp.ID)
	if err != nil {
		return SpecRow{}, false, err
	}
	if err := DeleteNodeRefs(ctx, x, "spec", sp.ID); err != nil {
		return SpecRow{}, false, err
	}
	for _, id := range reqIDs {
		if err := DeleteNodeRefs(ctx, x, "requirement", id); err != nil {
			return SpecRow{}, false, err
		}
	}
	for _, id := range storyIDs {
		if err := DeleteNodeRefs(ctx, x, "user_story", id); err != nil {
			return SpecRow{}, false, err
		}
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `req_spec` WHERE id=?", sp.ID); err != nil {
		return SpecRow{}, false, fmt.Errorf("delete spec %s: %w", key, err)
	}
	return sp, true, nil
}

// childIDs runs a single-column id query and collects the ids.
func childIDs(ctx context.Context, x Execer, query, arg string) ([]string, error) {
	rows, err := x.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
