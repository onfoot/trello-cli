package cli

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updateGolden regenerates the golden files when -update is passed.
var updateGolden = flag.Bool("update", false, "update golden files")

// goldenCompare compares got against testdata/<name>.golden, rewriting the
// golden file when -update is set.
func goldenCompare(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create it)", path, err)
	}
	if got != string(want) {
		t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

const whoamiFixture = `{
	"id": "5b02e7f4e1facdc393169f9d",
	"username": "bentleycook",
	"fullName": "Bentley Cook",
	"initials": "BC",
	"url": "https://trello.com/bentleycook",
	"avatarUrl": "https://trello-avatars.s3.amazonaws.com/fc8faaaee46666a4eb8b626c08933e16",
	"bio": "I'm a developer advocate at Trello!",
	"email": "bcook@atlassian.com",
	"confirmed": true
}`

const boardsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
	{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
]`

func goldenEnv(t *testing.T) Env {
	t.Helper()
	env := testEnv(t, "")
	writeConfig(t, filepath.Join(env.HomeDir, ".trello.yaml"), "board: my-board\nkey: proj-key\n")
	writeConfig(t, filepath.Join(env.HomeDir, ".config", "trello", "config.yaml"), "token: global-token\n")
	env.APIKey = ""
	env.Token = ""
	return env
}

func TestGoldenWhoamiHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, whoamiFixture)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, stderr := runCLI(t, []string{"whoami"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "whoami_human", stdout)
}

func TestGoldenWhoamiJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, whoamiFixture)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, stderr := runCLI(t, []string{"whoami", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "whoami_json", stdout)
}

func TestGoldenBoardListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, boardsFixture)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, stderr := runCLI(t, []string{"board", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "board_list_human", stdout)
}

func TestGoldenBoardListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, boardsFixture)
	}))
	defer srv.Close()

	env := testEnv(t, "")
	env.APIKey = "k"
	env.Token = "t"
	code, stdout, stderr := runCLI(t, []string{"board", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "board_list_json", stdout)
}

func TestGoldenConfigShowHuman(t *testing.T) {
	env := goldenEnv(t)
	code, stdout, stderr := runCLI(t, []string{"config", "show"}, strings.NewReader(""), env, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "config_show_human", normalizePaths(stdout, env.HomeDir))
}

func TestGoldenConfigShowJSON(t *testing.T) {
	env := goldenEnv(t)
	code, stdout, stderr := runCLI(t, []string{"config", "show", "--json"}, strings.NewReader(""), env, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "config_show_json", normalizePaths(stdout, env.HomeDir))
}

// normalizePaths replaces the temp home directory with a stable placeholder so
// golden files do not depend on the test machine's temp path.
func normalizePaths(s, home string) string {
	return strings.ReplaceAll(s, home, "<HOME>")
}

const goldenListsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961415","name":"To do","closed":false,"pos":16384,"idBoard":"5abbe4b7ddc1b351ef961414"},
	{"id":"5abbe4b7ddc1b351ef961416","name":"Done","closed":false,"pos":32768,"idBoard":"5abbe4b7ddc1b351ef961414"}
]`

const goldenCardsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961417","name":"Ship it","idList":"5abbe4b7ddc1b351ef961415","idBoard":"5abbe4b7ddc1b351ef961414","closed":false,"shortLink":"AbCdEf01","pos":65535,"url":"https://trello.com/c/AbCdEf01/ship-it"},
	{"id":"5abbe4b7ddc1b351ef961418","name":"Fix bug","idList":"5abbe4b7ddc1b351ef961416","idBoard":"5abbe4b7ddc1b351ef961414","closed":false,"due":"2026-09-04T12:00:00.000Z","shortLink":"AbCdEf02","pos":131071,"url":"https://trello.com/c/AbCdEf02/fix-bug"}
]`

const goldenCardFixture = `{
	"id":"5abbe4b7ddc1b351ef961417",
	"name":"Ship it",
	"desc":"Ship the release",
	"idList":"5abbe4b7ddc1b351ef961415",
	"idBoard":"5abbe4b7ddc1b351ef961414",
	"closed":false,
	"due":"2026-09-04T12:00:00.000Z",
	"shortLink":"AbCdEf01",
	"pos":65535,
	"url":"https://trello.com/c/AbCdEf01/ship-it"
}`

func TestGoldenListListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/lists": goldenListsFixture,
	})
	defer srv.Close()

	code, stdout, stderr := runCLI(t, []string{"list", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "list_list_human", stdout)
}

func TestGoldenListListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/lists": goldenListsFixture,
	})
	defer srv.Close()

	code, stdout, stderr := runCLI(t, []string{"list", "list", "--json"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "list_list_json", stdout)
}

func TestGoldenCardListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/cards": goldenCardsFixture,
	})
	defer srv.Close()

	code, stdout, stderr := runCLI(t, []string{"card", "list"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "card_list_human", stdout)
}

func TestGoldenCardListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                     testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/cards": goldenCardsFixture,
	})
	defer srv.Close()

	code, stdout, stderr := runCLI(t, []string{"card", "list", "--json"}, strings.NewReader(""), boardEnv(t), srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "card_list_json", stdout)
}

func TestGoldenCardGetHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenCardFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"card", "get", "5abbe4b7ddc1b351ef961417"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "card_get_human", stdout)
}

func TestGoldenCardGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenCardFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"card", "get", "5abbe4b7ddc1b351ef961417", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "card_get_json", stdout)
}

const goldenCommentsFixture = `[
	{
		"id": "5dc9b507756e182c76007621",
		"idMemberCreator": "5b02e7f4e1facdc393169f9d",
		"type": "commentCard",
		"date": "2020-03-09T19:41:51.396Z",
		"data": {
			"text": "Can never go wrong with bowie",
			"card": {"id": "5abbe4b7ddc1b351ef961417", "name": "Bowie", "idShort": 7, "shortLink": "AbCdEf01"},
			"board": {"id": "5abbe4b7ddc1b351ef961414", "name": "Mullets", "shortLink": "3CsPkqOF"}
		}
	},
	{
		"id": "5dc9b507756e182c76007622",
		"idMemberCreator": "5b02e7f4e1facdc393169f9d",
		"type": "commentCard",
		"date": "2020-03-10T10:00:00.000Z",
		"data": {
			"text": "Ship it",
			"card": {"id": "5abbe4b7ddc1b351ef961417", "name": "Bowie", "idShort": 7, "shortLink": "AbCdEf01"},
			"board": {"id": "5abbe4b7ddc1b351ef961414", "name": "Mullets", "shortLink": "3CsPkqOF"}
		}
	}
]`

const goldenChecklistsFixture = `[
	{
		"id": "5dc9b507756e182c76007621",
		"name": "Release checklist",
		"idBoard": "5abbe4b7ddc1b351ef961414",
		"checkItems": [
			{"id": "5dc9b509f02f4314edc4303a", "idChecklist": "5dc9b507756e182c76007621", "name": "Update docs", "state": "complete", "pos": 1673},
			{"id": "5dc9b509f02f4314edc4303b", "idChecklist": "5dc9b507756e182c76007621", "name": "Tag release", "state": "incomplete", "pos": 32767}
		]
	},
	{
		"id": "5dc9b507756e182c76007622",
		"name": "QA checklist",
		"idBoard": "5abbe4b7ddc1b351ef961414",
		"checkItems": []
	}
]`

const goldenSearchFixture = `{
	"boards": [
		{"id": "5abbe4b7ddc1b351ef961414", "name": "Trello Platform Changes", "shortLink": "3CsPkqOF"}
	],
	"cards": [
		{"id": "5abbe4b7ddc1b351ef961417", "name": "Ship it", "idList": "5abbe4b7ddc1b351ef961415", "idBoard": "5abbe4b7ddc1b351ef961414", "closed": false, "shortLink": "AbCdEf01", "pos": 65535}
	],
	"members": [
		{"id": "5b02e7f4e1facdc393169f9d", "username": "bentleycook", "fullName": "Bentley Cook", "initials": "BC", "confirmed": true}
	],
	"organizations": []
}`

// goldenSearchMixedFixture has empty boards and organizations but non-empty
// cards and members, exercising the "(no results)" rendering.
const goldenSearchMixedFixture = `{
	"boards": [],
	"cards": [
		{"id": "5abbe4b7ddc1b351ef961417", "name": "Ship it", "idList": "5abbe4b7ddc1b351ef961415", "idBoard": "5abbe4b7ddc1b351ef961414", "closed": false, "shortLink": "AbCdEf01", "pos": 65535}
	],
	"members": [
		{"id": "5b02e7f4e1facdc393169f9d", "username": "bentleycook", "fullName": "Bentley Cook", "initials": "BC", "confirmed": true}
	],
	"organizations": []
}`

// goldenSearchCardsOnlyFixture omits the boards, members, and organizations
// keys entirely — like the real /search endpoint does for unrequested model
// types. The decoded groups must still marshal as empty arrays, not null.
const goldenSearchCardsOnlyFixture = `{
	"cards": [
		{"id": "5abbe4b7ddc1b351ef961417", "name": "Ship it", "idList": "5abbe4b7ddc1b351ef961415", "idBoard": "5abbe4b7ddc1b351ef961414", "closed": false, "shortLink": "AbCdEf01", "pos": 65535}
	]
}`

// TestGoldenCoverage asserts that every Phase-1 domain has both a human and a
// --json golden file, guarding against accidental schema drift.
func TestGoldenCoverage(t *testing.T) {
	required := []string{
		// boards
		"board_list_human", "board_list_json",
		// lists
		"list_list_human", "list_list_json",
		// cards
		"card_list_human", "card_list_json",
		// comments
		"comment_list_human", "comment_list_json",
		// checklists
		"checklist_list_human", "checklist_list_json",
		// search
		"search_human", "search_json",
		// labels
		"label_list_human", "label_list_json",
		// members
		"member_list_human", "member_list_json",
		// attachments
		"attachment_list_human", "attachment_list_json",
		// custom fields
		"customfield_list_human", "customfield_list_json",
		// actions
		"action_list_human", "action_list_json",
		// notifications
		"notification_list_human", "notification_list_json",
		// organizations
		"org_list_human", "org_list_json",
		"org_boards_human", "org_boards_json",
		// config show
		"config_show_human", "config_show_json",
	}
	for _, name := range required {
		path := filepath.Join("testdata", name+".golden")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing golden file %s", path)
		}
	}
}

func TestGoldenCommentListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenCommentsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"comment", "list", "5abbe4b7ddc1b351ef961417"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "comment_list_human", stdout)
}

func TestGoldenCommentListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenCommentsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"comment", "list", "5abbe4b7ddc1b351ef961417", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "comment_list_json", stdout)
}

func TestGoldenChecklistListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenChecklistsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"checklist", "list", "5abbe4b7ddc1b351ef961417"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "checklist_list_human", stdout)
}

func TestGoldenChecklistListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenChecklistsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"checklist", "list", "5abbe4b7ddc1b351ef961417", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "checklist_list_json", stdout)
}

func TestGoldenSearchHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "bowie"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_human", stdout)
}

func TestGoldenSearchJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "bowie", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_json", stdout)
}

// An empty requested group still shows its header plus a "(no results)"
// indicator in human mode.
func TestGoldenSearchEmptyBoardsHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchMixedFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "x", "--boards"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_empty_boards_human", stdout)
}

// Mixed results: an empty group and a non-empty group both render.
func TestGoldenSearchMixedHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchMixedFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "x", "--cards", "--boards"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_mixed_human", stdout)
}

// --json keeps structured empty arrays for empty groups.
func TestGoldenSearchEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchMixedFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "x", "--boards", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_empty_json", stdout)
}

// --json must emit all four group keys as arrays even when the API omits the
// keys for unrequested model types (never null).
func TestGoldenSearchCardsOnlyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenSearchCardsOnlyFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"search", "x", "--cards", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "search_cards_only_json", stdout)
}

const goldenLabelsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961419","name":"Overdue","color":"red","idBoard":"5abbe4b7ddc1b351ef961414"},
	{"id":"5abbe4b7ddc1b351ef96141a","name":"Needs review","color":"blue","idBoard":"5abbe4b7ddc1b351ef961414"}
]`

