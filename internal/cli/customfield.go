package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newCustomFieldCmd() *cobra.Command {
	cf := &cobra.Command{
		Use:   "customfield",
		Short: "Manage custom fields",
		Long:  `Manage custom fields on boards and cards.`,
		Example: `  trello customfield list
  trello customfield set 5abbe4b7ddc1b351ef961417 "Priority" High
  trello customfield create "ETA" --type date`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cf.AddCommand(
		a.newCustomFieldListCmd(),
		a.newCustomFieldGetCmd(),
		a.newCustomFieldSetCmd(),
		a.newCustomFieldCreateCmd(),
		a.newCustomFieldDeleteCmd(),
	)
	return cf
}

func (a *App) newCustomFieldListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the custom fields on the resolved board",
		Long: `List the custom fields on the resolved board (--board or the
configured default). The board is resolved by exact name, id, or
shortLink.`,
		Example: `  trello customfield list
  trello customfield list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCustomFieldList(cmd.Context())
		},
	}
}

func (a *App) runCustomFieldList(ctx context.Context) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	fields, err := client.ListBoardCustomFields(ctx, board.ID)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(fields)
	}
	rows := make([][]string, 0, len(fields))
	for _, f := range fields {
		rows = append(rows, []string{f.ID, f.Name, f.Type})
	}
	return out.Table([]string{"ID", "NAME", "TYPE"}, rows)
}

func (a *App) newCustomFieldGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a custom field by id",
		Long:  `Get a single custom field, including its options.`,
		Example: `  trello customfield get 5ab10be237846c43015f108e
  trello customfield get 5ab10be237846c43015f108e --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCustomFieldGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCustomFieldGet(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	f, err := client.GetCustomField(ctx, id)
	if err != nil {
		return err
	}
	return a.renderCustomField(f)
}

func (a *App) newCustomFieldSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <card> <field> <value>",
		Short: "Set a card's custom field value",
		Long: `Set a card's value for a custom field.

<card> is the card's id or shortLink. <field> is the field's id or exact
name, resolved against the card's board. The value is interpreted
according to the field's type:

  text     any string
  number   a number
  date     an RFC3339 timestamp
  checkbox true or false
  list     an option matched by exact name or id`,
		Example: `  trello customfield set 5abbe4b7ddc1b351ef961417 "Priority" High
  trello customfield set AbCdEf01 "ETA" 2026-09-04T12:00:00Z --json`,
		Args: exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCustomFieldSet(cmd.Context(), args[0], args[1], args[2])
		},
	}
}

func (a *App) runCustomFieldSet(ctx context.Context, cardRef, fieldRef, value string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	board, err := client.GetCardBoard(ctx, cardRef)
	if err != nil {
		return err
	}
	field, err := client.ResolveCustomField(ctx, board.ID, fieldRef)
	if err != nil {
		return err
	}
	if field.Type == "list" {
		// Options are only reliably present on the full field object.
		full, err := client.GetCustomField(ctx, field.ID)
		if err != nil {
			return err
		}
		field = full
	}
	payload, err := buildCustomFieldPayload(field, value)
	if err != nil {
		return err
	}
	if err := client.SetCustomFieldItem(ctx, cardRef, field.ID, payload); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Card  string `json:"card"`
			Field string `json:"field"`
			Value string `json:"value"`
			Set   bool   `json:"set"`
		}{Card: cardRef, Field: field.ID, Value: value, Set: true})
	}
	out.Printf("Set custom field %s to %q on card %s\n", field.ID, value, cardRef)
	return nil
}

