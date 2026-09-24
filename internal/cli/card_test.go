package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestCardList(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/cards": testCardsFixture,
	})
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"card", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Ship it", "Fix bug", "5abbe4b7ddc1b351ef961417"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestCardListWithListFilter(t *testing.T) {
	var gotListPath bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/lists":
			fmt.Fprint(w, testListsFixture)
		case "/lists/5abbe4b7ddc1b351ef961415/cards":
			gotListPath = true
			fmt.Fprint(w, `[{"id":"5abbe4b7ddc1b351ef961417","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415"}]`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"card", "list", "--list", "To do"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !gotListPath {
		t.Error("expected a request to /lists/{id}/cards")
	}
	if !strings.Contains(stdout, "Ship it") {
		t.Errorf("output missing card: %q", stdout)
	}
}

func TestCardListListNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/lists": testListsFixture,
	})
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"card", "list", "--list", "bogus"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr should mention the list ref: %q", stderr)
	}
}

func TestCardGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","desc":"details","idList":"l1","idBoard":"b1","closed":false,"due":"2026-09-04T12:00:00Z","shortLink":"AbCdEf01","pos":65535,"url":"https://trello.com/c/AbCdEf01"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"card", "get", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Ship it", "c1", "2026-09-04T12:00:00Z"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestCardCreate(t *testing.T) {
	var gotName, gotList, gotDesc, gotDue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/lists":
			fmt.Fprint(w, testListsFixture)
		case "/cards":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			q := r.URL.Query()
			gotName, gotList, gotDesc, gotDue = q.Get("name"), q.Get("idList"), q.Get("desc"), q.Get("due")
			fmt.Fprint(w, `{"id":"c9","name":"New card","idList":"5abbe4b7ddc1b351ef961415"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{
		"card", "create", "New card",
		"--list", "To do",
		"--desc", "details",
		"--due", "2026-09-04T12:00:00Z",
	}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	// --list "To do" must be resolved to the list id before POST /cards.
	if gotName != "New card" || gotList != "5abbe4b7ddc1b351ef961415" || gotDesc != "details" || gotDue != "2026-09-04T12:00:00Z" {
		t.Errorf("query params: name=%q list=%q desc=%q due=%q", gotName, gotList, gotDesc, gotDue)
	}
	if !strings.Contains(stdout, "c9") {
		t.Errorf("output should include the new card id: %q", stdout)
	}
}

func TestCardCreateRequiresList(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "create", "New card"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--list") {
		t.Errorf("stderr should mention --list: %q", stderr)
	}
}

func TestCardCreateInvalidDue(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "create", "New card", "--list", "To do", "--due", "not-a-date"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "RFC3339") {
		t.Errorf("stderr should mention RFC3339: %q", stderr)
	}
}

func TestCardUpdate(t *testing.T) {
	var gotName, gotClosed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotClosed = r.URL.Query().Get("closed")
		fmt.Fprint(w, `{"id":"c1","name":"Renamed","closed":true}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--name", "Renamed", "--closed"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "Renamed" || gotClosed != "true" {
		t.Errorf("query params: name=%q closed=%q", gotName, gotClosed)
	}
}

func TestCardUpdateNoFlags(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "update", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--name") {
		t.Errorf("stderr should list the required flags: %q", stderr)
	}
}

func TestCardUpdateDueNullClears(t *testing.T) {
	var gotDue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotDue = r.URL.Query().Get("due")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it"}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--due", "null"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotDue != "null" {
		t.Errorf("due = %q, want null (clear sentinel)", gotDue)
	}
}

func TestCardUpdateDueEmptyClears(t *testing.T) {
	var gotDue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDue = r.URL.Query().Get("due")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it"}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--due", ""}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotDue != "null" {
		t.Errorf("due = %q, want null (empty sentinel)", gotDue)
	}
}

func TestCardUpdateDueValid(t *testing.T) {
	var gotDue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDue = r.URL.Query().Get("due")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it"}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--due", "2026-09-04T12:00:00Z"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotDue != "2026-09-04T12:00:00Z" {
		t.Errorf("due = %q, want the RFC3339 value", gotDue)
	}
}

