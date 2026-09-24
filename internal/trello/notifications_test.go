package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const notificationsFixture = `[
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

func TestListMyNotifications(t *testing.T) {
	var gotFilter, gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/members/me/notifications" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFilter = r.URL.Query().Get("filter")
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, notificationsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	notifs, err := c.ListMyNotifications(context.Background(), "cardDueSoon")
	if err != nil {
		t.Fatal(err)
	}
	if len(notifs) != 2 || notifs[0].Type != "cardDueSoon" || notifs[1].Unread {
		t.Errorf("unexpected notifications: %+v", notifs)
	}
	if notifs[0].Data.Card == nil || notifs[0].Data.Card.Name != "Bowie" {
		t.Errorf("unexpected data card: %+v", notifs[0].Data.Card)
	}
	if notifs[0].Date == nil || notifs[0].Date.IsZero() {
		t.Error("date should be parsed")
	}
	if gotFilter != "cardDueSoon" {
		t.Errorf("filter = %q", gotFilter)
	}
	if gotFields != "id,type,date,unread,data" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestSetNotificationRead(t *testing.T) {
	var gotUnread string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/notifications/n1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotUnread = r.URL.Query().Get("unread")
		fmt.Fprint(w, `{"id":"n1","type":"cardDueSoon","unread":false,"data":{"card":{"id":"c1","name":"Bowie"}}}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	n, err := c.SetNotificationRead(context.Background(), "n1", false)
	if err != nil {
		t.Fatal(err)
	}
	if n.Unread {
		t.Error("expected unread=false")
	}
	if gotUnread != "false" {
		t.Errorf("unread = %q, want false", gotUnread)
	}
}

func TestMarkAllNotificationsRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/notifications/all/read" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.MarkAllNotificationsRead(context.Background()); err != nil {
		t.Fatal(err)
	}
}
