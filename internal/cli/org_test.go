package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testOrgsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961414","name":"acme","displayName":"Acme Inc","desc":"The Acme team","url":"https://trello.com/acme","website":"https://acme.example.com"},
	{"id":"5abbe4b7ddc1b351ef961415","name":"globex","displayName":"Globex","desc":"","url":"https://trello.com/globex","website":""}
]`

func TestOrgList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/organizations" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, testOrgsFixture)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"org", "list"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"acme", "Acme Inc", "globex", "Globex"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestOrgGetByID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"id":"5abbe4b7ddc1b351ef961414","name":"acme","displayName":"Acme Inc","desc":"The Acme team","url":"https://trello.com/acme","website":"https://acme.example.com"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"org", "get", "5abbe4b7ddc1b351ef961414"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/organizations/5abbe4b7ddc1b351ef961414" {
		t.Errorf("path = %q, want direct GET by id", gotPath)
	}
	for _, want := range []string{"acme", "Acme Inc", "The Acme team", "https://acme.example.com"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestOrgGetByName(t *testing.T) {
	var gotListed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/members/me/organizations" {
			gotListed = true
			fmt.Fprint(w, testOrgsFixture)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"org", "get", "acme"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !gotListed {
		t.Error("expected the member orgs listing for name resolution")
	}
	if !strings.Contains(stdout, "Acme Inc") {
		t.Errorf("output missing org data: %q", stdout)
	}
}

func TestOrgBoards(t *testing.T) {
	var gotBoardsPath bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/organizations":
			fmt.Fprint(w, testOrgsFixture)
		case "/organizations/5abbe4b7ddc1b351ef961414/boards":
			gotBoardsPath = true
			fmt.Fprint(w, `[
				{"id":"b1","name":"Trello Platform Changes","shortLink":"3CsPkqOF","closed":false},
				{"id":"b2","name":"Release Planning","shortLink":"AbCdEf12","closed":true}
			]`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"org", "boards", "acme"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !gotBoardsPath {
		t.Error("expected a request to /organizations/{id}/boards")
	}
	for _, want := range []string{"Trello Platform Changes", "Release Planning", "3CsPkqOF", "AbCdEf12"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestOrgMembers(t *testing.T) {
	var gotMembersPath bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/organizations":
			fmt.Fprint(w, testOrgsFixture)
		case "/organizations/5abbe4b7ddc1b351ef961414/members":
			gotMembersPath = true
			fmt.Fprint(w, `[{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}]`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"org", "members", "acme"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !gotMembersPath {
		t.Error("expected a request to /organizations/{id}/members")
	}
	for _, want := range []string{"bentleycook", "Bentley Cook"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestOrgRefNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/organizations": `[]`,
	})
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"org", "get", "nonexistent"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "nonexistent") || !strings.Contains(stderr, "org list") {
		t.Errorf("stderr should mention the ref and hint: %q", stderr)
	}
}

func TestOrgGetNoArgs(t *testing.T) {
	code, _, _ := runCLI(t, []string{"org", "get"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestOrgUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"org", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
