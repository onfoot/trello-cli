package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newLabelCmd() *cobra.Command {
	label := &cobra.Command{
		Use:   "label",
		Short: "Manage labels",
		Long:  `Manage labels on boards and cards.`,
		Example: `  trello label list
  trello label create "Overdue" --color red
  trello label add 5abbe4b7ddc1b351ef961417 "Overdue"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	label.AddCommand(
		a.newLabelListCmd(),
		a.newLabelGetCmd(),
		a.newLabelCreateCmd(),
		a.newLabelUpdateCmd(),
		a.newLabelDeleteCmd(),
		a.newLabelAddCmd(),
		a.newLabelRemoveCmd(),
	)
	return label
}

func (a *App) newLabelListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the labels on the resolved board",
		Long: `List the labels on the resolved board (--board or the configured
default). The board is resolved by exact name, id, or shortLink.`,
		Example: `  trello label list
  trello label list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLabelList(cmd.Context())
		},
	}
}

func (a *App) runLabelList(ctx context.Context) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	labels, err := client.ListBoardLabels(ctx, board.ID)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(labels)
	}
	rows := make([][]string, 0, len(labels))
	for _, l := range labels {
		rows = append(rows, []string{l.ID, l.Name, l.Color})
	}
	return out.Table([]string{"ID", "NAME", "COLOR"}, rows)
}

func (a *App) newLabelGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a label by id",
		Long:  `Get a single label by its id.`,
		Example: `  trello label get 5abbe4b7ddc1b351ef961419
  trello label get 5abbe4b7ddc1b351ef961419 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLabelGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runLabelGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.GetLabel(ctx, id)
	if err != nil {
		return err
	}
	return a.renderLabel(l)
}

func (a *App) newLabelCreateCmd() *cobra.Command {
	var color string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a label on the resolved board",
		Long: `Create a new label on the resolved board (--board or the configured
default).

--color must be one of: yellow, purple, blue, red, green, orange, black,
sky, pink, lime.`,
		Example: `  trello label create "Overdue" --color red
  trello label create "Needs review" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var colorPtr *string
			if cmd.Flags().Changed("color") {
				if !trello.ValidLabelColor(color) {
					return output.WithCode(fmt.Errorf("invalid --color %q: must be one of %s", color, strings.Join(trello.LabelColors, ", ")), output.ExitUsage)
				}
				colorPtr = &color
			}
			return a.runLabelCreate(cmd.Context(), args[0], colorPtr)
		},
	}
	cmd.Flags().StringVar(&color, "color", "", "label color (one of the standard Trello colors)")
	return cmd
}

func (a *App) runLabelCreate(ctx context.Context, name string, color *string) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.CreateBoardLabel(ctx, board.ID, name, color)
	if err != nil {
		return err
	}
	return a.renderLabel(l)
}

func (a *App) newLabelUpdateCmd() *cobra.Command {
	var name, color string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a label",
		Long: `Update a label's name or color.

At least one of --name or --color is required. --color must be one of:
yellow, purple, blue, red, green, orange, black, sky, pink, lime.`,
		Example: `  trello label update 5abbe4b7ddc1b351ef961419 --name "Overdue"
  trello label update 5abbe4b7ddc1b351ef961419 --color green`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := trello.LabelUpdate{}
			if cmd.Flags().Changed("name") {
				u.Name = &name
			}
			if cmd.Flags().Changed("color") {
				if !trello.ValidLabelColor(color) {
					return output.WithCode(fmt.Errorf("invalid --color %q: must be one of %s", color, strings.Join(trello.LabelColors, ", ")), output.ExitUsage)
				}
				u.Color = &color
			}
			if u.Name == nil && u.Color == nil {
				return output.WithCode(errors.New("at least one of --name or --color is required"), output.ExitUsage)
			}
			return a.runLabelUpdate(cmd.Context(), args[0], u)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new name for the label")
	cmd.Flags().StringVar(&color, "color", "", "new label color (one of the standard Trello colors)")
	return cmd
}

func (a *App) runLabelUpdate(ctx context.Context, id string, u trello.LabelUpdate) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	l, err := client.UpdateLabel(ctx, id, u)
	if err != nil {
		return err
	}
	return a.renderLabel(l)
}

func (a *App) newLabelDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a label",
		Long:  `Permanently delete a label.`,
		Example: `  trello label delete 5abbe4b7ddc1b351ef961419
  trello label delete 5abbe4b7ddc1b351ef961419 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLabelDelete(cmd.Context(), args[0])
		},
	}
}

func (a *App) runLabelDelete(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if err := client.DeleteLabel(ctx, id); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{ID: id, Deleted: true})
	}
	out.Printf("Deleted label %s\n", id)
	return nil
}

func (a *App) newLabelAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <card> <label>",
		Short: "Add a label to a card",
		Long: `Add a label to a card.

<card> is the card's id or shortLink. <label> is the label's id or exact
name; the label is resolved against the card's board.`,
		Example: `  trello label add 5abbe4b7ddc1b351ef961417 5abbe4b7ddc1b351ef961419
  trello label add AbCdEf01 "Overdue" --json`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLabelAdd(cmd.Context(), args[0], args[1])
		},
	}
}

func (a *App) runLabelAdd(ctx context.Context, cardRef, labelRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	board, err := client.GetCardBoard(ctx, cardRef)
	if err != nil {
		return err
	}
	l, err := client.ResolveLabel(ctx, board.ID, labelRef)
	if err != nil {
		return err
	}
	if err := client.AddLabelToCard(ctx, cardRef, l.ID); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Card  string `json:"card"`
			Label string `json:"label"`
			Added bool   `json:"added"`
		}{Card: cardRef, Label: l.ID, Added: true})
	}
	out.Printf("Added label %s to card %s\n", l.ID, cardRef)
	return nil
}

func (a *App) newLabelRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <card> <label>",
		Short: "Remove a label from a card",
		Long: `Remove a label from a card.

<card> is the card's id or shortLink. <label> is the label's id or exact
name; the label is resolved against the card's board.`,
		Example: `  trello label remove 5abbe4b7ddc1b351ef961417 5abbe4b7ddc1b351ef961419
  trello label remove AbCdEf01 "Overdue" --json`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLabelRemove(cmd.Context(), args[0], args[1])
		},
	}
}

func (a *App) runLabelRemove(ctx context.Context, cardRef, labelRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	board, err := client.GetCardBoard(ctx, cardRef)
	if err != nil {
		return err
	}
	l, err := client.ResolveLabel(ctx, board.ID, labelRef)
	if err != nil {
		return err
	}
	if err := client.RemoveLabelFromCard(ctx, cardRef, l.ID); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Card    string `json:"card"`
			Label   string `json:"label"`
			Removed bool   `json:"removed"`
		}{Card: cardRef, Label: l.ID, Removed: true})
	}
	out.Printf("Removed label %s from card %s\n", l.ID, cardRef)
	return nil
}

// renderLabel prints a label in human or JSON form.
func (a *App) renderLabel(l *trello.Label) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(l)
	}
	return out.KeyValue([][2]string{
		{"ID", l.ID},
		{"Name", l.Name},
		{"Color", l.Color},
		{"Board", l.IDBoard},
	})
}
