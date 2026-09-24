package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestCommentAdd(t *testing.T) {
	var gotText, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotPath = r.URL.Path
		gotText = r.URL.Query().Get("text")
		fmt.Fprint(w, `{"id":"a1","idMemberCreator":"m1","type":"commentCard","date":"2020-03-09T19:41:51.396Z","data":{"text":"hello"}}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"comment", "add", "AbCdEf01", "--text", "hello"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/cards/AbCdEf01/actions/comments" {
		t.Errorf("path = %q, want shortLink passthrough", gotPath)
	}
	if gotText != "hello" {
		t.Errorf("text = %q, want hello", gotText)
	}
	if !strings.Contains(stdout, "a1") || !strings.Contains(stdout, "hello") {
		t.Errorf("output missing comment data: %q", stdout)
	}
}

func TestCommentAddMissingText(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"comment", "add", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--text") {
		t.Errorf("stderr should mention --text: %q", stderr)
	}
}

func TestCommentList(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/actions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFilter = r.URL.Query().Get("filter")
		fmt.Fprint(w, `[
			{"id":"a1","type":"commentCard","date":"2020-03-09T19:41:51.396Z","data":{"text":"first"}},
			{"id":"a2","type":"commentCard","date":"2020-03-10T10:00:00.000Z","data":{"text":"second"}}
		]`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"comment", "list", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilter != "commentCard" {
		t.Errorf("filter = %q, want commentCard", gotFilter)
	}
	for _, want := range []string{"first", "second", "a1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestCommentUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"comment", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
