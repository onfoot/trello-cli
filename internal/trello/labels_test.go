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

func TestListBoardLabels(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/b1/labels" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"l1","name":"Overdue","color":"red","idBoard":"b1"},
			{"id":"l2","name":"Needs review","color":"blue","idBoard":"b1"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	labels, err := c.ListBoardLabels(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 2 || labels[0].Name != "Overdue" || labels[1].Color != "blue" {
		t.Errorf("unexpected labels: %+v", labels)
	}
	if gotFields != "id,name,color,idBoard" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestGetLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/labels/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"l1","name":"Overdue","color":"red","idBoard":"b1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	l, err := c.GetLabel(context.Background(), "l1")
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "l1" || l.Name != "Overdue" || l.Color != "red" || l.IDBoard != "b1" {
		t.Errorf("unexpected label: %+v", l)
	}
}

func TestCreateBoardLabel(t *testing.T) {
	var gotName, gotColor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/boards/b1/labels" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotColor = r.URL.Query().Get("color")
		fmt.Fprint(w, `{"id":"l1","name":"Overdue","color":"red","idBoard":"b1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	color := "red"
	l, err := c.CreateBoardLabel(context.Background(), "b1", "Overdue", &color)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "l1" || l.Name != "Overdue" {
		t.Errorf("unexpected label: %+v", l)
	}
	if gotName != "Overdue" || gotColor != "red" {
		t.Errorf("query params: name=%q color=%q", gotName, gotColor)
	}
}

func TestCreateBoardLabelWithoutColor(t *testing.T) {
	var gotColor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotColor = r.URL.Query().Get("color")
		fmt.Fprint(w, `{"id":"l1","name":"Overdue"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if _, err := c.CreateBoardLabel(context.Background(), "b1", "Overdue", nil); err != nil {
		t.Fatal(err)
	}
	if gotColor != "" {
		t.Errorf("color = %q, want unset", gotColor)
	}
}

func TestUpdateLabel(t *testing.T) {
	var gotName, gotColor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/labels/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		gotColor = r.URL.Query().Get("color")
		fmt.Fprint(w, `{"id":"l1","name":"New","color":"green"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	l, err := c.UpdateLabel(context.Background(), "l1", LabelUpdate{Name: strPtr("New"), Color: strPtr("green")})
	if err != nil {
		t.Fatal(err)
	}
	if l.Name != "New" || l.Color != "green" {
		t.Errorf("unexpected label: %+v", l)
	}
	if gotName != "New" || gotColor != "green" {
		t.Errorf("query params: name=%q color=%q", gotName, gotColor)
	}
}

func TestDeleteLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/labels/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.DeleteLabel(context.Background(), "l1"); err != nil {
		t.Fatal(err)
	}
}

func TestAddLabelToCard(t *testing.T) {
	var gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/c1/idLabels" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotValue = r.URL.Query().Get("value")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.AddLabelToCard(context.Background(), "c1", "l1"); err != nil {
		t.Fatal(err)
	}
	if gotValue != "l1" {
		t.Errorf("value = %q, want l1", gotValue)
	}
}

func TestRemoveLabelFromCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/cards/c1/idLabels/l1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.RemoveLabelFromCard(context.Background(), "c1", "l1"); err != nil {
		t.Fatal(err)
	}
}

func TestResolveLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":"l1","name":"Overdue","color":"red","idBoard":"b1"},
			{"id":"l2","name":"Needs review","color":"blue","idBoard":"b1"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string
	}{
		{"l1", "l1"},
		{"Needs review", "l2"},
		{"Overdue", "l1"},
	}
	for _, tt := range tests {
		l, err := c.ResolveLabel(ctx, "b1", tt.ref)
		if err != nil {
			t.Errorf("ResolveLabel(%q) error: %v", tt.ref, err)
			continue
		}
		if l.ID != tt.want {
			t.Errorf("ResolveLabel(%q) = %s, want %s", tt.ref, l.ID, tt.want)
		}
	}

	// No fuzzy matching: a prefix must not match.
	_, err := c.ResolveLabel(ctx, "b1", "Over")
	if err == nil {
		t.Error("prefix match should fail")
	} else {
		var nf *LabelNotFoundError
		if !errors.As(err, &nf) {
			t.Errorf("expected LabelNotFoundError, got %T", err)
		}
		if output.CodeFor(err) != output.ExitConfig {
			t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitConfig)
		}
	}
}

func TestGetCardBoard(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/AbCdEf01/board" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `{"id":"b1","name":"Board"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	b, err := c.GetCardBoard(context.Background(), "AbCdEf01")
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != "b1" {
		t.Errorf("board id = %q, want b1", b.ID)
	}
	if gotFields != "id" {
		t.Errorf("fields = %q, want id", gotFields)
	}
}

func TestValidLabelColor(t *testing.T) {
	for _, color := range LabelColors {
		if !ValidLabelColor(color) {
			t.Errorf("ValidLabelColor(%q) = false, want true", color)
		}
	}
	for _, color := range []string{"chartreuse", "", "RED", "yellowish"} {
		if ValidLabelColor(color) {
			t.Errorf("ValidLabelColor(%q) = true, want false", color)
		}
	}
}

func TestCreateLabelValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"badrequest","message":"invalid color"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	color := "chartreuse"
	_, err := c.CreateBoardLabel(context.Background(), "b1", "Overdue", &color)
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
