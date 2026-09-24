package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testActionsFixture = `[
	{
		"id":"5dc9b507756e182c76007621",
		"idMemberCreator":"5b02e7f4e1facdc393169f9d",
		"type":"commentCard",
		"date":"2020-03-09T19:41:51.396Z",
		"data":{"text":"Can never go wrong with bowie","card":{"id":"c1","name":"Bowie"},"board":{"id":"b1","name":"Mullets"}},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}
	},
	{
		"id":"5dc9b507756e182c76007622",
		"idMemberCreator":"5b02e7f4e1facdc393169f9e",
		"type":"updateCard",
		"date":"2020-03-10T10:00:00.000Z",
		"data":{"card":{"id":"c1","name":"Bowie"},"list":{"id":"l1","name":"Amazing"},"board":{"id":"b1","name":"Mullets"}},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9e","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
	}
]`

func TestActionList(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/actions":
			gotFilter = r.URL.Query().Get("filter")
			fmt.Fprint(w, testActionsFixture)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"action", "list", "--filter", "commentCard"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilter != "commentCard" {
		t.Errorf("filter = %q, want commentCard", gotFilter)
	}
	for _, want := range []string{"commentCard", "updateCard", "bentleycook", "bobloblaw", "5dc9b507756e182c76007621"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestActionListByCard(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, testActionsFixture)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"action", "list", "--card", "AbCdEf01"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/cards/AbCdEf01/actions" {
		t.Errorf("path = %q, want card actions path", gotPath)
	}
	if !strings.Contains(stdout, "commentCard") {
		t.Errorf("output missing action data: %q", stdout)
	}
}

func TestActionListDefaultFilter(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/actions":
			gotFilter = r.URL.Query().Get("filter")
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"action", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilter != "all" {
		t.Errorf("filter = %q, want all (default)", gotFilter)
	}
}

func TestActionGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"id":"a1","idMemberCreator":"5b02e7f4e1facdc393169f9d","type":"commentCard",
			"date":"2020-03-09T19:41:51.396Z",
			"data":{"text":"hello","card":{"id":"c1","name":"Bowie"},"board":{"id":"b1","name":"Mullets"}},
			"memberCreator":{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}
		}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"action", "get", "a1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"commentCard", "bentleycook", "hello", "Bowie"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestActionGetNoArgs(t *testing.T) {
	code, _, _ := runCLI(t, []string{"action", "get"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestActionUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"action", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
