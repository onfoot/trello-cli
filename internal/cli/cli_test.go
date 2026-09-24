package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// runCLI executes the CLI with the given args and captures the exit code and
// streams.
func runCLI(t *testing.T, args []string, stdin io.Reader, env Env, baseURL string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = runWithEnv(args, stdin, &out, &errb, env, baseURL)
	return code, out.String(), errb.String()
}

// testEnv builds an Env rooted in a fresh temp directory. cwd defaults to a
// "proj" subdirectory of the temp home.
func testEnv(t *testing.T, cwd string) Env {
	t.Helper()
	home := t.TempDir()
	if cwd == "" {
		cwd = filepath.Join(home, "proj")
	}
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	return Env{
		Getwd:         func() (string, error) { return cwd, nil },
		UserConfigDir: func() (string, error) { return filepath.Join(home, ".config"), nil },
		HomeDir:       home,
	}
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHelpExitZero(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"--help"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "trello") {
		t.Errorf("help output missing usage: %q", stdout)
	}
}

func TestRootNoArgsShowsHelp(t *testing.T) {
	code, stdout, _ := runCLI(t, nil, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("expected help, got %q", stdout)
	}
}

func TestUnknownFlagExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"--bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Errorf("stderr should mention the bad flag: %q", stderr)
	}
}

func TestUnknownCommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"frobnicate"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}

func TestInvalidTimeoutValueExitUsage(t *testing.T) {
	code, _, _ := runCLI(t, []string{"whoami", "--timeout", "not-a-duration"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestWhoamiMissingCreds(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "credentials") {
		t.Errorf("stderr should explain missing credentials: %q", stderr)
	}
}

func TestMissingCredsMessage(t *testing.T) {
	tests := []struct {
		name         string
		keyPresent   bool
		tokenPresent bool
		wantKey      bool // message should mention the key
		wantToken    bool // message should mention the token
	}{
		{"both missing", false, false, true, true},
		{"key missing", false, true, true, false},
		{"token missing", true, false, false, true},
		{"both present", true, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := missingCredsMessage(tt.keyPresent, tt.tokenPresent)
			if tt.keyPresent && tt.tokenPresent {
				// Both present is not an error path; the message is irrelevant.
				return
			}
			if strings.Contains(msg, "key") != tt.wantKey {
				t.Errorf("missingCredsMessage(%v,%v) = %q; mentions key: %v, want %v",
					tt.keyPresent, tt.tokenPresent, msg, strings.Contains(msg, "key"), tt.wantKey)
			}
			if strings.Contains(msg, "token") != tt.wantToken {
				t.Errorf("missingCredsMessage(%v,%v) = %q; mentions token: %v, want %v",
					tt.keyPresent, tt.tokenPresent, msg, strings.Contains(msg, "token"), tt.wantToken)
			}
			if !strings.Contains(msg, "credentials") {
				t.Errorf("missingCredsMessage(%v,%v) = %q; should contain the word credentials",
					tt.keyPresent, tt.tokenPresent, msg)
			}
		})
	}
}

func TestWhoamiMissingKey(t *testing.T) {
	// Only the token is set: the error must say the key is missing.
	env := testEnv(t, "")
	env.Token = "t"
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "key not found") {
		t.Errorf("stderr should say the key is missing: %q", stderr)
	}
	if strings.Contains(stderr, "token not found") {
		t.Errorf("stderr should not say the token is missing: %q", stderr)
	}
}

func TestWhoamiMissingToken(t *testing.T) {
	// Only the key is set: the error must say the token is missing.
	env := testEnv(t, "")
	env.APIKey = "k"
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "token not found") {
		t.Errorf("stderr should say the token is missing: %q", stderr)
	}
	if strings.Contains(stderr, "key not found") {
		t.Errorf("stderr should not say the key is missing: %q", stderr)
	}
}

func TestWhoamiAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":"unauthorized","message":"invalid key"}`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitAuth {
		t.Errorf("exit = %d, want %d", code, output.ExitAuth)
	}
	if !strings.Contains(stderr, "401") {
		t.Errorf("stderr should surface the status: %q", stderr)
	}
}

func TestWhoamiNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	}))
	baseURL := srv.URL
	srv.Close() // force connection errors

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, _, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, baseURL)
	if code != output.ExitNetwork {
		t.Errorf("exit = %d, want %d", code, output.ExitNetwork)
	}
	if !strings.Contains(stderr, "connection error") {
		t.Errorf("stderr should explain the network failure: %q", stderr)
	}
}

func TestWhoamiSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, _ := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "bentleycook") || !strings.Contains(stdout, "Bentley Cook") {
		t.Errorf("unexpected output: %q", stdout)
	}
}

func TestWhoamiJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, _ := runCLI(t, []string{"whoami", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, `"username": "bentleycook"`) {
		t.Errorf("expected JSON output, got: %q", stdout)
	}
}

func TestBoardListSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/boards" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		fmt.Fprint(w, `[
			{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
			{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
		]`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, _ := runCLI(t, []string{"board", "list"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Trello Platform Changes", "Release Planning", "3CsPkqOF", "5abbe4b7ddc1b351ef961414"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestBoardListNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such member"}`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, _, stderr := runCLI(t, []string{"board", "list"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitNotFound {
		t.Errorf("exit = %d, want %d", code, output.ExitNotFound)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should surface status: %q", stderr)
	}
}

func TestConfigShowHuman(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: my-board\nkey: k\n")
	code, stdout, _ := runCLI(t, []string{"config", "show"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "my-board") || !strings.Contains(stdout, "present") || !strings.Contains(stdout, "missing") {
		t.Errorf("unexpected output: %q", stdout)
	}
}

func TestConfigShowJSON(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: my-board\nkey: k\n")
	writeConfig(t, filepath.Join(env.HomeDir, ".config", "trello", "config.yaml"), "token: super-secret-token\n")
	code, stdout, _ := runCLI(t, []string{"config", "show", "--json"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, `"value": "my-board"`) {
		t.Errorf("expected board value in JSON: %q", stdout)
	}
	// Credential values must never appear in the output — only presence/source.
	if strings.Contains(stdout, `"value": "k"`) {
		t.Errorf("key value leaked into JSON: %q", stdout)
	}
	if strings.Contains(stdout, "super-secret-token") {
		t.Errorf("token value leaked into JSON: %q", stdout)
	}
}

func TestConfigShowNeverLeaksToken(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: b\nkey: k\n")
	writeConfig(t, filepath.Join(env.HomeDir, ".config", "trello", "config.yaml"), "token: super-secret-token\n")
	code, stdout, _ := runCLI(t, []string{"config", "show", "--json"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if strings.Contains(stdout, "super-secret-token") {
		t.Errorf("token value leaked into output: %q", stdout)
	}
}

func TestConfigPath(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: b\n")
	code, stdout, _ := runCLI(t, []string{"config", "path"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, filepath.Join(env.HomeDir, "proj", ".trello.yaml")) {
		t.Errorf("project path missing: %q", stdout)
	}
	if !strings.Contains(stdout, filepath.Join(env.HomeDir, ".config", "trello", "config.yaml")) {
		t.Errorf("global path missing: %q", stdout)
	}
}

func TestConfigInitWithBoardFlag(t *testing.T) {
	env := testEnv(t, "")
	code, stdout, _ := runCLI(t, []string{"config", "init", "--board", "my-board"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stdout)
	}
	if !strings.Contains(stdout, ".trello.yaml") {
		t.Errorf("expected written path in output: %q", stdout)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "board: my-board") {
		t.Errorf("unexpected file content: %q", content)
	}
}

func TestConfigInitBoardValidated(t *testing.T) {
	// With credentials, --board is validated against the member's boards and
	// the resolved board id is persisted.
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": testBoardsFixture,
	})
	defer srv.Close()

	env := credsEnv(t)
	code, _, stderr := runCLI(t, []string{"config", "init", "--board", "Trello Platform Changes"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "board: 5abbe4b7ddc1b351ef961414") {
		t.Errorf("expected the resolved board id to be persisted: %q", content)
	}
}

func TestConfigInitBoardNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": `[]`,
	})
	defer srv.Close()

	env := credsEnv(t)
	code, _, stderr := runCLI(t, []string{"config", "init", "--board", "nonexistent"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("stderr should mention the board ref: %q", stderr)
	}
}

func TestConfigInitBoardUnvalidatedWarns(t *testing.T) {
	// Without credentials the reference is persisted as-is with a warning.
	env := testEnv(t, "")
	code, _, stderr := runCLI(t, []string{"config", "init", "--board", "my-board"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "unvalidated") {
		t.Errorf("stderr should warn that the board was persisted unvalidated: %q", stderr)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "board: my-board") {
		t.Errorf("expected the raw ref to be persisted: %q", content)
	}
}

func TestConfigInitWritesKeyTokenFromFlags(t *testing.T) {
	// Credentials are present, so --board "b" is validated and resolved.
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": `[{"id":"board-1","name":"b","shortLink":"BBB"}]`,
	})
	defer srv.Close()

	env := testEnv(t, "")
	code, _, stderr := runCLI(t, []string{"config", "init", "--board", "b", "--key", "flagkey", "--token", "flagtoken"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"board: board-1", "key: flagkey", "token: flagtoken"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("file missing %q: %q", want, content)
		}
	}
}

func TestConfigInitWritesKeyTokenFromEnv(t *testing.T) {
	// Credentials are present, so --board "b" is validated and resolved.
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": `[{"id":"board-1","name":"b","shortLink":"BBB"}]`,
	})
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "envkey"
	env.Token = "envtoken"
	code, _, stderr := runCLI(t, []string{"config", "init", "--board", "b"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "board: board-1") || !strings.Contains(string(content), "key: envkey") || !strings.Contains(string(content), "token: envtoken") {
		t.Errorf("env creds not written: %q", content)
	}
}

func TestConfigInitInteractive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
			{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
		]`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	env.IsTTY = func() bool { return true }
	code, stdout, _ := runCLI(t, []string{"config", "init"}, strings.NewReader("2\n"), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Release Planning") {
		t.Errorf("prompt should list boards: %q", stdout)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "board: 5abbe4b7ddc1b351ef961415") {
		t.Errorf("expected second board's id to be written: %q", content)
	}
}

func TestConfigInitInteractiveInvalidSelection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"b1","name":"Only","shortLink":"AAA"}]`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	env.IsTTY = func() bool { return true }
	code, _, _ := runCLI(t, []string{"config", "init"}, strings.NewReader("99\n"), env, srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
}

func TestConfigInitNoTTYWithoutBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"b1","name":"Only","shortLink":"AAA"}]`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	env.IsTTY = func() bool { return false }
	code, _, stderr := runCLI(t, []string{"config", "init"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "--board") {
		t.Errorf("stderr should hint at --board: %q", stderr)
	}
}

func TestConfigInitMergesExisting(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "custom: keep-me\n")
	code, _, _ := runCLI(t, []string{"config", "init", "--board", "new-board"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	content, err := os.ReadFile(filepath.Join(env.HomeDir, "proj", ".trello.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "custom: keep-me") {
		t.Errorf("existing keys not preserved: %q", content)
	}
}

func TestUnknownConfigKeyWarns(t *testing.T) {
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: b\nbord: typo\n")
	code, _, stderr := runCLI(t, []string{"config", "show"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "bord") {
		t.Errorf("expected unknown-key warning on stderr: %q", stderr)
	}
}

func TestBoardUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"board", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}

func TestConfigUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"config", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}

func TestBareGroupCommandShowsHelp(t *testing.T) {
	for _, args := range [][]string{{"board"}, {"config"}} {
		code, stdout, _ := runCLI(t, args, strings.NewReader(""), testEnv(t, ""), "")
		if code != output.ExitOK {
			t.Errorf("trello %s: exit = %d, want 0", args[0], code)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("trello %s: expected help, got %q", args[0], stdout)
		}
	}
}

func TestConfigEnvErrorsExitConfig(t *testing.T) {
	t.Run("getwd fails", func(t *testing.T) {
		env := testEnv(t, "")
		env.Getwd = func() (string, error) { return "", errors.New("getwd failed") }
		code, _, stderr := runCLI(t, []string{"config", "show"}, strings.NewReader(""), env, "")
		if code != output.ExitConfig {
			t.Errorf("exit = %d, want %d", code, output.ExitConfig)
		}
		if !strings.Contains(stderr, "working directory") {
			t.Errorf("stderr should explain the failure: %q", stderr)
		}
	})

	t.Run("user config dir fails", func(t *testing.T) {
		env := testEnv(t, "")
		env.UserConfigDir = func() (string, error) { return "", errors.New("no config dir") }
		code, _, stderr := runCLI(t, []string{"config", "show"}, strings.NewReader(""), env, "")
		if code != output.ExitConfig {
			t.Errorf("exit = %d, want %d", code, output.ExitConfig)
		}
		if !strings.Contains(stderr, "user config directory") {
			t.Errorf("stderr should explain the failure: %q", stderr)
		}
	})
}

func TestConfigInitJSON(t *testing.T) {
	env := testEnv(t, "")
	code, stdout, _ := runCLI(t, []string{"config", "init", "--board", "my-board", "--json"}, strings.NewReader(""), env, "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{`"path"`, `"board": "my-board"`, `"written": true`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("JSON result missing %s: %q", want, stdout)
		}
	}
}

func TestConfigInitInteractiveJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
			{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
		]`)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	env.IsTTY = func() bool { return true }
	code, stdout, stderr := runCLI(t, []string{"config", "init", "--json"}, strings.NewReader("2\n"), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	// stdout must be pure JSON; the prompt goes to stderr.
	if !strings.Contains(stdout, `"board": "5abbe4b7ddc1b351ef961415"`) {
		t.Errorf("expected JSON result on stdout: %q", stdout)
	}
	if strings.Contains(stdout, "Boards:") {
		t.Errorf("prompt leaked into JSON stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "Release Planning") {
		t.Errorf("prompt should be on stderr: %q", stderr)
	}
}

// TestNoArgsViolationsExitUsage guards the SPEC §3.5 exit-code contract for
// leaf commands using Args: cobra.NoArgs: extra positional args produce the
// "unknown command" error which must exit 2 (usage error).
func TestNoArgsViolationsExitUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"whoami extra", []string{"whoami", "extra"}},
		{"board list extra", []string{"board", "list", "extra"}},
		{"list list extra", []string{"list", "list", "extra"}},
		{"card list extra", []string{"card", "list", "extra"}},
		{"config show extra", []string{"config", "show", "extra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tt.args, strings.NewReader(""), testEnv(t, ""), "")
			if code != output.ExitUsage {
				t.Errorf("exit = %d, want %d (usage error); stderr: %q", code, output.ExitUsage, stderr)
			}
		})
	}
}

