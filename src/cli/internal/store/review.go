package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/endermalkoc/cusp/internal/ids"
)

// This file holds the review layer's store functions: the changeset lookup a
// comment/review verb needs to resolve a branch to its row, and CRUD for the
// rev_comment / rev_review tables. These rows live on `main` (docs/entities/
// review.md) — the caller pins that branch; here we only read/write via Execer.
//
// Timestamps read back as RFC3339-ish strings via DATE_FORMAT; the columns are
// written as time.Now().UTC(), so the literal 'Z' is accurate.
const tsFormat = "'%Y-%m-%dT%H:%i:%sZ'"

// ---- Changeset lookup -----------------------------------------------------

// ChangesetRow is a rev_changeset row (the review workflow's unit). It always
// lives on main; the branch string is its stable public id.
type ChangesetRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Branch      string `json:"branch"`
	BaseCommit  string `json:"baseCommit,omitempty"`
	HeadCommit  string `json:"headCommit,omitempty"`
	MergeCommit string `json:"mergeCommit,omitempty"`
}

// GetChangesetByBranch fetches the changeset with the given Dolt branch. ok is
// false when none exists.
func GetChangesetByBranch(ctx context.Context, x Execer, branch string) (ChangesetRow, bool, error) {
	var c ChangesetRow
	err := x.QueryRowContext(ctx,
		"SELECT id, COALESCE(title,''), COALESCE(description,''), status, branch, "+
			"COALESCE(base_commit,''), COALESCE(head_commit,''), COALESCE(merge_commit,'') "+
			"FROM `rev_changeset` WHERE branch=?", branch).
		Scan(&c.ID, &c.Title, &c.Description, &c.Status, &c.Branch, &c.BaseCommit, &c.HeadCommit, &c.MergeCommit)
	if err == sql.ErrNoRows {
		return ChangesetRow{}, false, nil
	}
	if err != nil {
		return ChangesetRow{}, false, err
	}
	return c, true, nil
}

// SetChangesetStatus updates a changeset's status column (e.g. from a review verdict).
func SetChangesetStatus(ctx context.Context, x Execer, changesetID, status string) error {
	if _, err := x.ExecContext(ctx,
		"UPDATE `rev_changeset` SET status=?, updated_at=? WHERE id=?",
		status, time.Now().UTC(), changesetID); err != nil {
		return fmt.Errorf("set changeset %s status: %w", changesetID, err)
	}
	return nil
}

// ---- Comment --------------------------------------------------------------

