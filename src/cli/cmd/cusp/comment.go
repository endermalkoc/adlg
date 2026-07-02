package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/endermalkoc/cusp/internal/app"
	"github.com/endermalkoc/cusp/internal/enums"
	"github.com/endermalkoc/cusp/internal/store"
	"github.com/endermalkoc/cusp/internal/workspace"
)

var (
	commentBody        string
	commentSubject     string // TYPE:key (or bare fr_key) — resolved on the changeset branch
	commentSubjectType string // explicit escape hatch for non-resolvable subjects
	commentSubjectID   string
	commentLocator     string
	commentReply       string // parent comment id (threading)
	commentUnresolved  bool
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Review comments on a changeset (threaded, optionally anchored to an entity)",
	Long: "Comment on a changeset under review — optionally anchored to a specific requirement,\n" +
		"spec, or entity (and a field/hunk locator), and threaded via replies. Comments live on\n" +
		"`main` (not the changeset branch), so they survive the changeset's merge.",
}

// resolveCommentSubject turns the --subject / --subject-type+--subject-id flags into a stored
// (subject_type, subject_id) pair plus a display ref. A --subject is resolved against the
// CHANGESET branch (the entity may have been added in the changeset, so it isn't on main yet).
// Returns all-empty for a changeset-level comment (no subject flags).
func resolveCommentSubject(ctx context.Context, ws *workspace.Workspace, branch string) (subjectType, subjectID, subjectRef string, err error) {
	switch {
	case commentSubject != "":
		r, release, e := app.Reader(ctx, ws, branch)
		if e != nil {
			return "", "", "", e
		}
		resolver, e := app.LoadResolver(ctx, r)
		_ = release()
		if e != nil {
			return "", "", "", e
		}
		target, ok := app.ResolveRef(resolver, commentSubject)
		if !ok {
			return "", "", "", app.NotFoundErr(fmt.Errorf("no entity %q in changeset %s", commentSubject, branch))
		}
		if !enums.Valid(enums.CommentSubjectType, target.Type) {
			return "", "", "", app.ValidationFailed(fmt.Errorf(
				"cannot anchor a comment to a %s (%s:%s); comment subjects are %v (or use --subject-type/--subject-id)",
				target.Type, target.Type, target.Key, enums.CommentSubjectType))
		}
		return target.Type, target.ID, target.Type + ":" + target.Key, nil
	case commentSubjectType != "" || commentSubjectID != "":
		if commentSubjectType == "" || commentSubjectID == "" {
			return "", "", "", app.ValidationFailed(fmt.Errorf("--subject-type and --subject-id must be given together"))
		}
		if e := app.ValidateEnum("subject type", commentSubjectType, enums.CommentSubjectType); e != nil {
			return "", "", "", e
		}
		return commentSubjectType, commentSubjectID, commentSubjectType + ":" + commentSubjectID, nil
	default:
		return "", "", "", nil
	}
}

var commentAddCmd = &cobra.Command{
	Use:   "add [changeset]",
	Short: "Add a comment to a changeset (defaults to the active changeset)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if strings.TrimSpace(commentBody) == "" {
			return app.ValidationFailed(fmt.Errorf("--body is required"))
		}
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		branch, err := resolveChangeset(ws, args)
		if err != nil {
			return err
		}
		cs, ok, err := store.GetChangesetByBranch(ctx, ws.DB(), branch)
		if err != nil {
			return err
		}
		if !ok {
			return app.NotFound("changeset", branch)
		}
		subjectType, subjectID, subjectRef, err := resolveCommentSubject(ctx, ws, branch)
		if err != nil {
			return err
		}

		var c store.CommentRow
		err = runMutateOnMain(cmd, ws, app.MutateOpts{
			Summary: "comment on " + branch,
			Validate: func(vctx context.Context, r store.Execer) error {
				if commentReply != "" {
					p, ok, e := store.GetComment(vctx, r, commentReply)
					if e != nil {
						return e
					}
					if !ok {
						return app.NotFound("parent comment", commentReply)
					}
					if p.ChangesetID != cs.ID {
						return app.ValidationFailed(fmt.Errorf("parent comment %s belongs to a different changeset", commentReply))
					}
				}
				return nil
			},
		}, func(ctx context.Context, w *app.Write) error {
			authorID, e := store.SeedActor(ctx, w.Tx, w.Actor.Handle, w.Actor.Name)
			if e != nil {
				return e
			}
			c, e = store.AddComment(ctx, w.Tx, cs.ID, authorID, commentReply, commentBody, subjectType, subjectID, commentLocator)
			if e != nil {
				return e
			}
			w.MarkDirty("rev_comment")
			w.MarkDirty("rev_actor")
			return nil
		})
		if err != nil {
			return err
		}
		c.SubjectRef = subjectRef
		emit(c, fmt.Sprintf("commented on %s — %s  (id=%s)", commentAnchorLabel(c, branch), oneLineComment(c.Body), c.ID))
		return nil
	},
}

