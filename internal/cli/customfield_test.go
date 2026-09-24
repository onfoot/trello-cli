package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testCustomFieldsFixture = `[
	{"id":"cf1","name":"Priority","type":"list","idModel":"b1","options":[
		{"id":"opt1","value":{"text":"High"}},
		{"id":"opt2","value":{"text":"Low"}}
	]},
	{"id":"cf2","name":"Score","type":"number","idModel":"b1"},
	{"id":"cf3","name":"ETA","type":"date","idModel":"b1"},
	{"id":"cf4","name":"Done","type":"checkbox","idModel":"b1"},
	{"id":"cf5","name":"Notes","type":"text","idModel":"b1"}
]`

// customFieldSetServer serves the requests made by `customfield set` and
// captures the PUT item body into gotBody.
func customFieldSetServer(t *testing.T, gotBody *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/cards/c1/board":
			fmt.Fprint(w, `{"id":"b1"}`)
		case r.URL.Path == "/boards/b1/customFields":
			fmt.Fprint(w, testCustomFieldsFixture)
		case r.URL.Path == "/customFields/cf1":
			fmt.Fprint(w, `{"id":"cf1","name":"Priority","type":"list","idModel":"b1","options":[{"id":"opt1","value":{"text":"High"}},{"id":"opt2","value":{"text":"Low"}}]}`)
		case strings.HasPrefix(r.URL.Path, "/cards/c1/customField/"):
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			buf := make([]byte, 4096)
			n, _ := r.Body.Read(buf)
			*gotBody = string(buf[:n])
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestCustomFieldList(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                            testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/customFields": testCustomFieldsFixture,
	})
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"customfield", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Priority", "Score", "list", "number"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestCustomFieldGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customFields/cf1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"cf1","name":"Priority","type":"list","idModel":"b1","options":[{"id":"opt1","value":{"text":"High"}}]}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"customfield", "get", "cf1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Priority", "list", "High"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestCustomFieldSetText(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"customfield", "set", "c1", "Notes", "hello world"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"text":"hello world"`) {
		t.Errorf("body = %q, want the text payload", gotBody)
	}
	if !strings.Contains(stdout, "Set custom field") {
		t.Errorf("output should confirm the set: %q", stdout)
	}
}

func TestCustomFieldSetNumber(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"customfield", "set", "c1", "Score", "42.5"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"number":42.5`) {
		t.Errorf("body = %q, want the number payload", gotBody)
	}
}

func TestCustomFieldSetNumberInvalid(t *testing.T) {
	srv := customFieldSetServer(t, new(string))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"customfield", "set", "c1", "Score", "abc"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "must be a number") {
		t.Errorf("stderr should explain the number requirement: %q", stderr)
	}
}

func TestCustomFieldSetDate(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"customfield", "set", "c1", "ETA", "2026-09-04T12:00:00Z"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"date":"2026-09-04T12:00:00Z"`) {
		t.Errorf("body = %q, want the date payload", gotBody)
	}
}

func TestCustomFieldSetDateInvalid(t *testing.T) {
	srv := customFieldSetServer(t, new(string))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"customfield", "set", "c1", "ETA", "tomorrow"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "RFC3339") {
		t.Errorf("stderr should mention RFC3339: %q", stderr)
	}
}

func TestCustomFieldSetCheckbox(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"customfield", "set", "c1", "Done", "true"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"checked":true`) {
		t.Errorf("body = %q, want the checkbox payload", gotBody)
	}
}

func TestCustomFieldSetCheckboxInvalid(t *testing.T) {
	srv := customFieldSetServer(t, new(string))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"customfield", "set", "c1", "Done", "maybe"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "true or false") {
		t.Errorf("stderr should explain the checkbox values: %q", stderr)
	}
}

func TestCustomFieldSetListOption(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"customfield", "set", "c1", "Priority", "High"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"idValue":"opt1"`) {
		t.Errorf("body = %q, want the option id payload", gotBody)
	}
}

func TestCustomFieldSetListOptionByID(t *testing.T) {
	var gotBody string
	srv := customFieldSetServer(t, &gotBody)
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"customfield", "set", "c1", "cf1", "opt2"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(gotBody, `"idValue":"opt2"`) {
		t.Errorf("body = %q, want the option id payload", gotBody)
	}
}

func TestCustomFieldSetListOptionNoMatch(t *testing.T) {
	srv := customFieldSetServer(t, new(string))
	defer srv.Close()

	code, _, stderr := runCLI(t, []string{"customfield", "set", "c1", "Priority", "bogus"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitConfig {
		t.Errorf("exit = %d, want %d", code, output.ExitConfig)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr should mention the option ref: %q", stderr)
	}
}

func TestCustomFieldCreate(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/members/me/boards":
			fmt.Fprint(w, testBoardsFixture)
		case "/customFields":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			buf := make([]byte, 4096)
			n, _ := r.Body.Read(buf)
			gotBody = string(buf[:n])
			fmt.Fprint(w, `{"id":"cf1","name":"Priority","type":"list","idModel":"5abbe4b7ddc1b351ef961414"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"customfield", "create", "Priority", "--type", "list", "--options", "High, Medium, Low"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{`"idModel":"5abbe4b7ddc1b351ef961414"`, `"modelType":"board"`, `"name":"Priority"`, `"type":"list"`, `"text":"High"`, `"text":"Medium"`, `"text":"Low"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body missing %s: %q", want, gotBody)
		}
	}
	if !strings.Contains(stdout, "cf1") {
		t.Errorf("output should include the new field id: %q", stdout)
	}
}

func TestCustomFieldCreateInvalidType(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"customfield", "create", "X", "--type", "dropdown"}, strings.NewReader(""), boardEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "checkbox, list, number, text, date") {
		t.Errorf("stderr should list valid types: %q", stderr)
	}
}

func TestCustomFieldCreateMissingType(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"customfield", "create", "X"}, strings.NewReader(""), boardEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--type") {
		t.Errorf("stderr should mention --type: %q", stderr)
	}
}

func TestCustomFieldCreateOptionsOnNonList(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"customfield", "create", "X", "--type", "text", "--options", "a,b"}, strings.NewReader(""), boardEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "only valid for type=list") {
		t.Errorf("stderr should explain the restriction: %q", stderr)
	}
}

func TestCustomFieldDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customFields/cf1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"customfield", "delete", "cf1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "cf1") {
		t.Errorf("output should mention the deleted field: %q", stdout)
	}
}

func TestCustomFieldUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"customfield", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
