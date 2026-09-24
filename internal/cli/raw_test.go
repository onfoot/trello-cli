package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

func TestRawGet(t *testing.T) {
	var gotPath, gotKey, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		gotToken = r.URL.Query().Get("token")
		fmt.Fprint(w, `{"id":"m1","username":"bentleycook"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"raw", "GET", "/members/me"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/members/me" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "k" || gotToken != "t" {
		t.Errorf("key/token injection: key=%q token=%q", gotKey, gotToken)
	}
	// Human mode pretty-prints the JSON.
	if !strings.Contains(stdout, "\"id\": \"m1\"") || !strings.Contains(stdout, "\n") {
		t.Errorf("expected pretty-printed JSON, got: %q", stdout)
	}
}

func TestRawJSONMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"m1"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"raw", "GET", "/members/me", "--json"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if stdout != "{\"id\":\"m1\"}\n" {
		t.Errorf("raw JSON mode should emit the body verbatim, got: %q", stdout)
	}
}

func TestRawPostQueryParams(t *testing.T) {
	var gotName, gotList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotList = r.URL.Query().Get("idList")
		fmt.Fprint(w, `{"id":"c1","name":"New card"}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{
		"raw", "POST", "/cards",
		"--query", "name=New card",
		"-q", "idList=l1",
	}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotName != "New card" || gotList != "l1" {
		t.Errorf("query params: name=%q idList=%q", gotName, gotList)
	}
}

func TestRawLowercaseMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"raw", "get", "/members/me"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRawInvalidMethod(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"raw", "PATCH", "/members/me"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "GET, POST, PUT, DELETE") {
		t.Errorf("stderr should list valid methods: %q", stderr)
	}
}

func TestRawInvalidPath(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"raw", "GET", "members/me"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "start with /") {
		t.Errorf("stderr should explain the path rule: %q", stderr)
	}
}

func TestRawInvalidQuery(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"raw", "GET", "/members/me", "--query", "nokeyvalue"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "k=v") {
		t.Errorf("stderr should explain the k=v rule: %q", stderr)
	}
}

func TestRawErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such board"}`)
	}))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"raw", "GET", "/boards/missing"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitNotFound {
		t.Errorf("exit = %d, want %d", code, output.ExitNotFound)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should surface the status: %q", stderr)
	}
}

func TestRawEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"raw", "DELETE", "/cards/c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if stdout != "" {
		t.Errorf("expected empty output for empty body, got: %q", stdout)
	}
}
