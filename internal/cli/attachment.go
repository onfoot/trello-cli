package cli

import (
	"context"
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newAttachmentCmd() *cobra.Command {
	att := &cobra.Command{
		Use:   "attachment",
		Short: "Manage attachments",
		Long:  `Manage attachments on cards.`,
		Example: `  trello attachment list 5abbe4b7ddc1b351ef961417
  trello attachment add 5abbe4b7ddc1b351ef961417 --url https://example.com/doc.pdf`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	att.AddCommand(
		a.newAttachmentListCmd(),
		a.newAttachmentGetCmd(),
		a.newAttachmentAddCmd(),
		a.newAttachmentDeleteCmd(),
	)
	return att
}

func (a *App) newAttachmentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <card>",
		Aliases: []string{"ls"},
		Short:   "List the attachments on a card",
		Long: `List the attachments on a card.

<card> is the card's id or shortLink.`,
		Example: `  trello attachment list 5abbe4b7ddc1b351ef961417
  trello attachment list AbCdEf01 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runAttachmentList(cmd.Context(), args[0])
		},
	}
}

func (a *App) runAttachmentList(ctx context.Context, cardRef string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	atts, err := client.ListCardAttachments(ctx, cardRef)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(atts)
	}
	rows := make([][]string, 0, len(atts))
	for _, att := range atts {
		rows = append(rows, []string{att.ID, att.Name, att.MimeType, formatBytes(att.Bytes), att.URL})
	}
	return out.Table([]string{"ID", "NAME", "MIME TYPE", "BYTES", "URL"}, rows)
}

func (a *App) newAttachmentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <card> <attachment-id>",
		Short: "Get an attachment by id",
		Long: `Get a single attachment by id.

<card> is the card's id or shortLink.`,
		Example: `  trello attachment get 5abbe4b7ddc1b351ef961417 5bc79d4206526d2279c1e6ea
  trello attachment get AbCdEf01 5bc79d4206526d2279c1e6ea --json`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runAttachmentGet(cmd.Context(), args[0], args[1])
		},
	}
}

func (a *App) runAttachmentGet(ctx context.Context, cardRef, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	att, err := client.GetAttachment(ctx, cardRef, id)
	if err != nil {
		return err
	}
	return a.renderAttachment(att)
}

func (a *App) newAttachmentAddCmd() *cobra.Command {
	var attachURL, filePath, name string
	cmd := &cobra.Command{
		Use:   "add <card>",
		Short: "Add an attachment to a card",
		Long: `Add an attachment to a card, either by URL or by uploading a local
file.

Exactly one of --url or --file is required. <card> is the card's id or
shortLink.`,
		Example: `  trello attachment add 5abbe4b7ddc1b351ef961417 --url https://example.com/doc.pdf
  trello attachment add AbCdEf01 --file ./notes.txt --name "Notes" --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (attachURL == "") == (filePath == "") {
				return output.WithCode(errors.New("exactly one of --url or --file is required"), output.ExitUsage)
			}
			return a.runAttachmentAdd(cmd.Context(), args[0], attachURL, filePath, name)
		},
	}
	cmd.Flags().StringVar(&attachURL, "url", "", "URL of the attachment")
	cmd.Flags().StringVar(&filePath, "file", "", "path to a local file to upload")
	cmd.Flags().StringVar(&name, "name", "", "attachment name (optional)")
	return cmd
}

func (a *App) runAttachmentAdd(ctx context.Context, cardRef, attachURL, filePath, name string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	var att *trello.Attachment
	if filePath != "" {
		att, err = client.UploadAttachment(ctx, cardRef, filePath, name)
	} else {
		att, err = client.AddAttachmentByURL(ctx, cardRef, attachURL, name)
	}
	if err != nil {
		return err
	}
	return a.renderAttachment(att)
}

func (a *App) newAttachmentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <card> <attachment-id>",
		Short: "Delete an attachment",
		Long: `Permanently delete an attachment from a card.

<card> is the card's id or shortLink.`,
		Example: `  trello attachment delete 5abbe4b7ddc1b351ef961417 5bc79d4206526d2279c1e6ea
  trello attachment delete AbCdEf01 5bc79d4206526d2279c1e6ea --json`,
		Args: exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runAttachmentDelete(cmd.Context(), args[0], args[1])
		},
	}
}

func (a *App) runAttachmentDelete(ctx context.Context, cardRef, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if err := client.DeleteAttachment(ctx, cardRef, id); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Card    string `json:"card"`
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{Card: cardRef, ID: id, Deleted: true})
	}
	out.Printf("Deleted attachment %s from card %s\n", id, cardRef)
	return nil
}

// renderAttachment prints an attachment in human or JSON form.
func (a *App) renderAttachment(att *trello.Attachment) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(att)
	}
	return out.KeyValue([][2]string{
		{"ID", att.ID},
		{"Name", att.Name},
		{"MimeType", att.MimeType},
		{"Bytes", formatBytes(att.Bytes)},
		{"URL", att.URL},
		{"Upload", strconv.FormatBool(att.IsUpload)},
	})
}
