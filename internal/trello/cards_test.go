package trello

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListBoardCards(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/b1/cards" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"c1","name":"Ship it","idList":"l1","idBoard":"b1","closed":false},
			{"id":"c2","name":"Fix bug","idList":"l2","idBoard":"b1","closed":false}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	cards, err := c.ListBoardCards(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 || cards[0].Name != "Ship it" || cards[1].IDList != "l2" {
		t.Errorf("unexpected cards: %+v", cards)
	}
	if gotFields != "id,name,desc,idList,idBoard,closed,due,shortLink,pos,url,labels" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestListListCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/l1/cards" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `[{"id":"c1","name":"Ship it","idList":"l1"}]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	cards, err := c.ListListCards(context.Background(), "l1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].ID != "c1" {
		t.Errorf("unexpected cards: %+v", cards)
	}
}

func TestGetCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","desc":"details","idList":"l1","idBoard":"b1","closed":false,"due":"2026-09-04T12:00:00Z","shortLink":"AbCdEf01","pos":65535,"url":"https://trello.com/c/AbCdEf01"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	card, err := c.GetCard(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if card.ID != "c1" || card.Name != "Ship it" || card.Desc != "details" {
		t.Errorf("unexpected card: %+v", card)
	}
	if card.Due == nil || card.Due.Format("2006-01-02T15:04:05Z07:00") != "2026-09-04T12:00:00Z" {
		t.Errorf("unexpected due: %v", card.Due)
	}
}

func TestGetCardDecodesLabels(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","labels":[
			{"id":"lbl1","idBoard":"b1","name":"Overdue","color":"red"},
			{"id":"lbl2","idBoard":"b1","name":"","color":"blue"}
		]}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	card, err := c.GetCard(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotFields, "labels") {
		t.Errorf("fields = %q, want it to request labels", gotFields)
	}
	if len(card.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d: %+v", len(card.Labels), card.Labels)
	}
	if card.Labels[0].ID != "lbl1" || card.Labels[0].Name != "Overdue" || card.Labels[0].Color != "red" || card.Labels[0].IDBoard != "b1" {
		t.Errorf("unexpected first label: %+v", card.Labels[0])
	}
	if card.Labels[1].Name != "" || card.Labels[1].Color != "blue" {
		t.Errorf("unexpected second label: %+v", card.Labels[1])
	}
}

func TestCreateCard(t *testing.T) {
	var gotName, gotList, gotDesc, gotPos, gotDue, gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		gotName, gotList, gotDesc, gotPos, gotDue = q.Get("name"), q.Get("idList"), q.Get("desc"), q.Get("pos"), q.Get("due")
		gotFields = q.Get("fields")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"l1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	pos := 65535.0
	due := "2026-09-04T12:00:00Z"
	card, err := c.CreateCard(context.Background(), CardCreate{IDList: "l1", Name: "Ship it", Desc: "details", Pos: &pos, Due: &due})
	if err != nil {
		t.Fatal(err)
	}
	if card.ID != "c1" {
		t.Errorf("unexpected card: %+v", card)
	}
	if gotName != "Ship it" || gotList != "l1" || gotDesc != "details" || gotPos != "65535" || gotDue != "2026-09-04T12:00:00Z" {
		t.Errorf("query params: name=%q list=%q desc=%q pos=%q due=%q", gotName, gotList, gotDesc, gotPos, gotDue)
	}
	if !strings.Contains(gotFields, "labels") {
		t.Errorf("create fields = %q, want it to request labels", gotFields)
	}
}

func TestUpdateCard(t *testing.T) {
	var gotName, gotDesc, gotClosed, gotPos, gotDue, gotList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		gotName, gotDesc, gotClosed, gotPos, gotDue, gotList = q.Get("name"), q.Get("desc"), q.Get("closed"), q.Get("pos"), q.Get("due"), q.Get("idList")
		fmt.Fprint(w, `{"id":"c1","name":"New","closed":true}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	closed := true
	pos := 100.5
	due := "2026-09-04T12:00:00Z"
	card, err := c.UpdateCard(context.Background(), "c1", CardUpdate{
		Name:   strPtr("New"),
		Desc:   strPtr("new desc"),
		Closed: &closed,
		Due:    &due,
		Pos:    &pos,
		IDList: strPtr("l2"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "New" || !card.Closed {
		t.Errorf("unexpected card: %+v", card)
	}
	if gotName != "New" || gotDesc != "new desc" || gotClosed != "true" || gotPos != "100.5" || gotDue != "2026-09-04T12:00:00Z" || gotList != "l2" {
		t.Errorf("query params: name=%q desc=%q closed=%q pos=%q due=%q list=%q", gotName, gotDesc, gotClosed, gotPos, gotDue, gotList)
	}
}

func TestMoveCard(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/cards/c1/idList" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotValue = r.URL.Query().Get("value")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","idList":"l2"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	card, err := c.MoveCard(context.Background(), "c1", "l2")
	if err != nil {
		t.Fatal(err)
	}
	if card.IDList != "l2" {
		t.Errorf("unexpected card: %+v", card)
	}
	if gotValue != "l2" {
		t.Errorf("value = %q, want l2", gotValue)
	}
}

func TestArchiveCard(t *testing.T) {
	var gotClosed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotClosed = r.URL.Query().Get("closed")
		fmt.Fprint(w, `{"id":"c1","name":"Ship it","closed":true}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	card, err := c.ArchiveCard(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if !card.Closed {
		t.Error("expected closed card")
	}
	if gotClosed != "true" {
		t.Errorf("closed = %q, want true", gotClosed)
	}
}

func TestDeleteCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/cards/c1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.DeleteCard(context.Background(), "c1"); err != nil {
		t.Fatal(err)
	}
}

func TestCardNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such card"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.GetCard(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
	var ne *NotFoundError
	if !errors.As(err, &ne) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}