const goldenMembersFixture = `[
	{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true},
	{"id":"5b02e7f4e1facdc393169f9e","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
]`

func TestGoldenLabelListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                      testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/labels": goldenLabelsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"label", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "label_list_human", stdout)
}

func TestGoldenLabelListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                      testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/labels": goldenLabelsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"label", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "label_list_json", stdout)
}

func TestGoldenMemberListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                       testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/members": goldenMembersFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"member", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "member_list_human", stdout)
}

func TestGoldenMemberListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                       testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/members": goldenMembersFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"member", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "member_list_json", stdout)
}

const goldenAttachmentsFixture = `[
	{"id":"5bc79d4206526d2279c1e6ea","name":"Notice","mimeType":"application/pdf","bytes":52845,"url":"https://example.com/notice.pdf","isUpload":false},
	{"id":"5bc79d4206526d2279c1e6eb","name":"Image","mimeType":"image/png","bytes":1234,"url":"https://example.com/img.png","isUpload":true}
]`

const goldenCustomFieldsFixture = `[
	{"id":"5ab10be237846c43015f108e","name":"Priority","type":"list","idModel":"5abbe4b7ddc1b351ef961414"},
	{"id":"5ab10be237846c43015f108f","name":"ETA","type":"date","idModel":"5abbe4b7ddc1b351ef961414"}
]`

func TestGoldenAttachmentListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenAttachmentsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"attachment", "list", "5abbe4b7ddc1b351ef961417"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "attachment_list_human", stdout)
}

func TestGoldenAttachmentListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenAttachmentsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"attachment", "list", "5abbe4b7ddc1b351ef961417", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "attachment_list_json", stdout)
}

func TestGoldenCustomFieldListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                            testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/customFields": goldenCustomFieldsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"customfield", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "customfield_list_human", stdout)
}

func TestGoldenCustomFieldListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                            testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/customFields": goldenCustomFieldsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"customfield", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "customfield_list_json", stdout)
}

const goldenActionsFixture = `[
	{
		"id":"5dc9b507756e182c76007621",
		"idMemberCreator":"5b02e7f4e1facdc393169f9d",
		"type":"commentCard",
		"date":"2020-03-09T19:41:51.396Z",
		"data":{"text":"Can never go wrong with bowie","card":{"id":"5abbe4b7ddc1b351ef961417","name":"Bowie","idShort":7,"shortLink":"AbCdEf01"},"board":{"id":"5abbe4b7ddc1b351ef961414","name":"Mullets","shortLink":"3CsPkqOF"}},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}
	},
	{
		"id":"5dc9b507756e182c76007622",
		"idMemberCreator":"5b02e7f4e1facdc393169f9e",
		"type":"updateCard",
		"date":"2020-03-10T10:00:00.000Z",
		"data":{"card":{"id":"5abbe4b7ddc1b351ef961417","name":"Bowie"},"list":{"id":"5abbe4b7ddc1b351ef961415","name":"Amazing"},"board":{"id":"5abbe4b7ddc1b351ef961414","name":"Mullets"}},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9e","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
	}
]`

