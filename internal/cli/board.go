package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func (a *App) newBoardCmd() *cobra.Command {
	board := &cobra.Command{
		Use:   "board",
		Short: "Manage boards",
		Long:  `Manage Trello boards.`,
		Example: `  trello board list
  trello board list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	board.AddCommand(a.newBoardListCmd())
	return board
}

func (a *App) newBoardListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the boards you are a member of",
		Long: `List the boards the authenticated member belongs to.

Use this to find the id or shortLink of a board to pin with
'trello config init' or --board.`,
		Example: `  trello board list
  trello board list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runBoardList(cmd.Context())
		},
	}
}

func (a *App) runBoardList(ctx context.Context) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	boards, err := client.ListMemberBoards(ctx)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(boards)
	}
	rows := make([][]string, 0, len(boards))
	for _, b := range boards {
		rows = append(rows, []string{b.ID, b.ShortLink, b.Name})
	}
	return out.Table([]string{"ID", "SHORT LINK", "NAME"}, rows)
}
