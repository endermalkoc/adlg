package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CRUD show/edit/delete for the remaining entities (domain/entity/glossary-term/edge/
// section). Deletes reuse DeleteNodeRefs for polymorphic references and the FK cascade for
// structured children; non-FK junctions (entity relationships, glossary aliases) are
// cleaned explicitly.

// ---- Domain ---------------------------------------------------------------

// GetDomain fetches one domain by slug. ok is false when none exists.
func GetDomain(ctx context.Context, x Execer, slug string) (Domain, bool, error) {
	var d Domain
	err := x.QueryRowContext(ctx,
		"SELECT id, slug, name, COALESCE(description,''), status FROM `req_domain` WHERE slug=?", slug).
		Scan(&d.ID, &d.Slug, &d.Name, &d.Description, &d.Status)
	if err == sql.ErrNoRows {
		return Domain{}, false, nil
	}
	if err != nil {
		return Domain{}, false, err
	}
	return d, true, nil
}

// UpdateDomain writes a domain's editable fields (name, description, status) by id.
func UpdateDomain(ctx context.Context, x Execer, id, name, description, status string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `req_domain` SET name=?, description=?, status=?, updated_at=? WHERE id=?",
		name, nullIfEmpty(description), status, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("update domain %s: %w", id, err)
	}
	return nil
}

// DeleteDomain removes a domain and every spec under it. The spec→domain FK is RESTRICT,
// so child specs are deleted explicitly via deleteSpecByID (which also cleans their refs and
// cascades their requirements/sections/stories); then the domain's own refs and the row.
func DeleteDomain(ctx context.Context, x Execer, slug string) (Domain, bool, error) {
	d, ok, err := GetDomain(ctx, x, slug)
	if err != nil || !ok {
		return Domain{}, ok, err
	}
	specIDs, err := childIDs(ctx, x, "SELECT id FROM `req_spec` WHERE domain_id=?", d.ID)
	if err != nil {
		return Domain{}, false, err
	}
	for _, id := range specIDs {
		if err := deleteSpecByID(ctx, x, id); err != nil {
			return Domain{}, false, err
		}
	}
	if err := DeleteNodeRefs(ctx, x, "domain", d.ID); err != nil {
		return Domain{}, false, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `req_domain` WHERE id=?", d.ID); err != nil {
		return Domain{}, false, fmt.Errorf("delete domain %s: %w", slug, err)
	}
	return d, true, nil
}

// ---- Entity ---------------------------------------------------------------

// GetEntity fetches one entity by name. ok is false when none exists.
func GetEntity(ctx context.Context, x Execer, name string) (EntityRow, bool, error) {
	var e EntityRow
	var subdir string
	err := x.QueryRowContext(ctx,
		"SELECT id, name, COALESCE(description,''), status, COALESCE(path,'') FROM `ent_entity` WHERE name=?", name).
		Scan(&e.ID, &e.Name, &e.Description, &e.Status, &subdir)
	if err == sql.ErrNoRows {
		return EntityRow{}, false, nil
	}
	if err != nil {
		return EntityRow{}, false, err
	}
	e.DocPath = EntityDocPath(subdir, e.Name)
	return e, true, nil
}

// UpdateEntity writes an entity's editable fields (description, status) by id. Name/path
// are identity and not touched here.
func UpdateEntity(ctx context.Context, x Execer, id, description, status string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `ent_entity` SET description=?, status=?, updated_at=? WHERE id=?",
		nullIfEmpty(description), status, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("update entity %s: %w", id, err)
	}
	return nil
}

// DeleteEntity removes an entity, its sections (FK CASCADE), its entity relationships (no
// FK — cleaned here), its polymorphic refs, and the row.
func DeleteEntity(ctx context.Context, x Execer, name string) (EntityRow, bool, error) {
	e, ok, err := GetEntity(ctx, x, name)
	if err != nil || !ok {
		return EntityRow{}, ok, err
	}
	if _, err := x.ExecContext(ctx,
		"DELETE FROM `ent_relationship` WHERE from_entity_id=? OR to_entity_id=?", e.ID, e.ID); err != nil {
		return EntityRow{}, false, fmt.Errorf("delete relationships for entity %s: %w", name, err)
	}
	if err := DeleteNodeRefs(ctx, x, "entity", e.ID); err != nil {
		return EntityRow{}, false, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `ent_entity` WHERE id=?", e.ID); err != nil {
		return EntityRow{}, false, fmt.Errorf("delete entity %s: %w", name, err)
	}
	return e, true, nil
}

