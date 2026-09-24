package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestChecklistList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/checklists" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `[
			{"id":"cl1","name":"Release","idBoard":"b1","checkItems":[
				{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673}
			]}
		]`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"checklist", "list", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Release", "cl1", "1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestChecklistGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checklists/cl1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1","checkItems":[
			{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673},
			{"id":"ci2","idChecklist":"cl1","name":"Tag release","state":"incomplete","pos":32767}
		]}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"checklist", "get", "cl1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Release", "Update docs", "Tag release", "complete", "incomplete"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestChecklistCreate(t *testing.T) {
	var gotCard, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checklists" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotCard = r.URL.Query().Get("idCard")
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"checklist", "create", "AbCdEf01", "--name", "Release"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotCard != "AbCdEf01" || gotName != "Release" {
		t.Errorf("query params: idCard=%q name=%q", gotCard, gotName)
	}
	if !strings.Contains(stdout, "cl1") {
		t.Errorf("output should include the new checklist id: %q", stdout)
	}
}

func TestChecklistCreateMissingName(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"checklist", "create", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--name") {
		t.Errorf("stderr should mention --name: %q", stderr)
	}
}

func TestChecklistAddItem(t *testing.T) {
	var gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checklists/cl1/checkItems" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"incomplete","pos":1673}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"checklist", "add-item", "cl1", "--name", "Update docs"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "Update docs" {
		t.Errorf("name = %q, want Update docs", gotName)
	}
	if !strings.Contains(stdout, "ci1") {
		t.Errorf("output should include the new item id: %q", stdout)
	}
}

func TestChecklistAddItemMissingName(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"checklist", "add-item", "cl1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--name") {
		t.Errorf("stderr should mention --name: %q", stderr)
	}
}

func TestChecklistCheckItemDefaultComplete(t *testing.T) {
	var gotState, gotPutPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checklists/cl1":
			// Fetch the checklist to learn the card id.
			fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1","idCard":"c1"}`)
		case "/cards/c1/checkItem/ci1":
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			gotPutPath = r.URL.Path
			gotState = r.URL.Query().Get("state")
			fmt.Fprint(w, `{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"checklist", "check-item", "cl1", "ci1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPutPath != "/cards/c1/checkItem/ci1" {
		t.Errorf("put path = %q, want the card-scoped endpoint", gotPutPath)
	}
	if gotState != "complete" {
		t.Errorf("state = %q, want complete (default)", gotState)
	}
}

func TestChecklistCheckItemStateFlag(t *testing.T) {
	var gotState, gotPutPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checklists/cl1":
			fmt.Fprint(w, `{"id":"cl1","name":"Release","idCard":"c1"}`)
		case "/cards/c1/checkItem/ci1":
			gotPutPath = r.URL.Path
			gotState = r.URL.Query().Get("state")
			fmt.Fprint(w, `{"id":"ci1","state":"incomplete"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"checklist", "check-item", "cl1", "ci1", "--state", "incomplete"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPutPath != "/cards/c1/checkItem/ci1" {
		t.Errorf("put path = %q, want the card-scoped endpoint", gotPutPath)
	}
	if gotState != "incomplete" {
		t.Errorf("state = %q, want incomplete", gotState)
	}
}

func TestChecklistCheckItemInvalidState(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"checklist", "check-item", "cl1", "ci1", "--state", "bogus"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "complete or incomplete") {
		t.Errorf("stderr should explain valid states: %q", stderr)
	}
}

func TestChecklistUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"checklist", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
