package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newSearchCmd() *cobra.Command {
	var boards, cards, members, organizations bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Trello",
		Long: `Search boards, cards, members, and organizations.

By default cards and boards are searched. Use the model-type flags to
control what is searched. --board scopes the search to a single board
(matched by exact name, id, or shortLink).`,
		Example: `  trello search "release"
  trello search "bentley" --members
  trello search "ship it" --cards --board "Trello Platform Changes" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := trello.SearchOptions{}
			switch {
			case boards || cards || members || organizations:
				if boards {
					opts.ModelTypes = append(opts.ModelTypes, "boards")
				}
				if cards {
					opts.ModelTypes = append(opts.ModelTypes, "cards")
				}
				if members {
					opts.ModelTypes = append(opts.ModelTypes, "members")
				}
				if organizations {
					opts.ModelTypes = append(opts.ModelTypes, "organizations")
				}
			default:
				opts.ModelTypes = []string{"cards", "boards"}
			}
			return a.runSearch(cmd.Context(), args[0], opts)
		},
	}
	cmd.Flags().BoolVar(&boards, "boards", false, "search boards")
	cmd.Flags().BoolVar(&cards, "cards", false, "search cards")
	cmd.Flags().BoolVar(&members, "members", false, "search members")
	cmd.Flags().BoolVar(&organizations, "organizations", false, "search organizations")
	return cmd
}

func (a *App) runSearch(ctx context.Context, query string, opts trello.SearchOptions) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	// Scope to the explicitly given --board only; a config default must not
	// silently narrow search results.
	if a.opts.Board != "" {
		board, err := a.board(ctx)
		if err != nil {
			return err
		}
		opts.IDBoards = board.ID
	}
	res, err := client.Search(ctx, query, opts)
	if err != nil {
		return err
	}
	return a.renderSearch(res, opts.ModelTypes)
}

// renderSearch prints search results grouped by model type in human form, or
// the structured result object in JSON form. In human mode every requested
// model type is printed — a group with no results shows a "(no results)"
// indicator so empty results are never silently dropped.
func (a *App) renderSearch(res *trello.SearchResult, modelTypes []string) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(res)
	}
	requested := make(map[string]bool, len(modelTypes))
	for _, mt := range modelTypes {
		requested[mt] = true
	}

	boardRows := make([][]string, 0, len(res.Boards))
	for _, b := range res.Boards {
		boardRows = append(boardRows, []string{b.ID, b.Name})
	}
	cardRows := make([][]string, 0, len(res.Cards))
	for _, c := range res.Cards {
		cardRows = append(cardRows, []string{c.ID, c.Name})
	}
	memberRows := make([][]string, 0, len(res.Members))
	for _, m := range res.Members {
		memberRows = append(memberRows, []string{m.ID, m.Username, m.FullName})
	}
	orgRows := make([][]string, 0, len(res.Organizations))
	for _, o := range res.Organizations {
		orgRows = append(orgRows, []string{o.ID, o.DisplayName})
	}

	printed := false
	printGroup := func(title string, headers []string, rows [][]string) error {
		if printed {
			out.Printf("\n")
		}
		printed = true
		out.Printf("%s\n", out.Bold(title))
		if len(rows) == 0 {
			out.Printf("  (no results)\n")
			return nil
		}
		return out.Table(headers, rows)
	}

	if requested["boards"] {
		if err := printGroup("Boards", []string{"ID", "NAME"}, boardRows); err != nil {
			return err
		}
	}
	if requested["cards"] {
		if err := printGroup("Cards", []string{"ID", "NAME"}, cardRows); err != nil {
			return err
		}
	}
	if requested["members"] {
		if err := printGroup("Members", []string{"ID", "USERNAME", "FULL NAME"}, memberRows); err != nil {
			return err
		}
	}
	if requested["organizations"] {
		if err := printGroup("Organizations", []string{"ID", "NAME"}, orgRows); err != nil {
			return err
		}
	}
	return nil
}
