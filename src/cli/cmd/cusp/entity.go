package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/endermalkoc/cusp/internal/app"
	"github.com/endermalkoc/cusp/internal/enums"
	"github.com/endermalkoc/cusp/internal/generate"
	"github.com/endermalkoc/cusp/internal/store"
)

var (
	entityDescription string
	entityStatus      string
)

// entityCmd groups entity reads + the entity section / section-type verbs (the section
// trees are attached in section.go). Entities themselves are created by import today.
var entityCmd = &cobra.Command{Use: "entity", Short: "Inspect entities and manage their doc sections"}

var entityLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List entities",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		r, done, err := connectRead(ctx)
		if err != nil {
			return err
		}
		defer done()
		ents, err := store.ListEntities(ctx, r)
		if err != nil {
			return err
		}
		if flagJSON {
			emit(ents, "")
			return nil
		}
		var b strings.Builder
		for _, e := range ents {
			fmt.Fprintf(&b, "%-28s %s\n", e.Name, e.Status)
		}
		fmt.Print(b.String())
		return nil
	},
}

var entityShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show an entity's fields",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rd, done, err := connectRead(ctx)
		if err != nil {
			return err
		}
		defer done()
		e, ok, err := store.GetEntity(ctx, rd, args[0])
		if err != nil {
			return err
		}
		if !ok {
			return app.NotFound("entity", args[0])
		}
		if flagJSON {
			emit(e, "")
			return nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s  (%s)\n", e.Name, e.Status)
		if e.Description != "" {
			fmt.Fprintf(&b, "  description: %s\n", e.Description)
		}
		fmt.Fprintf(&b, "  doc:         %s\n", e.DocPath)
		fmt.Fprintf(&b, "  id:          %s\n", e.ID)
		fmt.Print(b.String())
		return nil
	},
}

var entityEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an entity's description/status (only the flags you pass change)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		if cmd.Flags().NFlag() == 0 {
			return fmt.Errorf("nothing to edit — pass --description and/or --status")
		}
		var e store.EntityRow
		err = runMutate(cmd, ws, app.MutateOpts{
			Summary: "edit entity " + args[0],
			Validate: func(vctx context.Context, r store.Execer) error {
				cur, ok, er := store.GetEntity(vctx, r, args[0])
				if er != nil {
					return er
				}
				if !ok {
					return app.NotFound("entity", args[0])
				}
				if cmd.Flags().Changed("status") {
					if er := app.ValidateEnum("status", entityStatus, enums.EntityStatus); er != nil {
						return er
					}
					cur.Status = entityStatus
				}
				if cmd.Flags().Changed("description") {
					cur.Description = entityDescription
				}
				e = cur
				return nil
			},
		}, func(ctx context.Context, w *app.Write) error {
			if er := store.UpdateEntity(ctx, w.Tx, e.ID, e.Description, e.Status); er != nil {
				return er
			}
			w.MarkDirty("ent_entity")
			return nil
		})
		if err != nil {
			return err
		}
		emit(e, fmt.Sprintf("updated entity %s", e.Name))
		return nil
	},
}

var entityDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an entity and its sections, relationships, and references",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ws, err := connect(ctx)
		if err != nil {
			return err
		}
		defer ws.Close()
		var deleted store.EntityRow
		err = runMutate(cmd, ws, app.MutateOpts{
			Summary: "delete entity " + args[0],
		}, func(ctx context.Context, w *app.Write) error {
			e, ok, er := store.DeleteEntity(ctx, w.Tx, args[0])
			if er != nil {
				return er
			}
			if !ok {
				return app.NotFound("entity", args[0])
			}
			deleted = e
			for _, t := range []string{"ent_entity", "ent_entity_section", "ent_relationship", "req_entity_ref", "req_edge", "pub_external_ref"} {
				w.MarkDirty(t)
			}
			return nil
		})
		if err != nil {
			return err
		}
		emit(deleted, fmt.Sprintf("deleted entity %s", deleted.Name))
		return nil
	},
}

// --- entity tree: entities with their relationships + sections, for the Entities view ---

type entTreeRel struct {
	Name        string `json:"name"`
	DocPath     string `json:"docPath"`
	Cardinality string `json:"cardinality,omitempty"`
	Outgoing    bool   `json:"outgoing"`
}

