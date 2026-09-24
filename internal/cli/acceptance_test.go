package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/nomadicworks/trello-cli/internal/output"
	"github.com/nomadicworks/trello-cli/internal/trello"
)

// The tests in this file codify the SPEC §9 acceptance criteria so they are
// enforced by the test suite, not just manually checked.

// Criterion 1: `trello board list --json` returns valid JSON and exit 0.
func TestAcceptanceBoardListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/members/me/boards": testBoardsFixture})
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"board", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	var boards []trello.Board
	if err := json.Unmarshal([]byte(stdout), &boards); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout)
	}
	if len(boards) != 1 || boards[0].ID == "" {
		t.Errorf("unexpected boards: %+v", boards)
	}
}

// Criterion 2: commands honor --board and the config-file default identically.
func TestAcceptanceBoardFlagAndConfigEquivalent(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/lists": testListsFixture,
	})
	defer srv.Close()

	envFlag := credsEnv(t)
	code1, out1, _ := runCLI(t, []string{"list", "list", "--board", "5abbe4b7ddc1b351ef961414"}, strings.NewReader(""), envFlag, srv.URL)
	if code1 != output.ExitOK {
		t.Fatalf("--board path: exit = %d, want 0", code1)
	}

	envCfg := boardEnv(t)
	code2, out2, _ := runCLI(t, []string{"list", "list"}, strings.NewReader(""), envCfg, srv.URL)
	if code2 != output.ExitOK {
		t.Fatalf("config path: exit = %d, want 0", code2)
	}
	if out1 != out2 {
		t.Errorf("--board and config default produce different output:\n--board:\n%s\nconfig:\n%s", out1, out2)
	}
}

// Criterion 3: missing credentials → clear error, correct exit code, no panic.
func TestAcceptanceMissingCreds(t *testing.T) {
	commands := [][]string{
		{"whoami"},
		{"board", "list"},
		{"list", "list"},
		{"card", "list"},
		{"label", "list"},
		{"member", "list"},
		{"attachment", "list", "c1"},
		{"customfield", "list"},
		{"action", "list"},
		{"notification", "list"},
		{"org", "list"},
		{"search", "x"},
		{"raw", "GET", "/members/me"},
	}
	for _, args := range commands {
		code, _, stderr := runCLI(t, args, strings.NewReader(""), testEnv(t, ""), "")
		if code != output.ExitConfig {
			t.Errorf("trello %s: exit = %d, want %d", strings.Join(args, " "), code, output.ExitConfig)
		}
		if !strings.Contains(stderr, "credentials") {
			t.Errorf("trello %s: stderr should mention credentials: %q", strings.Join(args, " "), stderr)
		}
	}
}

// Criterion 4: --help on every command prints real usage + examples.
func TestAcceptanceHelpEveryCommand(t *testing.T) {
	app := newApp(&Options{Timeout: 15 * time.Second}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, defaultEnv(), trello.DefaultBaseURL)
	root := app.newRootCmd()

	var walk func(cmd *cobra.Command, path []string)
	walk = func(cmd *cobra.Command, path []string) {
		for _, sub := range cmd.Commands() {
			if sub.Name() == "help" {
				continue // cobra-provided help command
			}
			subPath := append(append([]string{}, path...), sub.Name())
			walk(sub, subPath)
		}
		args := append(append([]string{}, path...), "--help")
		code, stdout, _ := runCLI(t, args, strings.NewReader(""), testEnv(t, ""), "")
		if code != output.ExitOK {
			t.Errorf("trello %s --help: exit = %d, want 0", strings.Join(path, " "), code)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("trello %s --help missing Usage section", strings.Join(path, " "))
		}
		if !strings.Contains(stdout, "Examples:") {
			t.Errorf("trello %s --help missing Examples section", strings.Join(path, " "))
		}
	}
	walk(root, []string{})
}

// Criterion 4 (static check): every command has real Long and Example text.
func TestHelpAudit(t *testing.T) {
	app := newApp(&Options{Timeout: 15 * time.Second}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, defaultEnv(), trello.DefaultBaseURL)
	root := app.newRootCmd()

	if strings.TrimSpace(root.Long) == "" {
		t.Error("root command has empty Long")
	}
	if strings.TrimSpace(root.Example) == "" {
		t.Error("root command has empty Example")
	}

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, sub := range cmd.Commands() {
			if sub.Name() == "help" {
				continue // cobra-provided help command
			}
			if strings.TrimSpace(sub.Long) == "" {
				t.Errorf("command %q has empty Long", sub.CommandPath())
			}
			if strings.TrimSpace(sub.Example) == "" {
				t.Errorf("command %q has empty Example", sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(root)
}

// Criterion 5: network failure → distinct connection error, exit 8.
func TestAcceptanceNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	}))
	baseURL := srv.URL
	srv.Close() // force connection errors

	env := credsEnv(t)
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, baseURL)
	if code != output.ExitNetwork {
		t.Errorf("exit = %d, want %d", code, output.ExitNetwork)
	}
	if !strings.Contains(stderr, "connection error") {
		t.Errorf("stderr should explain the network failure: %q", stderr)
	}
}

// Criterion 6: HTTP 401/404/429 → exit codes 4/5/7 with the body surfaced.
func TestAcceptanceStatusExitCodes(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantMsg string
		want    int
	}{
		{"401", http.StatusUnauthorized, `{"code":"unauthorized","message":"invalid key"}`, "invalid key", output.ExitAuth},
		{"404", http.StatusNotFound, `{"code":"notfound","message":"no such board"}`, "no such board", output.ExitNotFound},
		{"429", http.StatusTooManyRequests, `{"code":"ratelimit","message":"slow down"}`, "slow down", output.ExitRateLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			env := credsEnv(t)
			code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, srv.URL)
			if code != tt.want {
				t.Errorf("exit = %d, want %d", code, tt.want)
			}
			if !strings.Contains(stderr, strconv.Itoa(tt.status)) {
				t.Errorf("stderr should surface the status code: %q", stderr)
			}
			if !strings.Contains(stderr, tt.wantMsg) {
				t.Errorf("stderr should surface the response message %q: %q", tt.wantMsg, stderr)
			}
		})
	}
}

// Criterion 8: piped output contains no ANSI color codes.
func TestAcceptanceNoANSIWhenPiped(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/members/me/boards": testBoardsFixture})
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"board", "list"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Errorf("piped output contains ANSI codes: %q", stdout)
	}
}
