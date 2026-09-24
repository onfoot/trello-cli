package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddComment(t *testing.T) {
	var gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/c1/actions/comments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotText = r.URL.Query().Get("text")
		fmt.Fprint(w, `{"id":"a1","idMemberCreator":"m1","type":"commentCard","date":"2020-03-09T19:41:51.396Z","data":{"text":"hello"}}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	comment, err := c.AddComment(context.Background(), "c1", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if comment.ID != "a1" || comment.Data.Text != "hello" {
		t.Errorf("unexpected comment: %+v", comment)
	}
	if gotText != "hello" {
		t.Errorf("text = %q, want hello", gotText)
	}
}

func TestAddCommentAcceptsShortLink(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"id":"a1","data":{"text":"hi"}}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if _, err := c.AddComment(context.Background(), "AbCdEf01", "hi"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/cards/AbCdEf01/actions/comments" {
		t.Errorf("path = %q, want shortLink passthrough", gotPath)
	}
}

func TestListComments(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
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

	c := New("k", "t", srv.URL, srv.Client())
	comments, err := c.ListComments(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 || comments[0].Data.Text != "first" || comments[1].Data.Text != "second" {
		t.Errorf("unexpected comments: %+v", comments)
	}
	if gotFilter != "commentCard" {
		t.Errorf("filter = %q, want commentCard", gotFilter)
	}
}
