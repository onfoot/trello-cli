package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testLabelsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961419","name":"Overdue","color":"red","idBoard":"5abbe4b7ddc1b351ef961414"},
	{"id":"5abbe4b7ddc1b351ef96141a","name":"Needs review","color":"blue","idBoard":"5abbe4b7ddc1b351ef961414"}
]`

const testMembersFixture = `[
	{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true},
	{"id":"5b02e7f4e1facdc393169f9e","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
]`

func TestLabelList(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                      testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/labels": testLabelsFixture,
	})
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Overdue", "Needs review", "red", "5abbe4b7ddc1b351ef961419"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestLabelListMissingBoard(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"label", "list"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "no board selected") {
		t.Errorf("stderr should explain the missing board: %q", stderr)
	}
}

func TestLabelGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/labels/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"l1","name":"Overdue","color":"red","idBoard":"b1"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "get", "l1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Overdue") || !strings.Contains(stdout, "red") {
		t.Errorf("output missing label data: %q", stdout)
	}
}

func TestLabelCreate(t *testing.T) {
	var gotName, gotColor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/boards/5abbe4b7ddc1b351ef961414/labels":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			gotName = r.URL.Query().Get("name")
			gotColor = r.URL.Query().Get("color")
			fmt.Fprint(w, `{"id":"l9","name":"Overdue","color":"red","idBoard":"5abbe4b7ddc1b351ef961414"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "create", "Overdue", "--color", "red"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "Overdue" || gotColor != "red" {
		t.Errorf("query params: name=%q color=%q", gotName, gotColor)
	}
	if !strings.Contains(stdout, "l9") {
		t.Errorf("output should include the new label id: %q", stdout)
	}
}

func TestLabelCreateInvalidColor(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"label", "create", "Overdue", "--color", "chartreuse"}, strings.NewReader(""), boardEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "yellow, purple, blue, red, green, orange, black, sky, pink, lime") {
		t.Errorf("stderr should list valid colors: %q", stderr)
	}
}

func TestLabelUpdate(t *testing.T) {
	var gotName, gotColor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/labels/l1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotColor = r.URL.Query().Get("color")
		fmt.Fprint(w, `{"id":"l1","name":"Renamed","color":"green"}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"label", "update", "l1", "--name", "Renamed", "--color", "green"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "Renamed" || gotColor != "green" {
		t.Errorf("query params: name=%q color=%q", gotName, gotColor)
	}
}

func TestLabelUpdateNoFlags(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"label", "update", "l1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--name") {
		t.Errorf("stderr should list the required flags: %q", stderr)
	}
}

func TestLabelUpdateInvalidColor(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"label", "update", "l1", "--color", "chartreuse"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "must be one of") {
		t.Errorf("stderr should explain valid colors: %q", stderr)
	}
}

func TestLabelDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/labels/l1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "delete", "l1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "l1") {
		t.Errorf("output should mention the deleted label: %q", stdout)
	}
}

func TestLabelAdd(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/c1/board":
			fmt.Fprint(w, `{"id":"5abbe4b7ddc1b351ef961414","name":"Board"}`)
		case "/boards/5abbe4b7ddc1b351ef961414/labels":
			fmt.Fprint(w, testLabelsFixture)
		case "/cards/c1/idLabels":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			gotValue = r.URL.Query().Get("value")
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "add", "c1", "Overdue"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	// "Overdue" must be resolved to its label id before the POST.
	if gotValue != "5abbe4b7ddc1b351ef961419" {
		t.Errorf("value = %q, want resolved label id", gotValue)
	}
	if !strings.Contains(stdout, "Added label") {
		t.Errorf("output should confirm the add: %q", stdout)
	}
}

func TestLabelAddByID(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/c1/board":
			fmt.Fprint(w, `{"id":"b1"}`)
		case "/boards/b1/labels":
			fmt.Fprint(w, testLabelsFixture)
		case "/cards/c1/idLabels":
			gotValue = r.URL.Query().Get("value")
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"label", "add", "c1", "5abbe4b7ddc1b351ef96141a"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotValue != "5abbe4b7ddc1b351ef96141a" {
		t.Errorf("value = %q, want the label id", gotValue)
	}
}

func TestLabelAddNoMatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/cards/c1/board":   `{"id":"b1"}`,
		"/boards/b1/labels": testLabelsFixture,
	})
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"label", "add", "c1", "bogus"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr should mention the label ref: %q", stderr)
	}
}

func TestLabelRemove(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/c1/board":
			fmt.Fprint(w, `{"id":"5abbe4b7ddc1b351ef961414"}`)
		case "/boards/5abbe4b7ddc1b351ef961414/labels":
			fmt.Fprint(w, testLabelsFixture)
		case "/cards/c1/idLabels/5abbe4b7ddc1b351ef961419":
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s, want DELETE", r.Method)
			}
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"label", "remove", "c1", "Overdue"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/cards/c1/idLabels/5abbe4b7ddc1b351ef961419" {
		t.Errorf("path = %q, want the resolved label id in the path", gotPath)
	}
	if !strings.Contains(stdout, "Removed label") {
		t.Errorf("output should confirm the removal: %q", stdout)
	}
}

func TestLabelAddWrongArgCount(t *testing.T) {
	code, _, _ := runCLI(t, []string{"label", "add", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestLabelUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"label", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
