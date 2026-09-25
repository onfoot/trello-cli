package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newCardCmd() *cobra.Command {
	card := &cobra.Command{
		Use:   "card",
		Short: "Manage cards",
		Long:  `Manage cards on a board.`,
		Example: `  trello card list
  trello card create "Ship it" --list "To do"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	card.AddCommand(
		a.newCardListCmd(),
		a.newCardGetCmd(),
		a.newCardCreateCmd(),
		a.newCardUpdateCmd(),
		a.newCardMoveCmd(),
		a.newCardArchiveCmd(),
		a.newCardDeleteCmd(),
	)
	return card
}

func (a *App) newCardListCmd() *cobra.Command {
	var listRef string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cards on the resolved board",
		Long: `List cards on the resolved board (--board or the configured default).

With --list, only cards in the given list are shown. The list reference is
matched by exact id or name.`,
		Example: `  trello card list
  trello card list --list "To do"
  trello card list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCardList(cmd.Context(), listRef)
		},
	}
	cmd.Flags().StringVar(&listRef, "list", "", "only show cards in this list (exact id or name)")
	return cmd
}

func (a *App) runCardList(ctx context.Context, listRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	var cards []trello.Card
	if listRef != "" {
		listID, err := a.resolveListRef(ctx, client, listRef)
		if err != nil {
			return err
		}
		cards, err = client.ListListCards(ctx, listID)
		if err != nil {
			return err
		}
	} else {
		board, err := a.board(ctx)
		if err != nil {
			return err
		}
		cards, err = client.ListBoardCards(ctx, board.ID)
		if err != nil {
			return err
		}
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(cards)
	}
	rows := make([][]string, 0, len(cards))
	for _, c := range cards {
		rows = append(rows, []string{c.ID, c.Name, formatCardLabels(c.Labels), c.IDList, strconv.FormatBool(c.Closed), formatDue(c.Due)})
	}
	return out.Table([]string{"ID", "NAME", "LABELS", "LIST", "CLOSED", "DUE"}, rows)
}

// resolveListRef resolves a --list reference to a list id. A 24-hex id is
// used directly (no board needed); a name requires the board to be resolved
// first.
func (a *App) resolveListRef(ctx context.Context, client *trello.Client, ref string) (string, error) {
	if trello.IsTrelloID(ref) {
		return ref, nil
	}
	board, err := a.board(ctx)
	if err != nil {
		return "", err
	}
	l, err := client.ResolveList(ctx, board.ID, ref)
	if err != nil {
		return "", err
	}
	return l.ID, nil
}

func (a *App) newCardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a card by id",
		Long:  `Get a single card by its id. The output includes the card's labels.`,
		Example: `  trello card get 5abbe4b7ddc1b351ef961414
  trello card get 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCardGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCardGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	card, err := client.GetCard(ctx, id)
	if err != nil {
		return err
	}
	return a.renderCard(card)
}

func (a *App) newCardCreateCmd() *cobra.Command {
	var listRef, desc, due string
	var pos float64
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a card",
		Long: `Create a new card in a list on the resolved board.

--list is required and is matched against the board's lists by exact id or
name. --due must be an RFC3339 timestamp.`,
		Example: `  trello card create "Ship it" --list "To do"
  trello card create "Fix bug" --list 5abbe4b7ddc1b351ef961415 --desc "details" --due 2026-09-04T12:00:00Z`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if listRef == "" {
				return output.WithCode(errors.New("--list is required (exact list id or name)"), output.ExitUsage)
			}
			var posPtr *float64
			if cmd.Flags().Changed("pos") {
				posPtr = &pos
			}
			var duePtr *string
			if cmd.Flags().Changed("due") {
				if _, err := time.Parse(time.RFC3339, due); err != nil {
					return output.WithCode(fmt.Errorf("invalid --due %q: must be an RFC3339 timestamp (e.g. 2026-09-04T12:00:00Z)", due), output.ExitUsage)
				}
				duePtr = &due
			}
			return a.runCardCreate(cmd.Context(), trello.CardCreate{
				IDList: listRef,
				Name:   args[0],
				Desc:   desc,
				Pos:    posPtr,
				Due:    duePtr,
			})
		},
	}
	cmd.Flags().StringVar(&listRef, "list", "", "destination list (exact id or name)")
	cmd.Flags().StringVar(&desc, "desc", "", "card description")
	cmd.Flags().Float64Var(&pos, "pos", 0, "position of the card in its list")
	cmd.Flags().StringVar(&due, "due", "", "due date (RFC3339)")
	return cmd
}

func (a *App) runCardCreate(ctx context.Context, in trello.CardCreate) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	listID, err := a.resolveListRef(ctx, client, in.IDList)
	if err != nil {
		return err
	}
	in.IDList = listID
	card, err := client.CreateCard(ctx, in)
	if err != nil {
		return err
	}
	return a.renderCard(card)
}

func (a *App) newCardUpdateCmd() *cobra.Command {
	var name, desc, due, listRef string
	var closed bool
	var pos float64
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a card",
		Long: `Update a card's fields.