var commentLsCmd = &cobra.Command{
	Use:   "ls [changeset]",
	Short: "List a changeset's comments (threaded)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		branch, err := resolveChangeset(ws, args)
		if err != nil {
			return err
		}
		cs, ok, err := store.GetChangesetByBranch(ctx, ws.DB(), branch)
		if err != nil {
			return err
		}
		if !ok {
			return app.NotFound("changeset", branch)
		}
		var filter store.CommentFilter
		filter.UnresolvedOnly = commentUnresolved
		if commentSubject != "" {
			st, sid, _, e := resolveCommentSubject(ctx, ws, branch)
			if e != nil {
				return e
			}
			filter.SubjectType, filter.SubjectID = st, sid
		}
		comments, err := store.ListComments(ctx, ws.DB(), cs.ID, filter)
		if err != nil {
			return err
		}
		labelComments(ctx, ws, branch, comments)
		if flagJSON {
			emit(comments, "")
			return nil
		}
		if len(comments) == 0 {
			fmt.Printf("%s: no comments\n", branch)
			return nil
		}
		fmt.Print(printCommentThread(comments))
		return nil
	},
}

var commentShowCmd = &cobra.Command{
	Use:   "show <comment-id>",
	Short: "Show one comment and its direct replies",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		c, ok, err := store.GetComment(ctx, ws.DB(), args[0])
		if err != nil {
			return err
		}
		if !ok {
			return app.NotFound("comment", args[0])
		}
		thread, err := store.ListComments(ctx, ws.DB(), c.ChangesetID, store.CommentFilter{})
		if err != nil {
			return err
		}
		// Keep the target and its direct replies; resolve subject labels best-effort on main.
		var shown []store.CommentRow
		for _, t := range thread {
			if t.ID == c.ID || t.ParentID == c.ID {
				shown = append(shown, t)
			}
		}
		labelCommentsOnMain(ctx, ws, shown)
		if flagJSON {
			emit(shown, "")
			return nil
		}
		fmt.Print(printCommentThread(shown))
		return nil
	},
}

var commentResolveCmd = &cobra.Command{
	Use:   "resolve <comment-id>",
	Short: "Mark a comment thread resolved",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return setCommentResolved(cmd, args[0], true) },
}

var commentReopenCmd = &cobra.Command{
	Use:   "reopen <comment-id>",
	Short: "Mark a comment thread unresolved",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return setCommentResolved(cmd, args[0], false) },
}

func setCommentResolved(cmd *cobra.Command, id string, resolved bool) error {
	ctx := cmd.Context()
	ws, err := connect(ctx)
	if err != nil {
		return err
	}
	defer ws.Close()
	verb := "resolve"
	if !resolved {
		verb = "reopen"
	}
	err = runMutateOnMain(cmd, ws, app.MutateOpts{Summary: verb + " comment " + id},
		func(ctx context.Context, w *app.Write) error {
			ok, e := store.SetCommentResolved(ctx, w.Tx, id, resolved)
			if e != nil {
				return e
			}
			if !ok {
				return app.NotFound("comment", id)
			}
			w.MarkDirty("rev_comment")
			return nil
		})
	if err != nil {
		return err
	}
	emit(map[string]any{"id": id, "resolved": resolved}, fmt.Sprintf("%sd comment %s", verb, id))
	return nil
}

var commentEditCmd = &cobra.Command{
	Use:   "edit <comment-id>",
	Short: "Edit a comment's body",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if strings.TrimSpace(commentBody) == "" {
			return app.ValidationFailed(fmt.Errorf("--body is required"))
		}
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		err = runMutateOnMain(cmd, ws, app.MutateOpts{Summary: "edit comment " + args[0]},
			func(ctx context.Context, w *app.Write) error {
				ok, e := store.UpdateCommentBody(ctx, w.Tx, args[0], commentBody)
				if e != nil {
					return e
				}
				if !ok {
					return app.NotFound("comment", args[0])
				}
				w.MarkDirty("rev_comment")
				return nil
			})
		if err != nil {
			return err
		}
		emit(map[string]any{"id": args[0], "body": commentBody}, fmt.Sprintf("updated comment %s", args[0]))
		return nil
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:   "delete <comment-id>",
	Short: "Delete a comment (replies survive, un-parented)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		var deleted store.CommentRow
		err = runMutateOnMain(cmd, ws, app.MutateOpts{Summary: "delete comment " + args[0]},
			func(ctx context.Context, w *app.Write) error {
				c, ok, e := store.DeleteComment(ctx, w.Tx, args[0])
				if e != nil {
					return e
				}
				if !ok {
					return app.NotFound("comment", args[0])
				}
				deleted = c
				w.MarkDirty("rev_comment")
				return nil
			})
		if err != nil {
			return err
		}
		emit(deleted, fmt.Sprintf("deleted comment %s", args[0]))
		return nil
	},
}

