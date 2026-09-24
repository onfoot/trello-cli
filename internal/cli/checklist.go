package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newChecklistCmd() *cobra.Command {
	checklist := &cobra.Command{
		Use:   "checklist",
		Short: "Manage checklists",
		Long:  `Manage checklists and their items on cards.`,
		Example: `  trello checklist list 5abbe4b7ddc1b351ef961417
  trello checklist create 5abbe4b7ddc1b351ef961417 --name "Release"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	checklist.AddCommand(
		a.newChecklistListCmd(),
		a.newChecklistGetCmd(),
		a.newChecklistCreateCmd(),
		a.newChecklistAddItemCmd(),
		a.newChecklistCheckItemCmd(),
	)
	return checklist
}

func (a *App) newChecklistListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <card>",
		Aliases: []string{"ls"},
		Short:   "List checklists on a card",
		Long: `List the checklists on a card.

<card> is the card's id or shortLink.`,
		Example: `  trello checklist list 5abbe4b7ddc1b351ef961417
  trello checklist list AbCdEf01 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runChecklistList(cmd.Context(), args[0])
		},
	}
}

func (a *App) runChecklistList(ctx context.Context, cardRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	checklists, err := client.ListCardChecklists(ctx, cardRef)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(checklists)
	}
	rows := make([][]string, 0, len(checklists))
	for _, cl := range checklists {
		rows = append(rows, []string{cl.ID, cl.Name, strconv.Itoa(len(cl.CheckItems))})
	}
	return out.Table([]string{"ID", "NAME", "ITEMS"}, rows)
}

func (a *App) newChecklistGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a checklist by id",
		Long:  `Get a single checklist, including its items.`,
		Example: `  trello checklist get 5dc9b507756e182c76007621
  trello checklist get 5dc9b507756e182c76007621 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runChecklistGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runChecklistGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	cl, err := client.GetChecklist(ctx, id)
	if err != nil {
		return err
	}
	return a.renderChecklist(cl)
}

func (a *App) newChecklistCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create <card>",
		Short: "Create a checklist on a card",
		Long: `Create a new checklist on a card.

<card> is the card's id or shortLink.`,
		Example: `  trello checklist create 5abbe4b7ddc1b351ef961417 --name "Release"
  trello checklist create AbCdEf01 --name "Release" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return output.WithCode(errors.New("--name is required"), output.ExitUsage)
			}
			return a.runChecklistCreate(cmd.Context(), args[0], name)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "checklist name")
	return cmd
}

func (a *App) runChecklistCreate(ctx context.Context, cardRef, name string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	cl, err := client.CreateChecklist(ctx, cardRef, name)
	if err != nil {
		return err
	}
	return a.renderChecklist(cl)
}

func (a *App) newChecklistAddItemCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-item <checklist-id>",
		Short: "Add an item to a checklist",
		Long:  `Add a new item to a checklist.`,
		Example: `  trello checklist add-item 5dc9b507756e182c76007621 --name "Update docs"
  trello checklist add-item 5dc9b507756e182c76007621 --name "Tag release" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return output.WithCode(errors.New("--name is required"), output.ExitUsage)
			}
			return a.runChecklistAddItem(cmd.Context(), args[0], name)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "item name")
	return cmd
}

func (a *App) runChecklistAddItem(ctx context.Context, checklistID, name string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	item, err := client.AddCheckItem(ctx, checklistID, name)
	if err != nil {
		return err
	}
	return a.renderCheckItem(item)
}

func (a *App) newChecklistCheckItemCmd() *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "check-item <checklist-id> <checkitem-id>",
		Short: "Check or uncheck a checklist item",
		Long: `Set a checklist item's state to complete or incomplete.

--state defaults to complete.`,
		Example: `  trello checklist check-item 5dc9b507756e182c76007621 5dc9b509f02f4314edc4303a
  trello checklist check-item 5dc9b507756e182c76007621 5dc9b509f02f4314edc4303a --state incomplete`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if state != "complete" && state != "incomplete" {
				return output.WithCode(fmt.Errorf("invalid --state %q: must be complete or incomplete", state), output.ExitUsage)
			}
			return a.runChecklistCheckItem(cmd.Context(), args[0], args[1], state)
		},
	}
	cmd.Flags().StringVar(&state, "state", "complete", "item state: complete or incomplete")
	return cmd
}

func (a *App) runChecklistCheckItem(ctx context.Context, checklistID, checkItemID, state string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	item, err := client.SetCheckItemState(ctx, checklistID, checkItemID, state)
	if err != nil {
		return err
	}
	return a.renderCheckItem(item)
}

// renderChecklist prints a checklist in human or JSON form.
func (a *App) renderChecklist(cl *trello.Checklist) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(cl)
	}
	if err := out.KeyValue([][2]string{
		{"ID", cl.ID},
		{"Name", cl.Name},
		{"Board", cl.IDBoard},
		{"Items", strconv.Itoa(len(cl.CheckItems))},
	}); err != nil {
		return err
	}
	if len(cl.CheckItems) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(cl.CheckItems))
	for _, item := range cl.CheckItems {
		rows = append(rows, []string{item.ID, item.Name, item.State})
	}
	return out.Table([]string{"ID", "NAME", "STATE"}, rows)
}

// renderCheckItem prints a check item in human or JSON form.
func (a *App) renderCheckItem(item *trello.CheckItem) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(item)
	}
	return out.KeyValue([][2]string{
		{"ID", item.ID},
		{"Checklist", item.IDChecklist},
		{"Name", item.Name},
		{"State", item.State},
	})
}
