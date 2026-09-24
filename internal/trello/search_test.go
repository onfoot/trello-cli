package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	var gotQuery, gotModelTypes, gotIDBoards string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotQuery = r.URL.Query().Get("query")
		gotModelTypes = r.URL.Query().Get("modelTypes")
		gotIDBoards = r.URL.Query().Get("idBoards")
		fmt.Fprint(w, `{
			"boards":[{"id":"b1","name":"Trello Platform Changes","shortLink":"3CsPkqOF"}],
			"cards":[{"id":"c1","name":"Ship it","idList":"l1","idBoard":"b1","closed":false}],
			"members":[{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}],
			"organizations":[{"id":"o1","name":"acme","displayName":"Acme Inc"}]
		}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	res, err := c.Search(context.Background(), "ship it", SearchOptions{
		ModelTypes: []string{"cards", "boards"},
		IDBoards:   "b1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Boards) != 1 || res.Boards[0].Name != "Trello Platform Changes" {
		t.Errorf("unexpected boards: %+v", res.Boards)
	}
	if len(res.Cards) != 1 || res.Cards[0].Name != "Ship it" {
		t.Errorf("unexpected cards: %+v", res.Cards)
	}
	if len(res.Members) != 1 || res.Members[0].Username != "bentleycook" {
		t.Errorf("unexpected members: %+v", res.Members)
	}
	if len(res.Organizations) != 1 || res.Organizations[0].DisplayName != "Acme Inc" {
		t.Errorf("unexpected organizations: %+v", res.Organizations)
	}
	if gotQuery != "ship it" || gotModelTypes != "cards,boards" || gotIDBoards != "b1" {
		t.Errorf("query params: query=%q modelTypes=%q idBoards=%q", gotQuery, gotModelTypes, gotIDBoards)
	}
}

func TestSearchNoModelTypes(t *testing.T) {
	var gotModelTypes string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModelTypes = r.URL.Query().Get("modelTypes")
		fmt.Fprint(w, `{"boards":[],"cards":[],"members":[],"organizations":[]}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if _, err := c.Search(context.Background(), "x", SearchOptions{}); err != nil {
		t.Fatal(err)
	}
	if gotModelTypes != "" {
		t.Errorf("modelTypes = %q, want unset when empty", gotModelTypes)
	}
}

func TestSearchResultNormalizesMissingKeys(t *testing.T) {
	// The real /search endpoint omits keys for model types that were not
	// requested; the decoded slices must still be non-nil empty slices so
	// marshaling emits arrays, never null.
	fixture := `{"cards":[{"id":"c1","name":"Ship it"}]}`
	var res SearchResult
	if err := json.Unmarshal([]byte(fixture), &res); err != nil {
		t.Fatal(err)
	}
	if res.Boards == nil {
		t.Error("Boards should be a non-nil empty slice")
	}
	if res.Members == nil {
		t.Error("Members should be a non-nil empty slice")
	}
	if res.Organizations == nil {
		t.Error("Organizations should be a non-nil empty slice")
	}
	if res.Cards == nil || len(res.Cards) != 1 || res.Cards[0].ID != "c1" {
		t.Errorf("Cards should hold the decoded card, got %+v", res.Cards)
	}

	out, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "null") {
		t.Errorf("marshaled output must not contain null: %s", out)
	}
	for _, key := range []string{`"boards":[]`, `"cards":`, `"members":[]`, `"organizations":[]`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("marshaled output missing %s: %s", key, out)
		}
	}
}

func TestSearchResultNullGroupsNormalized(t *testing.T) {
	// Explicit null keys are normalized to empty slices too.
	fixture := `{"boards":null,"cards":null,"members":null,"organizations":null}`
	var res SearchResult
	if err := json.Unmarshal([]byte(fixture), &res); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "null") {
		t.Errorf("marshaled output must not contain null: %s", out)
	}
}
