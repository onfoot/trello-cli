package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const searchFixture = `{
	"boards":[{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"}],
	"cards":[{"id":"5abbe4b7ddc1b351ef961417","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415","idBoard":"5abbe4b7ddc1b351ef961414","closed":false,"shortLink":"AbCdEf01","pos":65535}],
	"members":[{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}],
	"organizations":[{"id":"o1","name":"acme","displayName":"Acme Inc"}]
}`

func TestSearchDefaultModelTypes(t *testing.T) {
	var gotQuery, gotModelTypes string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotQuery = r.URL.Query().Get("query")
		gotModelTypes = r.URL.Query().Get("modelTypes")
		fmt.Fprint(w, searchFixture)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"search", "ship it"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotQuery != "ship it" {
		t.Errorf("query = %q, want ship it", gotQuery)
	}
	if gotModelTypes != "cards,boards" {
		t.Errorf("modelTypes = %q, want cards,boards (default)", gotModelTypes)
	}
	for _, want := range []string{"Boards", "Cards", "Trello Platform Changes", "Ship it"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestSearchModelTypeFlags(t *testing.T) {
	var gotModelTypes string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModelTypes = r.URL.Query().Get("modelTypes")
		fmt.Fprint(w, searchFixture)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"search", "x", "--boards", "--members", "--organizations"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotModelTypes != "boards,members,organizations" {
		t.Errorf("modelTypes = %q, want boards,members,organizations", gotModelTypes)
	}
}

func TestSearchWithBoardScoping(t *testing.T) {
	var gotIDBoards string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/search":
			gotIDBoards = r.URL.Query().Get("idBoards")
			fmt.Fprint(w, searchFixture)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"search", "x", "--board", "Trello Platform Changes"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotIDBoards != "5abbe4b7ddc1b351ef961414" {
		t.Errorf("idBoards = %q, want resolved board id", gotIDBoards)
	}
}

func TestSearchBoardNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": `[]`,
	})
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"search", "x", "--board", "nonexistent"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("stderr should mention the board ref: %q", stderr)
	}
}

func TestSearchConfigBoardDoesNotScope(t *testing.T) {
	// A config-default board must not silently narrow search results.
	var gotIDBoards string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			gotIDBoards = r.URL.Query().Get("idBoards")
		}
		fmt.Fprint(w, searchFixture)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"search", "x"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotIDBoards != "" {
		t.Errorf("idBoards = %q, want unset (config board must not scope search)", gotIDBoards)
	}
}

func TestSearchOnlyRequestedGroupsRendered(t *testing.T) {
	// The default search (cards+boards) must not render the members or
	// organizations groups even when the API returns them.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, searchFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"search", "x"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Boards", "Cards"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("default search should render %s: %q", want, stdout)
		}
	}
	for _, unwanted := range []string{"Members", "Organizations"} {
		if strings.Contains(stdout, unwanted) {
			t.Errorf("default search should not render %s: %q", unwanted, stdout)
		}
	}
}

func TestSearchMembersGroupRendered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, searchFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"search", "x", "--members"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Members") || !strings.Contains(stdout, "bentleycook") {
		t.Errorf("--members should render the members group: %q", stdout)
	}
}

func TestSearchEmptyGroupShowsNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"boards":[],"cards":[],"members":[],"organizations":[]}`)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"search", "x", "--boards", "--cards"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Boards", "(no results)", "Cards"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}