const goldenNotificationsFixture = `[
	{
		"id":"5dc591ac425f2a223aba0a8e",
		"type":"cardDueSoon",
		"date":"2019-11-08T16:02:52.763Z",
		"unread":true,
		"data":{"card":{"id":"5abbe4b7ddc1b351ef961417","name":"Bowie"},"board":{"id":"5abbe4b7ddc1b351ef961414","name":"Mullets"}}
	},
	{
		"id":"5dc591ac425f2a223aba0a8f",
		"type":"mentionedOnCard",
		"date":"2019-11-09T09:00:00.000Z",
		"unread":false,
		"data":{"card":{"id":"5abbe4b7ddc1b351ef961418","name":"Release"},"board":{"id":"5abbe4b7ddc1b351ef961414","name":"Mullets"},"text":"please review"}
	}
]`

func TestGoldenActionListHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                       testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/actions": goldenActionsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"action", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "action_list_human", stdout)
}

func TestGoldenActionListJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/boards":                       testBoardsFixture,
		"/boards/5abbe4b7ddc1b351ef961414/actions": goldenActionsFixture,
	})
	defer srv.Close()

	env := boardEnv(t)
	code, stdout, stderr := runCLI(t, []string{"action", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "action_list_json", stdout)
}

func TestGoldenNotificationListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenNotificationsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"notification", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "notification_list_human", stdout)
}

func TestGoldenNotificationListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenNotificationsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"notification", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "notification_list_json", stdout)
}

const goldenOrgsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961414","name":"acme","displayName":"Acme Inc","desc":"The Acme team","url":"https://trello.com/acme","website":"https://acme.example.com"},
	{"id":"5abbe4b7ddc1b351ef961415","name":"globex","displayName":"Globex","desc":"","url":"https://trello.com/globex","website":""}
]`

const goldenOrgBoardsFixture = `[
	{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF","closed":false},
	{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12","closed":true}
]`

func TestGoldenOrgListHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenOrgsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"org", "list"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "org_list_human", stdout)
}

func TestGoldenOrgListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, goldenOrgsFixture)
	}))
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"org", "list", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "org_list_json", stdout)
}

func TestGoldenOrgBoardsHuman(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/organizations":                      goldenOrgsFixture,
		"/organizations/5abbe4b7ddc1b351ef961414/boards": goldenOrgBoardsFixture,
	})
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"org", "boards", "acme"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "org_boards_human", stdout)
}

func TestGoldenOrgBoardsJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/members/me/organizations":                      goldenOrgsFixture,
		"/organizations/5abbe4b7ddc1b351ef961414/boards": goldenOrgBoardsFixture,
	})
	defer srv.Close()

	env := credsEnv(t)
	code, stdout, stderr := runCLI(t, []string{"org", "boards", "acme", "--json"}, strings.NewReader(""), env, srv.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr: %s", code, stderr)
	}
	goldenCompare(t, "org_boards_json", stdout)
}
