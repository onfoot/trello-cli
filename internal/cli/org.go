package cli

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newOrgCmd() *cobra.Command {
	org := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
		Long:  `List and inspect organizations (teams).`,
		Example: `  trello org list
  trello org get acme
  trello org boards acme
  trello org members acme`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	org.AddCommand(
		a.newOrgListCmd(),
		a.newOrgGetCmd(),
		a.newOrgBoardsCmd(),
		a.newOrgMembersCmd(),
	)
	return org
}

func (a *App) newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the organizations you belong to",
		Long:    `List the organizations the authenticated member belongs to.`,
		Example: `  trello org list
  trello org list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOrgList(cmd.Context())
		},
	}
}

func (a *App) runOrgList(ctx context.Context) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	orgs, err := client.ListMyOrganizations(ctx)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(orgs)
	}
	rows := make([][]string, 0, len(orgs))
	for _, o := range orgs {
		rows = append(rows, []string{o.ID, o.Name, o.DisplayName})
	}
	return out.Table([]string{"ID", "NAME", "DISPLAY NAME"}, rows)
}

func (a *App) newOrgGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <ref>",
		Short: "Get an organization",
		Long: `Get an organization by id, exact short name, or exact display name.

The reference is resolved against your organizations; a 24-hex id is used
directly.`,
		Example: `  trello org get acme
  trello org get 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOrgGet(cmd.Context(), args[0])
		},
	}
}

func (a *App) runOrgGet(ctx context.Context, ref string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	org, err := client.ResolveOrg(ctx, ref)
	if err != nil {
		return err
	}
	return a.renderOrg(org)
}

func (a *App) newOrgBoardsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "boards <ref>",
		Short: "List an organization's boards",
		Long: `List the boards of an organization, resolved by id, exact short name,
or exact display name.`,
		Example: `  trello org boards acme
  trello org boards 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOrgBoards(cmd.Context(), args[0])
		},
	}
}

func (a *App) runOrgBoards(ctx context.Context, ref string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	org, err := client.ResolveOrg(ctx, ref)
	if err != nil {
		return err
	}
	boards, err := client.ListOrgBoards(ctx, org.ID)
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(boards)
	}
	rows := make([][]string, 0, len(boards))
	for _, b := range boards {
		rows = append(rows, []string{b.ID, b.Name, b.ShortLink, strconv.FormatBool(b.Closed)})
	}
	return out.Table([]string{"ID", "NAME", "SHORT LINK", "CLOSED"}, rows)
}

func (a *App) newOrgMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members <ref>",
		Short: "List an organization's members",
		Long: `List the members of an organization, resolved by id, exact short name,
or exact display name.`,
		Example: `  trello org members acme
  trello org members 5abbe4b7ddc1b351ef961414 --json`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOrgMembers(cmd.Context(), args[0])
		},
	}
}

func (a *App) runOrgMembers(ctx context.Context, ref string) error {
	client, err := a.client()
	if err != nil {
		return err
	}
	org, err := client.ResolveOrg(ctx, ref)
	if err != nil {
		return err
	}
	members, err := client.ListOrgMembers(ctx, org.ID)
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

// renderOrg prints an organization in human or JSON form.
func (a *App) renderOrg(o *trello.Organization) error {
	out := a.out()
	if out.JSON {
		return out.JSONOut(o)
	}
	return out.KeyValue([][2]string{
		{"ID", o.ID},
		{"Name", o.Name},
		{"Display name", o.DisplayName},
		{"Desc", o.Desc},
		{"URL", o.URL},
		{"Website", o.Website},
	})
}
