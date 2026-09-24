package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestMemberList(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                       testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/members": testMembersFixture,
	})
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"member", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"bentleycook", "Bentley Cook", "bobloblaw", "Bob Loblaw"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestMemberListMissingBoard(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"member", "list"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "no board selected") {
		t.Errorf("stderr should explain the missing board: %q", stderr)
	}
}

func TestMemberGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/bentleycook" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"member", "get", "bentleycook"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "bentleycook") || !strings.Contains(stdout, "Bentley Cook") {
		t.Errorf("output missing member data: %q", stdout)
	}
}

func TestMemberGetMe(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"member", "get", "me", "--json"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/members/me" {
		t.Errorf("path = %q, want /members/me", gotPath)
	}
	if !strings.Contains(stdout, `"username": "bentleycook"`) {
		t.Errorf("expected JSON output: %q", stdout)
	}
}

func TestMemberGetNoArgs(t *testing.T) {
	code, _, _ := runCLI(t, []string{"member", "get"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestMemberUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"member", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
