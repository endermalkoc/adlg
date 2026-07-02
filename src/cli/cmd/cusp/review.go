package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/endermalkoc/cusp/internal/app"
	"github.com/endermalkoc/cusp/internal/enums"
	"github.com/endermalkoc/cusp/internal/store"
)

var (
	reviewVerdict string
	reviewSummary string
)

var reviewCmd = &cobra.Command{
	Use:   "review [changeset]",
	Short: "Record a review verdict on a changeset (approve | deny | request_changes)",
	Long: "Set your verdict on a changeset under review. One verdict per reviewer per changeset —\n" +
		"re-running replaces your prior verdict. The verdict also moves the changeset's status\n" +
		"(approve→approved, request_changes→changes_requested, deny→denied); merge/abandon stay\n" +
		"with the `changeset` verbs. Verdicts live on `main`, so they survive the merge.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if reviewVerdict == "" {
			return app.ValidationFailed(fmt.Errorf("--verdict is required (one of %v)", enums.ReviewVerdict))
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
		newStatus := enums.StatusForVerdict(reviewVerdict)

		var r store.ReviewRow
		err = runMutateOnMain(cmd, ws, app.MutateOpts{
			Summary: fmt.Sprintf("review %s: %s", branch, reviewVerdict),
			Validate: func(vctx context.Context, rd store.Execer) error {
				return app.ValidateEnum("verdict", reviewVerdict, enums.ReviewVerdict)
			},
		}, func(ctx context.Context, w *app.Write) error {
			reviewerID, e := store.SeedActor(ctx, w.Tx, w.Actor.Handle, w.Actor.Name)
			if e != nil {
				return e
			}
			r, e = store.UpsertReview(ctx, w.Tx, cs.ID, reviewerID, reviewVerdict, reviewSummary)
			if e != nil {
				return e
			}
			w.MarkDirty("rev_review")
			w.MarkDirty("rev_actor")
			if newStatus != "" {
				if e := store.SetChangesetStatus(ctx, w.Tx, cs.ID, newStatus); e != nil {
					return e
				}
				w.MarkDirty("rev_changeset")
			}
			return nil
		})
		if err != nil {
			return err
		}
		emit(r, fmt.Sprintf("%s reviewed %s → %s (changeset now %s)", r.ReviewerHandle, branch, reviewVerdict, newStatus))
		return nil
	},
}

var reviewLsCmd = &cobra.Command{
	Use:   "ls [changeset]",
	Short: "List a changeset's review verdicts",
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
		reviews, err := store.ListReviews(ctx, ws.DB(), cs.ID)
		if err != nil {
			return err
		}
		if flagJSON {
			emit(reviews, "")
			return nil
		}
		if len(reviews) == 0 {
			fmt.Printf("%s: no reviews\n", branch)
			return nil
		}
		var b strings.Builder
		for _, r := range reviews {
			fmt.Fprintf(&b, "%-16s %-16s %s\n", r.ReviewerHandle, r.Verdict, oneLineComment(r.Summary))
		}
		fmt.Print(b.String())
		return nil
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewVerdict, "verdict", "", "approve | deny | request_changes")
	reviewCmd.Flags().StringVar(&reviewSummary, "summary", "", "optional review summary")
	reviewCmd.AddCommand(reviewLsCmd)
	rootCmd.AddCommand(reviewCmd)
}