// CommentRow is a rev_comment row. SubjectRef is not stored — the CLI fills it
// from a LabelIndex (subject_id → "type:key") for display.
type CommentRow struct {
	ID           string `json:"id"`
	ChangesetID  string `json:"changesetId"`
	AuthorID     string `json:"authorId"`
	AuthorHandle string `json:"authorHandle"`
	ParentID     string `json:"parentId,omitempty"`
	Body         string `json:"body"`
	SubjectType  string `json:"subjectType,omitempty"`
	SubjectID    string `json:"subjectId,omitempty"`
	SubjectRef   string `json:"subjectRef,omitempty"`
	Locator      string `json:"locator,omitempty"`
	Resolved     bool   `json:"resolved"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

const commentCols = "c.id, c.changeset_id, c.author_id, COALESCE(a.handle,''), COALESCE(c.parent_id,''), " +
	"COALESCE(c.body,''), COALESCE(c.subject_type,''), COALESCE(c.subject_id,''), COALESCE(c.locator,''), " +
	"c.resolved, COALESCE(DATE_FORMAT(c.created_at," + tsFormat + "),''), " +
	"COALESCE(DATE_FORMAT(c.updated_at," + tsFormat + "),'')"

func scanComment(s interface{ Scan(...any) error }) (CommentRow, error) {
	var c CommentRow
	err := s.Scan(&c.ID, &c.ChangesetID, &c.AuthorID, &c.AuthorHandle, &c.ParentID, &c.Body,
		&c.SubjectType, &c.SubjectID, &c.Locator, &c.Resolved, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// AddComment inserts a comment. parentID/subjectType/subjectID/locator are optional
// ("" → NULL). The caller mints nothing — the id is a fresh ULID.
func AddComment(ctx context.Context, x Execer, changesetID, authorID, parentID, body, subjectType, subjectID, locator string) (CommentRow, error) {
	id := ids.New()
	now := time.Now().UTC()
	if _, err := x.ExecContext(ctx,
		"INSERT INTO `rev_comment` (id,changeset_id,author_id,parent_id,body,subject_type,subject_id,locator,resolved,created_at,updated_at) "+
			"VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		id, changesetID, authorID, nullIfEmpty(parentID), body,
		nullIfEmpty(subjectType), nullIfEmpty(subjectID), nullIfEmpty(locator), false, now, now); err != nil {
		return CommentRow{}, fmt.Errorf("add comment: %w", err)
	}
	return GetCommentReturning(ctx, x, id)
}

// GetComment fetches one comment by id (with the author handle). ok is false when none exists.
func GetComment(ctx context.Context, x Execer, id string) (CommentRow, bool, error) {
	row := x.QueryRowContext(ctx,
		"SELECT "+commentCols+" FROM `rev_comment` c LEFT JOIN `rev_actor` a ON c.author_id=a.id WHERE c.id=?", id)
	c, err := scanComment(row)
	if err == sql.ErrNoRows {
		return CommentRow{}, false, nil
	}
	if err != nil {
		return CommentRow{}, false, err
	}
	return c, true, nil
}

// GetCommentReturning is GetComment for a row we just wrote (it must exist); it drops the ok bool.
func GetCommentReturning(ctx context.Context, x Execer, id string) (CommentRow, error) {
	c, ok, err := GetComment(ctx, x, id)
	if err != nil {
		return CommentRow{}, err
	}
	if !ok {
		return CommentRow{}, fmt.Errorf("comment %s not found after write", id)
	}
	return c, nil
}

// CommentFilter narrows a comment listing.
type CommentFilter struct {
	SubjectType    string // "" → any
	SubjectID      string // "" → any
	UnresolvedOnly bool
}

// ListComments returns a changeset's comments ordered oldest-first (stable by id), with the
// author handle. The CLI threads them by ParentID and resolves SubjectRef.
func ListComments(ctx context.Context, x Execer, changesetID string, f CommentFilter) ([]CommentRow, error) {
	q := "SELECT " + commentCols + " FROM `rev_comment` c LEFT JOIN `rev_actor` a ON c.author_id=a.id WHERE c.changeset_id=?"
	args := []any{changesetID}
	if f.SubjectType != "" {
		q += " AND c.subject_type=?"
		args = append(args, f.SubjectType)
	}
	if f.SubjectID != "" {
		q += " AND c.subject_id=?"
		args = append(args, f.SubjectID)
	}
	if f.UnresolvedOnly {
		q += " AND c.resolved=FALSE"
	}
	q += " ORDER BY c.created_at, c.id"
	rows, err := x.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentRow
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetCommentResolved toggles a comment's resolved flag. ok is false when no such comment exists.
func SetCommentResolved(ctx context.Context, x Execer, id string, resolved bool) (bool, error) {
	res, err := x.ExecContext(ctx,
		"UPDATE `rev_comment` SET resolved=?, updated_at=? WHERE id=?", resolved, time.Now().UTC(), id)
	if err != nil {
		return false, fmt.Errorf("set comment %s resolved: %w", id, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UpdateCommentBody rewrites a comment's body. ok is false when no such comment exists.
func UpdateCommentBody(ctx context.Context, x Execer, id, body string) (bool, error) {
	res, err := x.ExecContext(ctx,
		"UPDATE `rev_comment` SET body=?, updated_at=? WHERE id=?", body, time.Now().UTC(), id)
	if err != nil {
		return false, fmt.Errorf("update comment %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeleteComment removes a comment. Replies (children) survive with parent_id nulled (FK ON
// DELETE SET NULL). ok is false when no such comment existed.
func DeleteComment(ctx context.Context, x Execer, id string) (CommentRow, bool, error) {
	c, ok, err := GetComment(ctx, x, id)
	if err != nil || !ok {
		return CommentRow{}, ok, err
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM `rev_comment` WHERE id=?", id); err != nil {
		return CommentRow{}, false, fmt.Errorf("delete comment %s: %w", id, err)
	}
	return c, true, nil
}

// ---- Review ---------------------------------------------------------------

// ReviewRow is a rev_review row — one reviewer's current verdict on a changeset.
type ReviewRow struct {
	ID             string `json:"id"`
	ChangesetID    string `json:"changesetId"`
	ReviewerID     string `json:"reviewerId"`
	ReviewerHandle string `json:"reviewerHandle"`
	Verdict        string `json:"verdict,omitempty"`
	Summary        string `json:"summary,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

const reviewCols = "r.id, r.changeset_id, r.reviewer_id, COALESCE(a.handle,''), COALESCE(r.verdict,''), " +
	"COALESCE(r.summary,''), COALESCE(DATE_FORMAT(r.created_at," + tsFormat + "),''), " +
	"COALESCE(DATE_FORMAT(r.updated_at," + tsFormat + "),'')"

func scanReview(s interface{ Scan(...any) error }) (ReviewRow, error) {
	var r ReviewRow
	err := s.Scan(&r.ID, &r.ChangesetID, &r.ReviewerID, &r.ReviewerHandle, &r.Verdict, &r.Summary,
		&r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// UpsertReview records reviewerID's verdict on changesetID, replacing any prior verdict by the
// same reviewer. The id is deterministic in (changeset, reviewer) so re-review updates the same
// row and it converges on merge — matching UNIQUE(changeset_id, reviewer_id).
func UpsertReview(ctx context.Context, x Execer, changesetID, reviewerID, verdict, summary string) (ReviewRow, error) {
	id := ids.Rel("review", changesetID, reviewerID)
	now := time.Now().UTC()
	if _, err := x.ExecContext(ctx,
		"INSERT INTO `rev_review` (id,changeset_id,reviewer_id,verdict,summary,created_at,updated_at) VALUES (?,?,?,?,?,?,?) "+
			"ON DUPLICATE KEY UPDATE verdict=?, summary=?, updated_at=?",
		id, changesetID, reviewerID, nullIfEmpty(verdict), nullIfEmpty(summary), now, now,
		nullIfEmpty(verdict), nullIfEmpty(summary), now); err != nil {
		return ReviewRow{}, fmt.Errorf("record review: %w", err)
	}
	row := x.QueryRowContext(ctx,
		"SELECT "+reviewCols+" FROM `rev_review` r LEFT JOIN `rev_actor` a ON r.reviewer_id=a.id WHERE r.id=?", id)
	return scanReview(row)
}

// ListReviews returns a changeset's reviews, newest-first.
func ListReviews(ctx context.Context, x Execer, changesetID string) ([]ReviewRow, error) {
	rows, err := x.QueryContext(ctx,
		"SELECT "+reviewCols+" FROM `rev_review` r LEFT JOIN `rev_actor` a ON r.reviewer_id=a.id "+
			"WHERE r.changeset_id=? ORDER BY r.updated_at DESC, r.id", changesetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReviewRow
	for rows.Next() {
		r, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