// ---- Glossary term --------------------------------------------------------

// GetGlossaryTerm fetches one term by slug, with its aliases. ok is false when none exists.
func GetGlossaryTerm(ctx context.Context, x Execer, slug string) (GlossaryTermRow, bool, error) {
	var t GlossaryTermRow
	err := x.QueryRowContext(ctx,
		"SELECT t.id, t.slug, COALESCE(t.term,''), COALESCE(t.definition,''), COALESCE(d.slug,''), t.status "+
			"FROM `req_glossary_term` t LEFT JOIN `req_domain` d ON t.domain_id=d.id WHERE t.slug=?", slug).
		Scan(&t.ID, &t.Slug, &t.Term, &t.Definition, &t.DomainSlug, &t.Status)
	if err == sql.ErrNoRows {
		return GlossaryTermRow{}, false, nil
	}
	if err != nil {
		return GlossaryTermRow{}, false, err
	}
	aliases, err := childIDs(ctx, x, "SELECT alias FROM `req_glossary_alias` WHERE term_id=? ORDER BY alias", t.ID)
	if err != nil {
		return GlossaryTermRow{}, false, err
	}
	t.Aliases = aliases
	return t, true, nil
}

// UpdateGlossaryTerm writes a term's editable fields (term, definition, status) by id.
func UpdateGlossaryTerm(ctx context.Context, x Execer, id, term, definition, status string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `req_glossary_term` SET term=?, definition=?, status=?, updated_at=? WHERE id=?",
		term, nullIfEmpty(definition), status, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("update glossary term %s: %w", id, err)
	}
	return nil
}

// DeleteGlossaryTerm removes a term, its aliases (no FK — cleaned here), its polymorphic
// refs (as a glossary_term target), and the row.
func DeleteGlossaryTerm(ctx context.Context, x Execer, slug string) (GlossaryTermRow, bool, error) {
	t, ok, err := GetGlossaryTerm(ctx, x, slug)
	if err != nil || !ok {
		return GlossaryTermRow{}, ok, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `req_glossary_alias` WHERE term_id=?", t.ID); err != nil {
		return GlossaryTermRow{}, false, fmt.Errorf("delete aliases for term %s: %w", slug, err)
	}
	if err := DeleteNodeRefs(ctx, x, "glossary_term", t.ID); err != nil {
		return GlossaryTermRow{}, false, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `req_glossary_term` WHERE id=?", t.ID); err != nil {
		return GlossaryTermRow{}, false, fmt.Errorf("delete glossary term %s: %w", slug, err)
	}
	return t, true, nil
}

// ---- Edge -----------------------------------------------------------------

// DeleteEdgeByEndpoints removes the edge with the given endpoints + kind. ok reports
// whether a matching edge existed.
func DeleteEdgeByEndpoints(ctx context.Context, x Execer, fromType, fromID, kind, toType, toID string) (bool, error) {
	res, err := x.ExecContext(ctx,
		"DELETE FROM `req_edge` WHERE from_type=? AND from_id=? AND kind=? AND to_type=? AND to_id=?",
		fromType, fromID, kind, toType, toID)
	if err != nil {
		return false, fmt.Errorf("delete edge: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ---- Section --------------------------------------------------------------

// DeleteSpecSection removes one spec section by its section-type slug. ok reports whether
// the section existed.
func DeleteSpecSection(ctx context.Context, x Execer, specID, sectionTypeSlug string) (bool, error) {
	return deleteOneSection(ctx, x, "req_spec_section", "spec_id", specID, sectionTypeSlug)
}

// DeleteEntitySection removes one entity section by its section-type slug.
func DeleteEntitySection(ctx context.Context, x Execer, entityID, sectionTypeSlug string) (bool, error) {
	return deleteOneSection(ctx, x, "ent_entity_section", "entity_id", entityID, sectionTypeSlug)
}

func deleteOneSection(ctx context.Context, x Execer, table, ownerCol, ownerID, slug string) (bool, error) {
	res, err := x.ExecContext(ctx,
		"DELETE FROM `"+table+"` WHERE "+ownerCol+"=? AND section_type_slug=?", ownerID, slug)
	if err != nil {
		return false, fmt.Errorf("delete section %s: %w", slug, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