// ---- display helpers ------------------------------------------------------

// labelComments fills SubjectRef on each anchored comment, resolving subject ids against the
// changeset branch (where changeset-new entities live).
func labelComments(ctx context.Context, ws *workspace.Workspace, branch string, comments []store.CommentRow) {
	if !anyAnchored(comments) {
		return
	}
	r, release, err := app.Reader(ctx, ws, branch)
	if err != nil {
		return // best-effort: leave SubjectRef empty
	}
	label, err := app.LabelIndex(ctx, r)
	_ = release()
	if err != nil {
		return
	}
	applyLabels(comments, label)
}

// labelCommentsOnMain is labelComments resolving against main (used by `show`, which has no
// changeset branch in hand). Changeset-new subjects fall back to "type:id".
func labelCommentsOnMain(ctx context.Context, ws *workspace.Workspace, comments []store.CommentRow) {
	if !anyAnchored(comments) {
		return
	}
	label, err := app.LabelIndex(ctx, ws.DB())
	if err != nil {
		return
	}
	applyLabels(comments, label)
}

func applyLabels(comments []store.CommentRow, label func(typ, id string) string) {
	for i := range comments {
		if comments[i].SubjectID != "" {
			comments[i].SubjectRef = label(comments[i].SubjectType, comments[i].SubjectID)
		}
	}
}

func anyAnchored(comments []store.CommentRow) bool {
	for _, c := range comments {
		if c.SubjectID != "" {
			return true
		}
	}
	return false
}

func commentAnchorLabel(c store.CommentRow, branch string) string {
	if c.SubjectRef != "" {
		if c.Locator != "" {
			return c.SubjectRef + "@" + c.Locator
		}
		return c.SubjectRef
	}
	return "changeset " + branch
}

func printCommentThread(comments []store.CommentRow) string {
	present := make(map[string]bool, len(comments))
	for _, c := range comments {
		present[c.ID] = true
	}
	// A comment roots the display if it has no parent, or its parent was filtered out of this
	// listing (e.g. an unresolved reply whose parent thread is resolved) — otherwise it would
	// be orphaned and never printed.
	byParent := map[string][]store.CommentRow{}
	for _, c := range comments {
		key := c.ParentID
		if key != "" && !present[key] {
			key = ""
		}
		byParent[key] = append(byParent[key], c)
	}
	var b strings.Builder
	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		for _, c := range byParent[parent] {
			indent := strings.Repeat("  ", depth)
			mark := " "
			if c.Resolved {
				mark = "✓"
			}
			anchor := "changeset"
			if c.SubjectRef != "" {
				anchor = c.SubjectRef
				if c.Locator != "" {
					anchor += "@" + c.Locator
				}
			}
			fmt.Fprintf(&b, "%s[%s] %s  @%s  (id=%s)\n", indent, mark, c.AuthorHandle, anchor, c.ID)
			fmt.Fprintf(&b, "%s    %s\n", indent, oneLineComment(c.Body))
			walk(c.ID, depth+1)
		}
	}
	walk("", 0)
	return b.String()
}

func oneLineComment(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > 100 {
		return s[:97] + "…"
	}
	return s
}

func init() {
	commentAddCmd.Flags().StringVar(&commentBody, "body", "", "comment text (required)")
	commentAddCmd.Flags().StringVar(&commentSubject, "subject", "", "anchor to an entity: TYPE:key (or a bare fr_key), resolved in the changeset")
	commentAddCmd.Flags().StringVar(&commentSubjectType, "subject-type", "", "explicit subject type (for subjects with no [[TYPE:key]] token)")
	commentAddCmd.Flags().StringVar(&commentSubjectID, "subject-id", "", "explicit subject id (with --subject-type)")
	commentAddCmd.Flags().StringVar(&commentLocator, "locator", "", "field/hunk within the subject (free-form)")
	commentAddCmd.Flags().StringVar(&commentReply, "reply", "", "parent comment id (make this a threaded reply)")

	commentLsCmd.Flags().StringVar(&commentSubject, "subject", "", "only comments anchored to this TYPE:key")
	commentLsCmd.Flags().BoolVar(&commentUnresolved, "unresolved", false, "only unresolved comments")

	commentEditCmd.Flags().StringVar(&commentBody, "body", "", "new comment text (required)")

	commentCmd.AddCommand(commentAddCmd, commentLsCmd, commentShowCmd, commentResolveCmd, commentReopenCmd, commentEditCmd, commentDeleteCmd)
	rootCmd.AddCommand(commentCmd)
}