At least one of --name, --desc, --due, --closed, --pos, or --list is
required. --list moves the card to the given list (matched by exact id or
name against the board's lists). --due must be an RFC3339 timestamp; pass
--due null (or --due "") to clear the due date.`,
		Example: `  trello card update 5abbe4b7ddc1b351ef961414 --name "Renamed"
  trello card update 5abbe4b7ddc1b351ef961414 --closed=true --list "Done"
  trello card update 5abbe4b7ddc1b351ef961414 --due null`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := trello.CardUpdate{}
			if cmd.Flags().Changed("name") {
				u.Name = &name
			}
			if cmd.Flags().Changed("desc") {
				u.Desc = &desc
			}
			if cmd.Flags().Changed("closed") {
				u.Closed = &closed
			}
			if cmd.Flags().Changed("pos") {
				u.Pos = &pos
			}
			if cmd.Flags().Changed("due") {
				// "null" and "" are clear sentinels that send due=null.
				if due == "" || due == "null" {
					clearDue := "null"
					u.Due = &clearDue
				} else {
					if _, err := time.Parse(time.RFC3339, due); err != nil {
						return output.WithCode(fmt.Errorf("invalid --due %q: must be an RFC3339 timestamp (e.g. 2026-09-04T12:00:00Z), or null/empty to clear", due), output.ExitUsage)
					}
					u.Due = &due
				}
			}
			if cmd.Flags().Changed("list") {
				u.IDList = &listRef
			}
			if u.Name == nil && u.Desc == nil && u.Closed == nil && u.Pos == nil && u.Due == nil && u.IDList == nil {
				return output.WithCode(errors.New("at least one of --name, --desc, --due, --closed, --pos, or --list is required"), output.ExitUsage)
			}
			return a.runCardUpdate(cmd.Context(), args[0], u)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new name for the card")
	cmd.Flags().StringVar(&desc, "desc", "", "new description for the card")
	cmd.Flags().StringVar(&due, "due", "", "new due date (RFC3339); null or empty clears it")
	cmd.Flags().BoolVar(&closed, "closed", false, "archive (true) or unarchive (false) the card")
	cmd.Flags().Float64Var(&pos, "pos", 0, "new position of the card")
	cmd.Flags().StringVar(&listRef, "list", "", "move the card to this list (exact id or name)")
	return cmd
}

func (a *App) runCardUpdate(ctx context.Context, id string, u trello.CardUpdate) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if u.IDList != nil {
		listID, err := a.resolveListRef(ctx, client, *u.IDList)
		if err != nil {
			return err
		}
		u.IDList = &listID
	}
	card, err := client.UpdateCard(ctx, id, u)
	if err != nil {
		return err
	}
	return a.renderCard(card)
}

func (a *App) newCardMoveCmd() *cobra.Command {
	var listRef string
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a card to another list",
		Long: `Move a card to another list on the resolved board.

--list is required and is matched against the board's lists by exact id or
name.`,
		Example: `  trello card move 5abbe4b7ddc1b351ef961414 --list "Done"`,
		Args:    exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if listRef == "" {
				return output.WithCode(errors.New("--list is required (exact list id or name)"), output.ExitUsage)
			}
			return a.runCardMove(cmd.Context(), args[0], listRef)
		},
	}
	cmd.Flags().StringVar(&listRef, "list", "", "destination list (exact id or name)")
	return cmd
}

func (a *App) runCardMove(ctx context.Context, id, listRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	listID, err := a.resolveListRef(ctx, client, listRef)
	if err != nil {
		return err
	}
	card, err := client.MoveCard(ctx, id, listID)
	if err != nil {
		return err
	}
	return a.renderCard(card)
}

func (a *App) newCardArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <id>",
		Short: "Archive a card",
		Long:  `Archive (close) a card by setting closed=true.`,
		Example: `  trello card archive 5abbe4b7ddc1b351ef961414
  trello card archive 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCardArchive(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCardArchive(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	card, err := client.ArchiveCard(ctx, id)
	if err != nil {
		return err
	}
	return a.renderCard(card)
}

func (a *App) newCardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a card",
		Long:  `Permanently delete a card.`,
		Example: `  trello card delete 5abbe4b7ddc1b351ef961414
  trello card delete 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCardDelete(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCardDelete(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if err := client.DeleteCard(ctx, id); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{ID: id, Deleted: true})
	}
	out.Printf("Deleted card %s\n", id)
	return nil
}

// renderCard prints a card in human or JSON form.
func (a *App) renderCard(c *trello.Card) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(c)
	}
	rows := [][2]string{
		{"ID", c.ID},
		{"Name", c.Name},
	}
	if c.Desc != "" {
		rows = append(rows, [2]string{"Desc", c.Desc})
	}
	rows = append(rows,
		[2]string{"List", c.IDList},
		[2]string{"Board", c.IDBoard},
		[2]string{"Labels", formatCardLabels(c.Labels)},
		[2]string{"Closed", strconv.FormatBool(c.Closed)},
		[2]string{"Due", formatDue(c.Due)},
		[2]string{"Pos", formatPos(c.Pos)},
		[2]string{"URL", c.URL},
	)
	return out.KeyValue(rows)
}
