// Package cli wires the Cobra command tree to the config, trello, and output
// packages. It owns flag parsing, exit-code plumbing, and the injection seams
// used by tests.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/nomadicworks/trello-cli/internal/config"
	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

// Options holds the global persistent flag values.
type Options struct {
	Board   string
	Key     string
	Token   string
	JSON    bool
	NoColor bool
	Timeout time.Duration
}

// Env carries the environment-dependent inputs a command may need. Every
// field is injectable so tests can control the working directory, the user
// config directory, and credentials without touching the real environment.
type Env struct {
	APIKey        string
	Token         string
	Getwd         func() (string, error)
	UserConfigDir func() (string, error)
	HomeDir       string
	IsTTY         func() bool
}

// defaultEnv reads the real process environment.
func defaultEnv() Env {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Env{
		APIKey:        os.Getenv("TRELLO_API_KEY"),
		Token:         os.Getenv("TRELLO_TOKEN"),
		Getwd:         os.Getwd,
		UserConfigDir: os.UserConfigDir,
		HomeDir:       home,
		IsTTY:         func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
	}
}

// App is the per-invocation command context. It holds no global mutable
// state; a fresh App is built for every Run.
type App struct {
	opts    *Options
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	env     Env
	baseURL string
	w       *output.Writer
	root    *cobra.Command
}

// Run executes the CLI with the given arguments and streams, returning the
// process exit code. main() calls os.Exit with this value.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	app := newApp(&Options{Timeout: 15 * time.Second}, stdin, stdout, stderr, defaultEnv(), trello.DefaultBaseURL)
	return runApp(app, args)
}

// runWithEnv executes the CLI with an injected environment and base URL
// (used by tests).
func runWithEnv(args []string, stdin io.Reader, stdout, stderr io.Writer, env Env, baseURL string) int {
	if env.IsTTY == nil {
		env.IsTTY = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	}
	app := newApp(&Options{Timeout: 15 * time.Second}, stdin, stdout, stderr, env, baseURL)
	return runApp(app, args)
}

func newApp(opts *Options, stdin io.Reader, stdout, stderr io.Writer, env Env, baseURL string) *App {
	return &App{
		opts:    opts,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		env:     env,
		baseURL: baseURL,
	}
}

func runApp(app *App, args []string) int {
	root := app.newRootCmd()
	app.root = root
	root.SetArgs(args)
	root.SetOut(app.stdout)
	root.SetErr(app.stderr)
	root.SetIn(app.stdin)

	err := root.Execute()
	if err == nil {
		return output.ExitOK
	}
	if errors.Is(err, flag.ErrHelp) || errors.Is(err, pflag.ErrHelp) {
		return output.ExitOK
	}

	code := output.CodeFor(err)
	if code == output.ExitUsage || strings.HasPrefix(err.Error(), "unknown command") {
		fmt.Fprintf(app.stderr, "trello: %s\n", err.Error())
		fmt.Fprintf(app.stderr, "Run 'trello --help' for usage.\n")
		return output.ExitUsage
	}
	app.out().PrintError(err)
	return code
}

// exactArgs wraps cobra.ExactArgs so that an argument-count violation maps to
// the usage exit code (2) instead of the generic runtime exit code (1),
// matching SPEC §3.5.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return output.WithCode(err, output.ExitUsage)
		}
		return nil
	}
}

func (a *App) newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "trello",
		Short: "A command-line client for the Trello REST API",
		Long: `trello is a command-line client for the Trello REST API.

It reads your Trello API key and token from flags, environment variables, or
configuration files, and operates against boards selected by exact name, id,
or shortLink.

Global flags:
  --board, -b   board to operate on (exact name, id, or shortLink)
  --key         Trello API key (overrides TRELLO_API_KEY and config files)
  --token       Trello API token (overrides TRELLO_TOKEN and config files)
  --json        emit machine-readable JSON
  --no-color    disable colored output
  --timeout     HTTP request timeout (default 15s)

Exit codes:
  0  success
  1  generic/runtime error
  2  usage/flag error
  3  config error (missing/invalid board or config)
  4  authentication failure (HTTP 401)
  5  not found (HTTP 404)
  6  validation/conflict (HTTP 400/422)
  7  rate limited (HTTP 429)
  8  network/connection error`,
		Example: `  trello whoami
  trello board list --json
  trello list list
  trello card list --list "To do"
  trello comment list AbCdEf01
  trello checklist list AbCdEf01
  trello label list
  trello member list
  trello attachment list AbCdEf01
  trello customfield list
  trello action list
  trello notification list
  trello org list
  trello search "release"
  trello raw GET /members/me
  trello completion bash
  trello config init
  trello config show`,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return output.WithCode(err, output.ExitUsage)
	})

	pf := root.PersistentFlags()
	pf.StringVarP(&a.opts.Board, "board", "b", "", "board to operate on (exact name, id, or shortLink)")
	pf.StringVar(&a.opts.Key, "key", "", "Trello API key (overrides TRELLO_API_KEY and config files)")
	pf.StringVar(&a.opts.Token, "token", "", "Trello API token (overrides TRELLO_TOKEN and config files)")
	pf.BoolVar(&a.opts.JSON, "json", false, "emit machine-readable JSON")
	pf.BoolVar(&a.opts.NoColor, "no-color", false, "disable colored output")
	pf.DurationVar(&a.opts.Timeout, "timeout", 15*time.Second, "HTTP request timeout")

	root.AddCommand(
		a.newWhoamiCmd(),
		a.newBoardCmd(),
		a.newListCmd(),
		a.newCardCmd(),
		a.newCommentCmd(),
		a.newChecklistCmd(),
		a.newLabelCmd(),
		a.newMemberCmd(),
		a.newAttachmentCmd(),
		a.newCustomFieldCmd(),
		a.newActionCmd(),
		a.newNotificationCmd(),
		a.newOrgCmd(),
		a.newSearchCmd(),
		a.newRawCmd(),
		a.newConfigCmd(),
		a.newCompletionCmd(),
	)
	return root
}

