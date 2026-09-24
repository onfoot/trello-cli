package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestCompletionBashScript(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"completion", "bash"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if stdout == "" {
		t.Fatal("bash script should be non-empty")
	}
	for _, want := range []string{"trello", "complete"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("bash script missing %q", want)
		}
	}
}

func TestCompletionZshScript(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"completion", "zsh"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if stdout == "" {
		t.Fatal("zsh script should be non-empty")
	}
	if !strings.Contains(stdout, "#compdef trello") {
		t.Errorf("zsh script missing the compdef directive")
	}
}

func TestCompletionFishScript(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"completion", "fish"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if stdout == "" {
		t.Fatal("fish script should be non-empty")
	}
	if !strings.Contains(stdout, "complete") {
		t.Errorf("fish script missing complete directives")
	}
}

func TestCompletionScriptsResolveSubcommands(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"completion", "bash"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, sub := range []string{
		"board", "list", "card", "comment", "checklist", "label", "member",
		"attachment", "customfield", "action", "notification", "org",
		"search", "config", "whoami", "raw", "completion",
	} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("bash script does not resolve subcommand %q", sub)
		}
	}
}

func TestCompletionInstallBash(t *testing.T) {
	dir := t.TempDir()
	code, stdout, _ := runCLI(t, []string{"completion", "install", "bash", "--dir", dir}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	path := filepath.Join(dir, "bash-completion", "completions", "trello")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
	if len(content) == 0 {
		t.Error("installed bash script should be non-empty")
	}
	if !strings.Contains(stdout, path) {
		t.Errorf("output should mention the installed path: %q", stdout)
	}
}

func TestCompletionInstallZsh(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runCLI(t, []string{"completion", "install", "zsh", "--dir", dir}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	path := filepath.Join(dir, "zsh", "completions", "_trello")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
	if len(content) == 0 {
		t.Error("installed zsh script should be non-empty")
	}
}

func TestCompletionInstallFish(t *testing.T) {
	dir := t.TempDir()
	code, _, _ := runCLI(t, []string{"completion", "install", "fish", "--dir", dir}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	path := filepath.Join(dir, "fish", "completions", "trello.fish")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
	if len(content) == 0 {
		t.Error("installed fish script should be non-empty")
	}
}

func TestCompletionInstallJSON(t *testing.T) {
	dir := t.TempDir()
	code, stdout, _ := runCLI(t, []string{"completion", "install", "bash", "--dir", dir, "--json"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	var result struct {
		Shell string `json:"shell"`
		Path  string `json:"path"`
		Hint  string `json:"hint"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout)
	}
	if result.Shell != "bash" || result.Path == "" || result.Hint == "" {
		t.Errorf("unexpected JSON result: %+v", result)
	}
	if strings.Contains(stdout, "Hint:") || strings.Contains(stdout, "Installed") {
		t.Errorf("JSON mode should keep hints out of stdout: %q", stdout)
	}
}

func TestCompletionInstallInvalidShell(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"completion", "install", "powershell", "--dir", t.TempDir()}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "bash, zsh, fish") {
		t.Errorf("stderr should list valid shells: %q", stderr)
	}
}

func TestCompletionInstallNoArgs(t *testing.T) {
	code, _, _ := runCLI(t, []string{"completion", "install"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestCompletionBareShowsHelp(t *testing.T) {
	code, stdout, _ := runCLI(t, []string{"completion"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("expected help output: %q", stdout)
	}
}

func TestCompletionUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"completion", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
