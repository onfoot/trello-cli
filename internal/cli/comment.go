package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newCommentCmd() *cobra.Command {
	comment := &cobra.Command{
		Use:   "comment",
		Short: "Manage comments on cards",
		Long:  `Add and list comments on cards.`,
		Example: `  trello comment add 5abbe4b7ddc1b351ef961417 --text "Looks good"
  trello comment list 5abbe4b7ddc1b351ef961417`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	comment.AddCommand(
		a.newCommentAddCmd(),
		a.newCommentListCmd(),
	)
	return comment
}

func (a *App) newCommentAddCmd() *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "add <card>",
		Short: "Add a comment to a card",
		Long: `Add a comment to a card.

<card> is the card's id or shortLink.`,
		Example: `  trello comment add 5abbe4b7ddc1b351ef961417 --text "Looks good"
  trello comment add AbCdEf01 --text "Ship it" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if text == "" {
				return output.WithCode(errors.New("--text is required"), output.ExitUsage)
			}
			return a.runCommentAdd(cmd.Context(), args[0], text)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "comment text")
	return cmd
}

func (a *App) runCommentAdd(ctx context.Context, cardRef, text string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	comment, err := client.AddComment(ctx, cardRef, text)
	if err != nil {
		return err
	}
	return a.renderComment(comment)
}

func (a *App) newCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <card>",
		Aliases: []string{"ls"},
		Short:   "List comments on a card",
		Long: `List the comments on a card.

<card> is the card's id or shortLink.`,
		Example: `  trello comment list 5abbe4b7ddc1b351ef961417
  trello comment list AbCdEf01 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCommentList(cmd.Context(), args[0])
		},
	}
}

func (a *App) runCommentList(ctx context.Context, cardRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	comments, err := client.ListComments(ctx, cardRef)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(comments)
	}
	rows := make([][]string, 0, len(comments))
	for _, c := range comments {
		rows = append(rows, []string{c.ID, formatTimePtr(c.Date), c.Data.Text})
	}
	return out.Table([]string{"ID", "DATE", "TEXT"}, rows)
}

// renderComment prints a comment in human or JSON form.
func (a *App) renderComment(c *trello.Action) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(c)
	}
	return out.KeyValue([][2]string{
		{"ID", c.ID},
		{"Date", formatTimePtr(c.Date)},
		{"Text", c.Data.Text},
	})
}