// out lazily builds the output.Writer once flags have been parsed.
func (a *App) out() *output.Writer {
	if a.w == nil {
		a.w = output.New(a.stdout, a.stderr, a.opts.JSON, output.ColorEnabled(a.opts.NoColor, a.stdout))
	}
	return a.w
}

// warnf routes config warnings to stderr.
func (a *App) warnf(msg string) {
	a.out().Warnf("%s", msg)
}

// resolver builds the config resolver from the injected environment.
func (a *App) resolver() (*config.Resolver, error) {
	cwd, err := a.env.Getwd()
	if err != nil {
		return nil, output.WithCode(fmt.Errorf("determining working directory: %w", err), output.ExitConfig)
	}
	ucd, err := a.env.UserConfigDir()
	if err != nil {
		return nil, output.WithCode(fmt.Errorf("determining user config directory: %w", err), output.ExitConfig)
	}
	return config.NewResolver(cwd, a.env.HomeDir, filepath.Join(ucd, "trello"), a.warnf), nil
}

// resolve returns the fully resolved configuration for this invocation.
func (a *App) resolve() (config.Resolved, error) {
	resolver, err := a.resolver()
	if err != nil {
		return config.Resolved{}, err
	}
	return resolver.Resolve(
		config.Credentials{Board: a.opts.Board, Key: a.opts.Key, Token: a.opts.Token},
		config.Env{APIKey: a.env.APIKey, Token: a.env.Token},
	)
}

// client builds a trello.Client from resolved credentials, failing with a
// clear config error when credentials are missing.
func (a *App) client() (*trello.Client, error) {
	resolved, err := a.resolve()
	if err != nil {
		return nil, err
	}
	if !resolved.Key.Present || !resolved.Token.Present {
		return nil, output.WithCode(
			errors.New(missingCredsMessage(resolved.Key.Present, resolved.Token.Present)),
			output.ExitConfig,
		)
	}
	return a.clientFromResolved(resolved), nil
}

// missingCredsMessage describes exactly which credentials are missing.
func missingCredsMessage(keyPresent, tokenPresent bool) string {
	switch {
	case !keyPresent && !tokenPresent:
		return "trello API key and token not found (credentials missing); set --key and --token, TRELLO_API_KEY and TRELLO_TOKEN, or add key and token to your config file"
	case !keyPresent:
		return "trello API key not found (credentials incomplete); set --key or TRELLO_API_KEY, or add a key to your config file"
	default:
		return "trello API token not found (credentials incomplete); set --token or TRELLO_TOKEN, or add a token to your config file"
	}
}

// clientFromResolved builds a trello.Client from an already-resolved config.
func (a *App) clientFromResolved(resolved config.Resolved) *trello.Client {
	return trello.New(resolved.Key.Value, resolved.Token.Value, a.baseURL, &http.Client{Timeout: a.opts.Timeout})
}

// credsFromFlagsEnv returns the key/token that originated from flags or the
// environment (i.e. values worth persisting via `config init`).
func (a *App) credsFromFlagsEnv() (key, token string, err error) {
	resolved, err := a.resolve()
	if err != nil {
		return "", "", err
	}
	if resolved.Key.Source == "flag" || resolved.Key.Source == "environment" {
		key = resolved.Key.Value
	}
	if resolved.Token.Source == "flag" || resolved.Token.Source == "environment" {
		token = resolved.Token.Value
	}
	return key, token, nil
}

// board resolves the board reference (--board or the configured default)
// against the member's boards. A missing board or a reference that matches
// nothing is a config error (exit 3), never a silent fallback.
func (a *App) board(ctx context.Context) (*trello.Board, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	resolved, err := a.resolve()
	if err != nil {
		return nil, err
	}
	if !resolved.Board.Present {
		return nil, output.WithCode(
			errors.New("no board selected; pass --board <name|id|shortLink> or run 'trello config init' to set a default board"),
			output.ExitConfig,
		)
	}
	return client.ResolveBoard(ctx, resolved.Board.Value)
}
