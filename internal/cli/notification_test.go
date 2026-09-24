package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testNotificationsFixture = `[
	{
		"id":"5dc591ac425f2a223aba0a8e",
		"type":"cardDueSoon",
		"date":"2019-11-08T16:02:52.763Z",
		"unread":true,
		"data":{"card":{"id":"c1","name":"Bowie"},"board":{"id":"b1","name":"Mullets"}}
	},
	{
		"id":"5dc591ac425f2a223aba0a8f",
		"type":"mentionedOnCard",
		"date":"2019-11-09T09:00:00.000Z",
		"unread":false,
		"data":{"card":{"id":"c2","name":"Release"},"board":{"id":"b1","name":"Mullets"},"text":"please review"}
	}
]`

func TestNotificationList(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/notifications" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFilter = r.URL.Query().Get("filter")
		fmt.Fprint(w, testNotificationsFixture)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"notification", "list", "--filter", "cardDueSoon"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilter != "cardDueSoon" {
		t.Errorf("filter = %q, want cardDueSoon", gotFilter)
	}
	for _, want := range []string{"cardDueSoon", "mentionedOnCard", "Bowie", "Release", "true", "false"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestNotificationListDefaultFilter(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	code, _, _ := runCLI(t, []string{"notification", "list"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilter != "all" {
		t.Errorf("filter = %q, want all (default)", gotFilter)
	}
}

func TestNotificationRead(t *testing.T) {
	var gotUnread string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notifications/n1" || r.Method != http.MethodPut {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotUnread = r.URL.Query().Get("unread")
		fmt.Fprint(w, `{"id":"n1","type":"cardDueSoon","unread":false,"data":{"card":{"id":"c1","name":"Bowie"}}}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"notification", "read", "n1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotUnread != "false" {
		t.Errorf("unread = %q, want false", gotUnread)
	}
	if !strings.Contains(stdout, "n1") || !strings.Contains(stdout, "false") {
		t.Errorf("output missing notification data: %q", stdout)
	}
}

func TestNotificationReadAll(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"notification", "read-all"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotPath != "/notifications/all/read" {
		t.Errorf("path = %q, want /notifications/all/read", gotPath)
	}
	if !strings.Contains(stdout, "Marked all notifications as read") {
		t.Errorf("output should confirm: %q", stdout)
	}
}

func TestNotificationReadNoArgs(t *testing.T) {
	code, _, _ := runCLI(t, []string{"notification", "read"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestNotificationUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"notification", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
