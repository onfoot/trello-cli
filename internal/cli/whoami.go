package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func (a *App) newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated Trello member",
		Long: `Validate your Trello credentials and print the member they belong to.

Requires a Trello API key and token, supplied via --key/--token,
TRELLO_API_KEY/TRELLO_TOKEN, or a config file.`,
		Example: `  trello whoami
  trello whoami --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runWhoami(cmd.Context())
		},
	}
}

func (a *App) runWhoami(ctx context.Context) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	member, err := client.GetMemberMe(ctx)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(member)
	}
	return out.KeyValue([][2]string{
		{"Username", member.Username},
		{"Full name", member.FullName},
		{"Initials", member.Initials},
		{"ID", member.ID},
		{"URL", member.URL},
		{"Email", member.Email},
		{"Bio", member.Bio},
	})
}
