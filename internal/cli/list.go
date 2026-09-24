package cli

import (
	"context"
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newListCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "Manage lists",
		Long:  `Manage lists on a board.`,
		Example: `  trello list list
  trello list create "To do"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	list.AddCommand(
		a.newListListCmd(),
		a.newListGetCmd(),
		a.newListCreateCmd(),
		a.newListUpdateCmd(),
		a.newListArchiveCmd(),
	)
	return list
}

func (a *App) newListListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the lists on the resolved board",
		Long: `List the lists on the resolved board (--board or the configured default).

The board is resolved by exact name, id, or shortLink.`,
		Example: `  trello list list
  trello list list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runListList(cmd.Context())
		},
	}
}

func (a *App) runListList(ctx context.Context) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	lists, err := client.ListBoardLists(ctx, board.ID)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(lists)
	}
	rows := make([][]string, 0, len(lists))
	for _, l := range lists {
		rows = append(rows, []string{l.ID, l.Name, strconv.FormatBool(l.Closed), formatPos(l.Pos)})
	}
	return out.Table([]string{"ID", "NAME", "CLOSED", "POS"}, rows)
}

func (a *App) newListGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a list by id",
		Long:  `Get a single list by its id.`,
		Example: `  trello list get 5abbe4b7ddc1b351ef961415
  trello list get 5abbe4b7ddc1b351ef961415 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runListGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runListGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.GetList(ctx, id)
	if err != nil {
		return err
	}
	return a.renderList(l)
}

func (a *App) newListCreateCmd() *cobra.Command {
	var pos float64
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a list on the resolved board",
		Long: `Create a new list on the resolved board (--board or the configured
default).`,
		Example: `  trello list create "To do"
  trello list create "Done" --pos 65535`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var posPtr *float64
			if cmd.Flags().Changed("pos") {
				posPtr = &pos
			}
			return a.runListCreate(cmd.Context(), args[0], posPtr)
		},
	}
	cmd.Flags().Float64Var(&pos, "pos", 0, "position of the list on the board")
	return cmd
}

func (a *App) runListCreate(ctx context.Context, name string, pos *float64) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.CreateList(ctx, board.ID, name, pos)
	if err != nil {
		return err
	}
	return a.renderList(l)
}

func (a *App) newListUpdateCmd() *cobra.Command {
	var name string
	var closed bool
	var pos float64
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a list",
		Long: `Update a list's name, closed state, or position.

At least one of --name, --closed, or --pos is required.`,
		Example: `  trello list update 5abbe4b7ddc1b351ef961415 --name "In progress"
  trello list update 5abbe4b7ddc1b351ef961415 --closed=true`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := trello.ListUpdate{}
			if cmd.Flags().Changed("name") {
				u.Name = &name
			}
			if cmd.Flags().Changed("closed") {
				u.Closed = &closed
			}
			if cmd.Flags().Changed("pos") {
				u.Pos = &pos
			}
			if u.Name == nil && u.Closed == nil && u.Pos == nil {
				return output.WithCode(errors.New("at least one of --name, --closed, or --pos is required"), output.ExitUsage)
			}
			return a.runListUpdate(cmd.Context(), args[0], u)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new name for the list")
	cmd.Flags().BoolVar(&closed, "closed", false, "archive (true) or unarchive (false) the list")
	cmd.Flags().Float64Var(&pos, "pos", 0, "new position of the list")
	return cmd
}

func (a *App) runListUpdate(ctx context.Context, id string, u trello.ListUpdate) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.UpdateList(ctx, id, u)
	if err != nil {
		return err
	}
	return a.renderList(l)
}

func (a *App) newListArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <id>",
		Short: "Archive a list",
		Long:  `Archive (close) a list by setting closed=true.`,
		Example: `  trello list archive 5abbe4b7ddc1b351ef961415
  trello list archive 5abbe4b7ddc1b351ef961415 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runListArchive(cmd.Context(), args[0])
		},
	}
}

func (a *App) runListArchive(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.ArchiveList(ctx, id)
	if err != nil {
		return err
	}
	return a.renderList(l)
}

// renderList prints a list in human or JSON form.
func (a *App) renderList(l *trello.List) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(l)
	}
	return out.KeyValue([][2]string{
		{"ID", l.ID},
		{"Name", l.Name},
		{"Board", l.IDBoard},
		{"Closed", strconv.FormatBool(l.Closed)},
		{"Pos", formatPos(l.Pos)},
	})
}