// buildCustomFieldPayload builds the type-shaped JSON body for setting a
// custom field value. Usage errors (bad number/date/checkbox) exit 2; a list
// option that matches nothing exits 3.
func buildCustomFieldPayload(field *trello.CustomField, value string) (any, error) {
	switch field.Type {
	case "text":
		return map[string]any{"value": map[string]any{"text": value}}, nil
	case "number":
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, output.WithCode(fmt.Errorf("invalid value %q for number custom field: must be a number", value), output.ExitUsage)
		}
		return map[string]any{"value": map[string]any{"number": n}}, nil
	case "date":
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return nil, output.WithCode(fmt.Errorf("invalid value %q for date custom field: must be an RFC3339 timestamp", value), output.ExitUsage)
		}
		return map[string]any{"value": map[string]any{"date": value}}, nil
	case "checkbox":
		var b bool
		switch value {
		case "true":
			b = true
		case "false":
			b = false
		default:
			return nil, output.WithCode(fmt.Errorf("invalid value %q for checkbox custom field: must be true or false", value), output.ExitUsage)
		}
		return map[string]any{"value": map[string]any{"checked": b}}, nil
	case "list":
		for i := range field.Options {
			opt := &field.Options[i]
			if opt.ID == value || opt.Value.Text == value {
				return map[string]any{"idValue": opt.ID}, nil
			}
		}
		return nil, output.WithCode(fmt.Errorf("option %q not found on the list custom field (matched by exact option name or id)", value), output.ExitConfig)
	default:
		return nil, fmt.Errorf("custom field %q has unsupported type %q", field.Name, field.Type)
	}
}

func (a *App) newCustomFieldCreateCmd() *cobra.Command {
	var fieldType, optionsFlag string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a custom field on the resolved board",
		Long: `Create a new custom field on the resolved board (--board or the
configured default).

--type is required and must be one of: checkbox, list, number, text, date.
--options is a comma-separated list of option names and is only valid for
type=list.`,
		Example: `  trello customfield create "Priority" --type list --options High,Medium,Low
  trello customfield create "ETA" --type date --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if fieldType == "" {
				return output.WithCode(errors.New("--type is required (one of checkbox, list, number, text, date)"), output.ExitUsage)
			}
			if !trello.ValidCustomFieldType(fieldType) {
				return output.WithCode(fmt.Errorf("invalid --type %q: must be one of %s", fieldType, strings.Join(trello.CustomFieldTypes, ", ")), output.ExitUsage)
			}
			var options []string
			if cmd.Flags().Changed("options") {
				if fieldType != "list" {
					return output.WithCode(fmt.Errorf("--options is only valid for type=list, not %q", fieldType), output.ExitUsage)
				}
				options = splitOptions(optionsFlag)
			}
			return a.runCustomFieldCreate(cmd.Context(), args[0], fieldType, options)
		},
	}
	cmd.Flags().StringVar(&fieldType, "type", "", "custom field type (checkbox, list, number, text, date)")
	cmd.Flags().StringVar(&optionsFlag, "options", "", "comma-separated option names (list type only)")
	return cmd
}

// splitOptions splits a comma-separated option list, trimming and dropping
// empty entries.
func splitOptions(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (a *App) runCustomFieldCreate(ctx context.Context, name, fieldType string, options []string) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	f, err := client.CreateCustomField(ctx, board.ID, name, fieldType, options)
	if err != nil {
		return err
	}
	return a.renderCustomField(f)
}

func (a *App) newCustomFieldDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a custom field",
		Long:  `Permanently delete a custom field definition.`,
		Example: `  trello customfield delete 5ab10be237846c43015f108e
  trello customfield delete 5ab10be237846c43015f108e --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCustomFieldDelete(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCustomFieldDelete(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if err := client.DeleteCustomField(ctx, id); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{ID: id, Deleted: true})
	}
	out.Printf("Deleted custom field %s\n", id)
	return nil
}

// renderCustomField prints a custom field in human or JSON form.
func (a *App) renderCustomField(f *trello.CustomField) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(f)
	}
	if err := out.KeyValue([][2]string{
		{"ID", f.ID},
		{"Name", f.Name},
		{"Type", f.Type},
		{"Board", f.IDModel},
	}); err != nil {
		return err
	}
	if len(f.Options) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(f.Options))
	for _, o := range f.Options {
		rows = append(rows, []string{o.ID, o.Value.Text})
	}
	return out.Table([]string{"ID", "OPTION"}, rows)
}
