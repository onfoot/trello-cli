package trello

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const actionsFixture = `[
	{
		"id":"5dc9b507756e182c76007621",
		"idMemberCreator":"5b02e7f4e1facdc393169f9d",
		"type":"commentCard",
		"date":"2020-03-09T19:41:51.396Z",
		"data":{
			"text":"Can never go wrong with bowie",
			"card":{"id":"c1","name":"Bowie","idShort":7,"shortLink":"3CsPkqOF"},
			"board":{"id":"b1","name":"Mullets","shortLink":"3CsPkqOF"}
		},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}
	},
	{
		"id":"5dc9b507756e182c76007622",
		"idMemberCreator":"5b02e7f4e1facdc393169f9e",
		"type":"updateCard",
		"date":"2020-03-10T10:00:00.000Z",
		"data":{
			"card":{"id":"c1","name":"Bowie"},
			"list":{"id":"l1","name":"Amazing"},
			"board":{"id":"b1","name":"Mullets"}
		},
		"memberCreator":{"id":"5b02e7f4e1facdc393169f9e","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
	}
]`

func TestListBoardActions(t *testing.T) {
	var gotFilter, gotFields, gotMemberCreator string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/b1/actions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFilter = r.URL.Query().Get("filter")
		gotFields = r.URL.Query().Get("fields")
		gotMemberCreator = r.URL.Query().Get("memberCreator")
		fmt.Fprint(w, actionsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	actions, err := c.ListBoardActions(context.Background(), "b1", "commentCard,updateCard")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[0].Type != "commentCard" || actions[1].Type != "updateCard" {
		t.Errorf("unexpected actions: %+v", actions)
	}
	if actions[0].MemberCreator == nil || actions[0].MemberCreator.Username != "bentleycook" {
		t.Errorf("unexpected member creator: %+v", actions[0].MemberCreator)
	}
	if actions[0].Date == nil || actions[0].Date.IsZero() {
		t.Error("date should be parsed")
	}
	if gotFilter != "commentCard,updateCard" {
		t.Errorf("filter = %q", gotFilter)
	}
	if gotFields != "id,type,date,idMemberCreator,data" {
		t.Errorf("fields = %q", gotFields)
	}
	if gotMemberCreator != "true" {
		t.Errorf("memberCreator = %q, want true", gotMemberCreator)
	}
}

func TestListCardActions(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/AbCdEf01/actions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFilter = r.URL.Query().Get("filter")
		fmt.Fprint(w, actionsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	actions, err := c.ListCardActions(context.Background(), "AbCdEf01", "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(actions))
	}
	if gotFilter != "all" {
		t.Errorf("filter = %q, want all", gotFilter)
	}
}

func TestGetAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"id":"a1",
			"idMemberCreator":"5b02e7f4e1facdc393169f9d",
			"type":"commentCard",
			"date":"2020-03-09T19:41:51.396Z",
			"data":{"text":"hello","card":{"id":"c1","name":"Bowie"},"board":{"id":"b1","name":"Mullets"}},
			"memberCreator":{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}
		}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	act, err := c.GetAction(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if act.ID != "a1" || act.Type != "commentCard" || act.Data.Text != "hello" {
		t.Errorf("unexpected action: %+v", act)
	}
	if act.MemberCreator == nil || act.MemberCreator.Username != "bentleycook" {
		t.Errorf("unexpected member creator: %+v", act.MemberCreator)
	}
	if act.Data.Card == nil || act.Data.Card.Name != "Bowie" {
		t.Errorf("unexpected card ref: %+v", act.Data.Card)
	}
}

func TestGetActionNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such action"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.GetAction(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
	var ne *NotFoundError
	if !errors.As(err, &ne) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
	if output.CodeFor(err) != output.ExitNotFound {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitNotFound)
	}
}
