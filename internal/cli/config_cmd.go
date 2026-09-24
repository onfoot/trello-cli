package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/config"
	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

func (a *App) newConfigCmd() *cobra.Command {
	cfg := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage trello configuration.

Configuration is read from project-local .trello.yaml files (walking up from
the current directory to $HOME) and from the user config file at
<UserConfigDir>/trello/config.yaml.`,
		Example: `  trello config init
  trello config show
  trello config path`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cfg.AddCommand(
		a.newConfigInitCmd(),
		a.newConfigShowCmd(),
		a.newConfigPathCmd(),
	)
	return cfg
}

func (a *App) newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the resolved configuration",
		Long: `Print the resolved board and credential status.

Credentials are reported as present/missing only — the token is never
printed. Each value is annotated with the file (or flag/environment) it
came from.`,
		Example: `  trello config show
  trello config show --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runConfigShow()
		},
	}
}

func (a *App) runConfigShow() error {
	resolved, err := a.resolve()
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(resolved.ShowView())
	}
	return out.KeyValue([][2]string{
		{"board", describeBoard(resolved.Board)},
		{"key", describePresence(resolved.Key)},
		{"token", describePresence(resolved.Token)},
	})
}

func describeBoard(v config.Value) string {
	if !v.Present {
		return "missing"
	}
	return fmt.Sprintf("%s (from: %s)", v.Value, v.Source)
}

func describePresence(v config.Value) string {
	if !v.Present {
		return "missing"
	}
	return fmt.Sprintf("present (from: %s)", v.Source)
}

func (a *App) newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the configuration file paths",
		Long: `Print the project-local and global configuration file paths.

The project path is the nearest .trello.yaml found by walking up from the
current directory (or where one would be created).`,
		Example: `  trello config path
  trello config path --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runConfigPath()
		},
	}
}

type pathInfo struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func (a *App) runConfigPath() error {
	resolver, err := a.resolver()
	if err != nil {
		return err
	}
	project, global := resolver.Paths()
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Project pathInfo `json:"project"`
			Global  pathInfo `json:"global"`
		}{
			Project: pathInfo{Path: project, Exists: fileExists(project)},
			Global:  pathInfo{Path: global, Exists: fileExists(global)},
		})
	}
	return out.KeyValue([][2]string{
		{"project", fmt.Sprintf("%s (%s)", project, existsLabel(project))},
		{"global", fmt.Sprintf("%s (%s)", global, existsLabel(global))},
	})
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func existsLabel(p string) string {
	if fileExists(p) {
		return "exists"
	}
	return "missing"
}

func (a *App) newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a project-local .trello.yaml",
		Long: `Create or update the project-local .trello.yaml with a board.

Boards are fetched from your account and you pick one interactively. Pass
--board <name|id|shortLink> to skip the prompt; the reference is validated
against your boards when credentials are available and stored as the
resolved board id. Without credentials the reference is stored as-is (with
a warning). If --key/--token or TRELLO_API_KEY/TRELLO_TOKEN are set, they
are written to the file too.`,
		Example: `  trello config init
  trello config init --board my-board
  trello config init --board 5abbe4b7ddc1b351ef961414`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runConfigInit(cmd.Context())
		},
	}
}

func (a *App) runConfigInit(ctx context.Context) error {
	board := a.opts.Board
	if board == "" {
		if !a.env.IsTTY() {
			return output.WithCode(errors.New("stdin is not a terminal; pass --board <name|id|shortLink> to choose a board non-interactively"), output.ExitConfig)
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		boards, err := client.ListMemberBoards(ctx)
		if err != nil {
			return err
		}
		if len(boards) == 0 {
			return output.WithCode(errors.New("no boards found for this account; create a board first"), output.ExitConfig)
		}
		board, err = a.promptBoard(boards)
		if err != nil {
			return err
		}
	} else {
		// Non-interactive path: validate the reference when credentials are
		// available. Without credentials the reference is stored as-is.
		resolved, err := a.resolve()
		if err != nil {
			return err
		}
		if resolved.Key.Present && resolved.Token.Present {
			rb, err := a.clientFromResolved(resolved).ResolveBoard(ctx, board)
			if err != nil {
				return err
			}
			board = rb.ID
		} else {
			a.out().Warnf("%s; persisting board %q unvalidated", missingCredsMessage(resolved.Key.Present, resolved.Token.Present), board)
		}
	}

	key, token, err := a.credsFromFlagsEnv()
	if err != nil {
		return err
	}
	resolver, err := a.resolver()
	if err != nil {
		return err
	}
	path, err := resolver.Init(config.InitInput{Board: board, Key: key, Token: token})
	if err != nil {
		return err
	}
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Path    string `json:"path"`
			Board   string `json:"board"`
			Written bool   `json:"written"`
		}{
			Path:    path,
			Board:   board,
			Written: true,
		})
	}
	out.Printf("Wrote %s\n", path)
	return nil
}

func (a *App) promptBoard(boards []trello.Board) (string, error) {
	out := a.out()
	// In JSON mode the prompt goes to stderr so stdout stays pure JSON.
	dest := out.Out
	if out.JSON {
		dest = out.Err
	}
	fmt.Fprintf(dest, "Boards:\n")
	for i, b := range boards {
		fmt.Fprintf(dest, "%3d  %-9s  %s\n", i+1, b.ShortLink, b.Name)
	}
	fmt.Fprintf(dest, "Select a board (1-%d): ", len(boards))
	line, err := bufio.NewReader(a.stdin).ReadString('\n')
	if err != nil {
		return "", output.WithCode(fmt.Errorf("reading selection: %w", err), output.ExitConfig)
	}
	sel := strings.TrimSpace(line)
	n, err := strconv.Atoi(sel)
	if err != nil || n < 1 || n > len(boards) {
		return "", output.WithCode(fmt.Errorf("invalid selection %q", sel), output.ExitConfig)
	}
	return boards[n-1].ID, nil
}
