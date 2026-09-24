package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testBoardsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"}
]`

const testListsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961415","name":"To do","closed":false,"pos":16384,"idBoard":"5abbe4b7ddc1b351ef961414"},
	{"id":"5abbe4b7ddc1b351ef961416","name":"Done","closed":false,"pos":32768,"idBoard":"5abbe4b7ddc1b351ef961414"}
]`

const testCardsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961417","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415","idBoard":"5abbe4b7ddc1b351ef961414","closed":false,"shortLink":"AbCdEf01","pos":65535,"url":"https://trello.com/c/AbCdEf01/ship-it"},
	{"id":"5abbe4b7ddc1b351ef961418","name":"Fix bug","idList":"5abbe4b7ddc1b351ef961416","idBoard":"5abbe4b7ddc1b351ef961414","closed":false,"shortLink":"AbCdEf02","pos":131071,"url":"https://trello.com/c/AbCdEf02/fix-bug"}
]`

// fixtureServer serves fixed bodies keyed by request path.
func fixtureServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
}

// credsEnv returns a test env with credentials set.
func credsEnv(t *testing.T) Env {
	t.Helper()
	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	return env
}

// boardEnv returns a test env with credentials and a configured board so
// commands that require board resolution work without --board.
func boardEnv(t *testing.T) Env {
	t.Helper()
	env := credsEnv(t)
	writeConfig(t, filepath.Join(env.HomeDir, "proj", ".trello.yaml"), "board: 5abbe4b7ddc1b351ef961414\n")
	return env
}

func TestListList(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/lists": testListsFixture,
	})
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"list", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"To do", "Done", "5abbe4b7ddc1b351ef961415", "16384"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestListListMissingBoard(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"list", "list"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "no board selected") {
		t.Errorf("stderr should explain the missing board: %q", stderr)
	}
}

func TestListListBoardNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards": `[]`,
	})
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"list", "list", "--board", "nonexistent"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("stderr should mention the board ref: %q", stderr)
	}
}

func TestListListMissingCreds(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"list", "list"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "credentials") {
		t.Errorf("stderr should mention credentials: %q", stderr)
	}
}

func TestListGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"l1","name":"To do","closed":false,"pos":16384,"idBoard":"b1"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"list", "get", "l1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "To do") || !strings.Contains(stdout, "l1") {
		t.Errorf("output missing list data: %q", stdout)
	}
}

func TestListCreate(t *testing.T) {
	var gotName, gotBoard, gotPos string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/lists":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			q := r.URL.Query()
			gotName, gotBoard, gotPos = q.Get("name"), q.Get("idBoard"), q.Get("pos")
			fmt.Fprint(w, `{"id":"l9","name":"New list","idBoard":"5abbe4b7ddc1b351ef961414","pos":65535}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"list", "create", "New list", "--pos", "65535"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "New list" || gotBoard != "5abbe4b7ddc1b351ef961414" || gotPos != "65535" {
		t.Errorf("query params: name=%q board=%q pos=%q", gotName, gotBoard, gotPos)
	}
	if !strings.Contains(stdout, "l9") {
		t.Errorf("output should include the new list id: %q", stdout)
	}
}

func TestListUpdate(t *testing.T) {
	var gotName, gotClosed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/l1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotClosed = r.URL.Query().Get("closed")
		fmt.Fprint(w, `{"id":"l1","name":"Renamed","closed":true}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"list", "update", "l1", "--name", "Renamed", "--closed"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "Renamed" || gotClosed != "true" {
		t.Errorf("query params: name=%q closed=%q", gotName, gotClosed)
	}
}

func TestListUpdateNoFlags(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"list", "update", "l1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--name") {
		t.Errorf("stderr should list the required flags: %q", stderr)
	}
}

func TestListArchive(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/l1/closed" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotValue = r.URL.Query().Get("value")
		fmt.Fprint(w, `{"id":"l1","name":"Done","closed":true}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"list", "archive", "l1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotValue != "true" {
		t.Errorf("value = %q, want true", gotValue)
	}
}

func TestListUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"list", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}

func TestCardUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
