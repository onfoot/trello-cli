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

func strPtr(s string) *string { return &s }

func TestListBoardLists(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/b1/lists" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"l1","name":"To do","closed":false,"pos":16384,"idBoard":"b1"},
			{"id":"l2","name":"Done","closed":false,"pos":32768,"idBoard":"b1"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	lists, err := c.ListBoardLists(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 || lists[0].Name != "To do" || lists[1].ID != "l2" {
		t.Errorf("unexpected lists: %+v", lists)
	}
	if gotFields != "id,name,closed,pos,idBoard" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestGetList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"l1","name":"To do","closed":false,"pos":16384,"idBoard":"b1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	l, err := c.GetList(context.Background(), "l1")
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "l1" || l.Name != "To do" || l.IDBoard != "b1" || l.Pos != 16384 {
		t.Errorf("unexpected list: %+v", l)
	}
}

func TestCreateList(t *testing.T) {
	var gotName, gotBoard, gotPos string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/lists" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotBoard = r.URL.Query().Get("idBoard")
		gotPos = r.URL.Query().Get("pos")
		fmt.Fprint(w, `{"id":"l1","name":"To do","idBoard":"b1","pos":65535}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	pos := 65535.0
	l, err := c.CreateList(context.Background(), "b1", "To do", &pos)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "l1" || l.Name != "To do" {
		t.Errorf("unexpected list: %+v", l)
	}
	if gotName != "To do" || gotBoard != "b1" || gotPos != "65535" {
		t.Errorf("query params: name=%q board=%q pos=%q", gotName, gotBoard, gotPos)
	}
}

func TestCreateListWithoutPos(t *testing.T) {
	var gotPos string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPos = r.URL.Query().Get("pos")
		fmt.Fprint(w, `{"id":"l1","name":"To do"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if _, err := c.CreateList(context.Background(), "b1", "To do", nil); err != nil {
		t.Fatal(err)
	}
	if gotPos != "" {
		t.Errorf("pos = %q, want unset", gotPos)
	}
}

func TestUpdateList(t *testing.T) {
	var gotName, gotClosed, gotPos string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/lists/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotClosed = r.URL.Query().Get("closed")
		gotPos = r.URL.Query().Get("pos")
		fmt.Fprint(w, `{"id":"l1","name":"New","closed":true,"pos":100.5}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	closed := true
	pos := 100.5
	l, err := c.UpdateList(context.Background(), "l1", ListUpdate{Name: strPtr("New"), Closed: &closed, Pos: &pos})
	if err != nil {
		t.Fatal(err)
	}
	if l.Name != "New" || !l.Closed {
		t.Errorf("unexpected list: %+v", l)
	}
	if gotName != "New" || gotClosed != "true" || gotPos != "100.5" {
		t.Errorf("query params: name=%q closed=%q pos=%q", gotName, gotClosed, gotPos)
	}
}

func TestArchiveList(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/lists/l1/closed" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotValue = r.URL.Query().Get("value")
		fmt.Fprint(w, `{"id":"l1","name":"Done","closed":true}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	l, err := c.ArchiveList(context.Background(), "l1")
	if err != nil {
		t.Fatal(err)
	}
	if !l.Closed {
		t.Error("expected closed list")
	}
	if gotValue != "true" {
		t.Errorf("value = %q, want true", gotValue)
	}
}

func TestResolveList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":"l1","name":"To do","closed":false,"pos":16384,"idBoard":"b1"},
			{"id":"l2","name":"Done","closed":false,"pos":32768,"idBoard":"b1"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string
	}{
		{"l1", "l1"},    // exact id
		{"Done", "l2"},  // exact name
		{"To do", "l1"}, // exact name with space
	}
	for _, tt := range tests {
		l, err := c.ResolveList(ctx, "b1", tt.ref)
		if err != nil {
			t.Errorf("ResolveList(%q) error: %v", tt.ref, err)
			continue
		}
		if l.ID != tt.want {
			t.Errorf("ResolveList(%q) = %s, want %s", tt.ref, l.ID, tt.want)
		}
	}

	// No fuzzy matching: a prefix must not match.
	_, err := c.ResolveList(ctx, "b1", "To")
	if err == nil {
		t.Error("prefix match should fail")
	} else {
		var nf *ListNotFoundError
		if !errors.As(err, &nf) {
			t.Errorf("expected ListNotFoundError, got %T", err)
		}
		if output.CodeFor(err) != output.ExitConfig {
			t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitConfig)
		}
	}
}

func TestCreateListValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"badrequest","message":"invalid name"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.CreateList(context.Background(), "b1", "", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if output.CodeFor(err) != output.ExitValidation {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitValidation)
	}
}