// TestExactArgsViolationsExitUsage guards the SPEC §3.5 exit-code contract:
// argument-count violations on every Args: exactArgs(N) command must exit 2
// (usage error), not 1 (generic runtime error).
func TestExactArgsViolationsExitUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"list get 0 args", []string{"list", "get"}},
		{"list get 2 args", []string{"list", "get", "a", "b"}},
		{"list create 0 args", []string{"list", "create"}},
		{"list update 0 args", []string{"list", "update"}},
		{"list archive 0 args", []string{"list", "archive"}},
		{"card get 0 args", []string{"card", "get"}},
		{"card create 0 args", []string{"card", "create"}},
		{"card update 0 args", []string{"card", "update"}},
		{"card move 0 args", []string{"card", "move"}},
		{"card archive 0 args", []string{"card", "archive"}},
		{"card delete 0 args", []string{"card", "delete"}},
		{"comment add 0 args", []string{"comment", "add"}},
		{"comment list 0 args", []string{"comment", "list"}},
		{"checklist list 0 args", []string{"checklist", "list"}},
		{"checklist get 0 args", []string{"checklist", "get"}},
		{"checklist create 0 args", []string{"checklist", "create"}},
		{"checklist add-item 0 args", []string{"checklist", "add-item"}},
		{"checklist check-item 1 arg", []string{"checklist", "check-item", "cl1"}},
		{"checklist check-item 3 args", []string{"checklist", "check-item", "cl1", "ci1", "extra"}},
		{"search 0 args", []string{"search"}},
		{"search 2 args", []string{"search", "a", "b"}},
		{"raw 0 args", []string{"raw"}},
		{"raw missing path", []string{"raw", "GET"}},
		{"raw 3 args", []string{"raw", "GET", "/x", "extra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tt.args, strings.NewReader(""), testEnv(t, ""), "")
			if code != output.ExitUsage {
				t.Errorf("exit = %d, want %d (usage error); stderr: %q", code, output.ExitUsage, stderr)
			}
			if !strings.Contains(stderr, "Run 'trello --help' for usage.") {
				t.Errorf("stderr should include the usage hint: %q", stderr)
			}
		})
	}
}
