package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// completionShells lists the supported completion shells.
var completionShells = []string{"bash", "zsh", "fish"}

// completionInstall describes where a shell's completion script is installed.
type completionInstall struct {
	shell string
	// rel is the path relative to the base directory (or --dir).
	rel string
	// defaultBase returns the OS-specific base directory when --dir is unset.
	defaultBase func() (string, error)
}

// completionInstalls maps each shell to its install target.
var completionInstalls = map[string]completionInstall{
	"bash": {shell: "bash", rel: "bash-completion/completions/trello", defaultBase: xdgDataHome},
	"zsh":  {shell: "zsh", rel: "completions/_trello", defaultBase: zshHome},
	"fish": {shell: "fish", rel: "fish/completions/trello.fish", defaultBase: xdgConfigHome},
}

// completionHints are the short activation hints printed after an install.
var completionHints = map[string]string{
	"bash": "ensure bash-completion is sourced (e.g. add 'source /usr/share/bash-completion/bash_completion' to ~/.bashrc)",
	"zsh":  "add the directory to fpath and run compinit (e.g. 'fpath+=(<dir>)' then 'autoload -Uz compinit && compinit')",
	"fish": "fish loads completions from this directory automatically",
}

// xdgDataHome returns $XDG_DATA_HOME or ~/.local/share.
func xdgDataHome() (string, error) {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// xdgConfigHome returns $XDG_CONFIG_HOME or ~/.config.
func xdgConfigHome() (string, error) {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// zshHome returns ~/.zsh, the base of the standard zsh completion directory.
func zshHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zsh"), nil
}

// installPath returns the destination file for the completion script. dir,
// when non-empty, overrides the shell's default base directory.
func (t completionInstall) installPath(dir string) (string, error) {
	if dir != "" {
		// --dir replaces the base directory. The zsh base is ~/.zsh, so the
		// override keeps the "zsh" segment: <dir>/zsh/completions/_trello.
		if t.shell == "zsh" {
			return filepath.Join(dir, "zsh", "completions", "_trello"), nil
		}
		return filepath.Join(dir, t.rel), nil
	}
	base, err := t.defaultBase()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, t.rel), nil
}

// rootCmd returns the executing root command for completion generation.
// Building a fresh root would re-bind the persistent flags and reset the
// parsed option values, so the executing root is reused when available.
func (a *App) rootCmd() *cobra.Command {
	if a.root != nil {
		return a.root
	}
	return a.newRootCmd()
}

func (a *App) newCompletionCmd() *cobra.Command {
	completion := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bash, zsh, and fish.

To enable completions for the current shell, source the generated script:

  source <(trello completion bash)

To install a script permanently, use 'trello completion install <shell>'.`,
		Example: `  source <(trello completion bash)
  trello completion install zsh`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	completion.AddCommand(
		a.newCompletionBashCmd(),
		a.newCompletionZshCmd(),
		a.newCompletionFishCmd(),
		a.newCompletionInstallCmd(),
	)
	return completion
}

func (a *App) newCompletionBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate a bash completion script",
		Long: `Generate a bash completion script and print it to stdout.

Source it for the current session:

  source <(trello completion bash)

or install it permanently with 'trello completion install bash'.`,
		Example: `  source <(trello completion bash)
  trello completion install bash`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := a.rootCmd()
			return root.GenBashCompletion(a.stdout)
		},
	}
}

func (a *App) newCompletionZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate a zsh completion script",
		Long: `Generate a zsh completion script and print it to stdout.

Source it for the current session:

  source <(trello completion zsh)

or install it permanently with 'trello completion install zsh'.`,
		Example: `  source <(trello completion zsh)
  trello completion install zsh`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := a.rootCmd()
			return root.GenZshCompletion(a.stdout)
		},
	}
}

func (a *App) newCompletionFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate a fish completion script",
		Long: `Generate a fish completion script and print it to stdout.

Source it for the current session:

  source (trello completion fish | psub)

or install it permanently with 'trello completion install fish'.`,
		Example: `  trello completion fish
  trello completion install fish`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := a.rootCmd()
			return root.GenFishCompletion(a.stdout, true)
		},
	}
}

func (a *App) newCompletionInstallCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "install <shell>",
		Short: "Install a completion script",
		Long: `Generate a completion script and write it to the shell's standard
completion directory.

<shell> is one of bash, zsh, or fish. Use --dir to override the base
directory (defaults to the OS-specific location).`,
		Example: `  trello completion install bash
  trello completion install zsh --dir ~/tmp/completions`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runCompletionInstall(cmd.Context(), args[0], dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "override the base directory for the completion file")
	return cmd
}

// completionScript generates the completion script for shell.
func (a *App) completionScript(shell string) (string, error) {
	root := a.rootCmd()
	var buf bytes.Buffer
	switch shell {
	case "bash":
		if err := root.GenBashCompletion(&buf); err != nil {
			return "", err
		}
	case "zsh":
		if err := root.GenZshCompletion(&buf); err != nil {
			return "", err
		}
	case "fish":
		if err := root.GenFishCompletion(&buf, true); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported completion shell %q", shell)
	}
	return buf.String(), nil
}

func (a *App) runCompletionInstall(ctx context.Context, shell, dir string) error {
	target, ok := completionInstalls[shell]
	if !ok {
		return output.WithCode(fmt.Errorf("invalid shell %q: must be one of %s", shell, strings.Join(completionShells, ", ")), output.ExitUsage)
	}
	path, err := target.installPath(dir)
	if err != nil {
		return err
	}
	script, err := a.completionScript(shell)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating completion directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		return fmt.Errorf("writing completion file %s: %w", path, err)
	}
	hint := completionHints[shell]
	out := a.out()
	if out.JSON {
		return out.JSONOut(struct {
			Shell string `json:"shell"`
			Path  string `json:"path"`
			Hint  string `json:"hint"`
		}{Shell: shell, Path: path, Hint: hint})
	}
	out.Printf("Installed %s completion to %s\n", shell, path)
	out.Printf("Hint: %s\n", hint)
	return nil
}
