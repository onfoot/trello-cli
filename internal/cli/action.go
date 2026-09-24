package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newActionCmd() *cobra.Command {
	action := &cobra.Command{
		Use:   "action",
		Short: "Manage actions",
		Long:  `List and inspect board and card actions.`,
		Example: `  trello action list
  trello action list --card 5abbe4b7ddc1b351ef961417
  trello action get 5dc9b507756e182c76007621`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	action.AddCommand(
		a.newActionListCmd(),
		a.newActionGetCmd(),
	)
	return action
}

func (a *App) newActionListCmd() *cobra.Command {
	var cardRef, filter string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List actions",
		Long: `List actions on the resolved board (--board or the configured
default), or on a specific card with --card.

--filter is a comma-separated list of action types (default "all").`,
		Example: `  trello action list
  trello action list --card 5abbe4b7ddc1b351ef961417
  trello action list --filter commentCard,updateCard --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runActionList(cmd.Context(), cardRef, filter)
		},
	}
	cmd.Flags().StringVar(&cardRef, "card", "", "list actions on this card instead (id or shortLink)")
	cmd.Flags().StringVar(&filter, "filter", "all", "comma-separated action types (default all)")
	return cmd
}

func (a *App) runActionList(ctx context.Context, cardRef, filter string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	var actions []trello.Action
	if cardRef != "" {
		actions, err = client.ListCardActions(ctx, cardRef, filter)
		if err != nil {
			return err
		}
	} else {
		board, err := a.board(ctx)
		if err != nil {
			return err
		}
		actions, err = client.ListBoardActions(ctx, board.ID, filter)
		if err != nil {
			return err
		}
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(actions)
	}
	rows := make([][]string, 0, len(actions))
	for _, act := range actions {
		rows = append(rows, []string{act.ID, act.Type, formatTimePtr(act.Date), memberName(act.MemberCreator, act.IDMemberCreator)})
	}
	return out.Table([]string{"ID", "TYPE", "DATE", "MEMBER"}, rows)
}

func (a *App) newActionGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an action by id",
		Long:  `Get a single action by id.`,
		Example: `  trello action get 5dc9b507756e182c76007621
  trello action get 5dc9b507756e182c76007621 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runActionGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runActionGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	act, err := client.GetAction(ctx, id)
	if err != nil {
		return err
	}
	return a.renderAction(act)
}

// memberName renders the member creator of an action: username when the
// member object is present, otherwise the raw id, otherwise "unknown".
func memberName(m *trello.Member, id string) string {
	if m != nil && m.Username != "" {
		return m.Username
	}
	if id != "" {
		return id
	}
	return "unknown"
}

// actionDataSummary renders a readable summary of an action's data.
func actionDataSummary(d trello.ActionData) string {
	parts := make([]string, 0, 4)
	if d.Text != "" {
		parts = append(parts, d.Text)
	}
	if d.Card != nil && d.Card.Name != "" {
		parts = append(parts, "card "+d.Card.Name)
	}
	if d.List != nil && d.List.Name != "" {
		parts = append(parts, "list "+d.List.Name)
	}
	if d.Board != nil && d.Board.Name != "" {
		parts = append(parts, "board "+d.Board.Name)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " on ")
}

// renderAction prints an action in human or JSON form.
func (a *App) renderAction(act *trello.Action) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(act)
	}
	return out.KeyValue([][2]string{
		{"ID", act.ID},
		{"Type", act.Type},
		{"Date", formatTimePtr(act.Date)},
		{"Member", memberName(act.MemberCreator, act.IDMemberCreator)},
		{"Data", actionDataSummary(act.Data)},
	})
}
