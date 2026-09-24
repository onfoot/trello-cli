package cli

import (
	"context"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newNotificationCmd() *cobra.Command {
	notif := &cobra.Command{
		Use:   "notification",
		Short: "Manage notifications",
		Long:  `List and manage the authenticated member's notifications.`,
		Example: `  trello notification list
  trello notification read 5dc591ac425f2a223aba0a8e
  trello notification read-all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	notif.AddCommand(
		a.newNotificationListCmd(),
		a.newNotificationReadCmd(),
		a.newNotificationReadAllCmd(),
	)
	return notif
}

func (a *App) newNotificationListCmd() *cobra.Command {
	var filter string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List notifications for the authenticated member",
		Long: `List notifications for the authenticated member.

--filter is a comma-separated list of notification types (default "all").`,
		Example: `  trello notification list
  trello notification list --filter cardDueSoon --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runNotificationList(cmd.Context(), filter)
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "all", "comma-separated notification types (default all)")
	return cmd
}

func (a *App) runNotificationList(ctx context.Context, filter string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	notifs, err := client.ListMyNotifications(ctx, filter)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(notifs)
	}
	rows := make([][]string, 0, len(notifs))
	for _, n := range notifs {
		rows = append(rows, []string{n.ID, n.Type, formatTimePtr(n.Date), strconv.FormatBool(n.Unread), notificationSummary(&n)})
	}
	return out.Table([]string{"ID", "TYPE", "DATE", "UNREAD", "DATA"}, rows)
}

func (a *App) newNotificationReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <id>",
		Short: "Mark a notification as read",
		Long:  `Mark a notification as read by setting unread=false.`,
		Example: `  trello notification read 5dc591ac425f2a223aba0a8e
  trello notification read 5dc591ac425f2a223aba0a8e --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runNotificationRead(cmd.Context(), args[0])
		},
	}
}

func (a *App) runNotificationRead(ctx context.Context, id string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	n, err := client.SetNotificationRead(ctx, id, false)
	if err != nil {
		return err
	}
	return a.renderNotification(n)
}

func (a *App) newNotificationReadAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read-all",
		Short: "Mark all notifications as read",
		Long:  `Mark all of the authenticated member's notifications as read.`,
		Example: `  trello notification read-all
  trello notification read-all --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runNotificationReadAll(cmd.Context())
		},
	}
}

func (a *App) runNotificationReadAll(ctx context.Context) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	if err := client.MarkAllNotificationsRead(ctx); err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Read bool `json:"read"`
		}{Read: true})
	}
	out.Printf("Marked all notifications as read\n")
	return nil
}

// notificationSummary renders a readable summary of a notification's data.
func notificationSummary(n *trello.Notification) string {
	parts := make([]string, 0, 4)
	if n.Data.Text != "" {
		parts = append(parts, n.Data.Text)
	}
	if n.Data.Card != nil && n.Data.Card.Name != "" {
		parts = append(parts, "card "+n.Data.Card.Name)
	}
	if n.Data.List != nil && n.Data.List.Name != "" {
		parts = append(parts, "list "+n.Data.List.Name)
	}
	if n.Data.Board != nil && n.Data.Board.Name != "" {
		parts = append(parts, "board "+n.Data.Board.Name)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " on ")
}

// renderNotification prints a notification in human or JSON form.
func (a *App) renderNotification(n *trello.Notification) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(n)
	}
	return out.KeyValue([][2]string{
		{"ID", n.ID},
		{"Type", n.Type},
		{"Date", formatTimePtr(n.Date)},
		{"Unread", strconv.FormatBool(n.Unread)},
		{"Data", notificationSummary(n)},
	})
}