type entTreeSection struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

type entTreeNode struct {
	Name          string           `json:"name"`
	DocPath       string           `json:"docPath"`
	Relationships []entTreeRel     `json:"relationships"`
	Sections      []entTreeSection `json:"sections"`
}

var entityTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Print the entities with their relationships and sections",
	Long: "Emit every entity with its structured relationships (to other entities) and its prose\n" +
		"sections. Use --json to drive an outline/tree view.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rd, done, err := connectRead(ctx)
		if err != nil {
			return err
		}
		defer done()

		ents, err := store.ListEntities(ctx, rd)
		if err != nil {
			return err
		}
		out := []entTreeNode{}
		for _, e := range ents {
			node := entTreeNode{Name: e.Name, DocPath: e.DocPath, Relationships: []entTreeRel{}, Sections: []entTreeSection{}}
			rels, err := store.ListEntityRelationships(ctx, rd, e.ID)
			if err != nil {
				return err
			}
			for _, rel := range rels {
				node.Relationships = append(node.Relationships, entTreeRel{
					Name: rel.OtherName, DocPath: rel.OtherDocPath, Cardinality: rel.Cardinality, Outgoing: rel.Outgoing,
				})
			}
			secs, err := store.ListEntitySections(ctx, rd, e.ID)
			if err != nil {
				return err
			}
			for _, sec := range secs {
				if sec.Title == "" {
					continue // headingless (preamble) — not navigable
				}
				node.Sections = append(node.Sections, entTreeSection{Key: sec.Key, Title: sec.Title})
			}
			out = append(out, node)
		}

		if flagJSON {
			emit(out, "")
			return nil
		}
		var b strings.Builder
		for _, e := range out {
			fmt.Fprintf(&b, "%s\n", e.Name)
			if len(e.Relationships) > 0 {
				fmt.Fprintf(&b, "  Relationships\n")
				for _, r := range e.Relationships {
					arrow := "←"
					if r.Outgoing {
						arrow = "→"
					}
					fmt.Fprintf(&b, "    %s %s (%s)\n", arrow, r.Name, r.Cardinality)
				}
			}
			if len(e.Sections) > 0 {
				fmt.Fprintf(&b, "  Sections\n")
				for _, s := range e.Sections {
					fmt.Fprintf(&b, "    %s\n", s.Title)
				}
			}
		}
		fmt.Print(b.String())
		return nil
	},
}

var entityRenderFormat string

var entityRenderCmd = &cobra.Command{
	Use:   "render <name-or-path>",
	Short: "Render an entity's document (HTML or Markdown) from the DB to stdout",
	Long: "Render a single entity's document on demand from the database — the render chokepoint\n" +
		"for the Entities view. Accepts an entity name or its doc path (entities/<slug>.md).\n" +
		"--format html emits a self-contained, inline-CSS page; --format md the raw Markdown.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rd, done, err := connectRead(ctx)
		if err != nil {
			return err
		}
		defer done()
		e, ok, err := store.GetEntity(ctx, rd, args[0])
		if err != nil {
			return err
		}
		if !ok {
			// Fall back to resolving the arg as a doc path (e.g. entities/student.md) — how the
			// webview addresses an entity when following a cross-reference link.
			ents, er := store.ListEntities(ctx, rd)
			if er != nil {
				return er
			}
			for _, cand := range ents {
				if cand.DocPath == args[0] || cand.DocPath == args[0]+".md" {
					e, ok = cand, true
					break
				}
			}
		}
		if !ok {
			return app.NotFound("entity", args[0])
		}
		out, err := generate.RenderEntityDoc(ctx, rd, e.ID, e.DocPath, entityRenderFormat)
		if err != nil {
			return err
		}
		fmt.Print(out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Println()
		}
		return nil
	},
}

func init() {
	entityEditCmd.Flags().StringVar(&entityDescription, "description", "", "new description")
	entityEditCmd.Flags().StringVar(&entityStatus, "status", "", "status (draft|active|deprecated)")
	entityRenderCmd.Flags().StringVar(&entityRenderFormat, "format", "html", "render format: html | md")
	entityCmd.AddCommand(entityLsCmd, entityShowCmd, entityTreeCmd, entityRenderCmd, entityEditCmd, entityDeleteCmd)
	rootCmd.AddCommand(entityCmd)
}
