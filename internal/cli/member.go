package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func (a *App) newMemberCmd() *cobra.Command {
	member := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
		Long:  `List and inspect board members.`,
		Example: `  trello member list
  trello member get bentleycook`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	member.AddCommand(
		a.newMemberListCmd(),
		a.newMemberGetCmd(),
	)
	return member
}

func (a *App) newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the members of the resolved board",
		Long: `List the members of the resolved board (--board or the configured
default). The board is resolved by exact name, id, or shortLink.`,
		Example: `  trello member list
  trello member list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runMemberList(cmd.Context())
		},
	}
}

func (a *App) runMemberList(ctx context.Context) error {
	board, err := a.board(ctx)
	if err != nil {
		return err
	}
	client, err := a.client()
	if err != nil {
		return err
	}
	members, err := client.ListBoardMembers(ctx, board.ID)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(members)
	}
	rows := make([][]string, 0, len(members))
	for _, m := range members {
		rows = append(rows, []string{m.ID, m.Username, m.FullName})
	}
	return out.Table([]string{"ID", "USERNAME", "FULL NAME"}, rows)
}

func (a *App) newMemberGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-username>",
		Short: "Get a member by id or username",
		Long: `Get a member by id, username, or the special value "me" for the
authenticated member.`,
		Example: `  trello member get bentleycook
  trello member get 5b02e7f4e1facdc393169f9d
  trello member get me --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runMemberGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runMemberGet(ctx context.Context, ref string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	m, err := client.GetMember(ctx, ref)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(m)
	}
	return out.KeyValue([][2]string{
		{"Username", m.Username},
		{"Full name", m.FullName},
		{"Initials", m.Initials},
		{"ID", m.ID},
		{"URL", m.URL},
		{"Email", m.Email},
		{"Bio", m.Bio},
	})
}