func TestCardUpdateInvalidDue(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "update", "c1", "--due", "not-a-date"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "RFC3339") {
		t.Errorf("stderr should mention RFC3339: %q", stderr)
	}
}

func TestCardUpdateWithListResolves(t *testing.T) {
	var gotList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/lists":
			fmt.Fprint(w, testListsFixture)
		case "/cards/c1":
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			gotList = r.URL.Query().Get("idList")
			fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"5abbe4b7ddc1b351ef961416"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--list", "Done"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotList != "5abbe4b7ddc1b351ef961416" {
		t.Errorf("idList = %q, want resolved list id", gotList)
	}
}

func TestCardMove(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/lists":
			fmt.Fprint(w, testListsFixture)
		case "/cards/c1/idList":
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			gotValue = r.URL.Query().Get("value")
			fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"5abbe4b7ddc1b351ef961416"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "move", "c1", "--list", "Done"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotValue != "5abbe4b7ddc1b351ef961416" {
		t.Errorf("value = %q, want resolved list id", gotValue)
	}
}

func TestCardMoveRequiresList(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"card", "move", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--list") {
		t.Errorf("stderr should mention --list: %q", stderr)
	}
}

func TestCardArchive(t *testing.T) {
	var gotClosed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotClosed = r.URL.Query().Get("closed")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","closed":true}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"card", "archive", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotClosed != "true" {
		t.Errorf("closed = %q, want true", gotClosed)
	}
}

func TestCardDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"card", "delete", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "c1") {
		t.Errorf("output should mention the deleted card: %q", stdout)
	}
}

func TestCardDeleteJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"card", "delete", "c1", "--json"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, `"deleted": true`) || !strings.Contains(stdout, `"id": "c1"`) {
		t.Errorf("expected JSON result: %q", stdout)
	}
}

func TestCardGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such card"}`)
	}))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"card", "get", "missing"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitNotFound {
		t.Errorf("exit = %d, want %d", code, output.ExitNotFound)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should surface the status: %q", stderr)
	}
}

func TestCardCreateListIDNoBoard(t *testing.T) {
	// A full 24-hex --list id must skip board resolution entirely.
	var gotList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotList = r.URL.Query().Get("idList")
		fmt.Fprint(w, `{"id":"c1","name":"New card","idList":"5abbe4b7ddc1b351ef961415"}`)
	}))
	defer srv.Close()

	env := credsEnv(t) // creds but NO board configured
	code, _, _ := runCLI(t, []string{"card", "create", "New card", "--list", "5abbe4b7ddc1b351ef961415"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotList != "5abbe4b7ddc1b351ef961415" {
		t.Errorf("idList = %q, want the raw list id", gotList)
	}
}

func TestCardMoveListIDNoBoard(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/idList" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotValue = r.URL.Query().Get("value")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415"}`)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, _, _ := runCLI(t, []string{"card", "move", "c1", "--list", "5abbe4b7ddc1b351ef961415"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotValue != "5abbe4b7ddc1b351ef961415" {
		t.Errorf("value = %q, want the raw list id", gotValue)
	}
}

func TestCardListListIDNoBoard(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `[{"id":"c1","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415"}]`)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, _, _ := runCLI(t, []string{"card", "list", "--list", "5abbe4b7ddc1b351ef961415"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/lists/5abbe4b7ddc1b351ef961415/cards" {
		t.Errorf("path = %q, want the list cards path", gotPath)
	}
}

func TestCardUpdateListIDNoBoard(t *testing.T) {
	var gotList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotList = r.URL.Query().Get("idList")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415"}`)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, _, _ := runCLI(t, []string{"card", "update", "c1", "--list", "5abbe4b7ddc1b351ef961415"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotList != "5abbe4b7ddc1b351ef961415" {
		t.Errorf("idList = %q, want the raw list id", gotList)
	}
}

func TestCardGetHumanOmitsEmptyDesc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"l1","idBoard":"b1","closed":false}`)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, _ := runCLI(t, []string{"card", "get", "c1"}, strings.NewReader(""), env, srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if strings.Contains(stdout, "Desc:") {
		t.Errorf("empty desc should be omitted from human output: %q", stdout)
	}
	if !strings.Contains(stdout, "Ship it") {
		t.Errorf("output missing card name: %q", stdout)
	}
}
