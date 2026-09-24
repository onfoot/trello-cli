package trello

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const customFieldsFixture = `[
	{"id":"cf1","name":"Priority","type":"list","idModel":"b1","options":[
		{"id":"opt1","value":{"text":"High"}},
		{"id":"opt2","value":{"text":"Low"}}
	]},
	{"id":"cf2","name":"ETA","type":"date","idModel":"b1"}
]`

func TestListBoardCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/b1/customFields" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, customFieldsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	fields, err := c.ListBoardCustomFields(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[0].Name != "Priority" || fields[1].Type != "date" {
		t.Errorf("unexpected fields: %+v", fields)
	}
	if len(fields[0].Options) != 2 || fields[0].Options[0].Value.Text != "High" {
		t.Errorf("unexpected options: %+v", fields[0].Options)
	}
}

func TestGetCustomField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/customFields/cf1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"cf1","name":"Priority","type":"list","idModel":"b1","options":[{"id":"opt1","value":{"text":"High"}}]}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	f, err := c.GetCustomField(context.Background(), "cf1")
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "cf1" || f.Type != "list" || len(f.Options) != 1 {
		t.Errorf("unexpected field: %+v", f)
	}
}

func TestCustomFieldDisplayFallback(t *testing.T) {
	// Some Trello responses nest name/options under display.
	fixture := `{
		"id":"cf1",
		"idModel":"b1",
		"modelType":"board",
		"type":"list",
		"display":{"cardFront":true,"name":"Priority","pos":"16384","options":[{"id":"opt1","value":{"text":"High"}}]}
	}`
	var f CustomField
	if err := json.Unmarshal([]byte(fixture), &f); err != nil {
		t.Fatal(err)
	}
	if f.Name != "Priority" {
		t.Errorf("name = %q, want Priority (from display)", f.Name)
	}
	if len(f.Options) != 1 || f.Options[0].Value.Text != "High" {
		t.Errorf("options = %+v, want display options", f.Options)
	}
}

func TestResolveCustomField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, customFieldsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string
	}{
		{"cf1", "cf1"},
		{"ETA", "cf2"},
		{"Priority", "cf1"},
	}
	for _, tt := range tests {
		f, err := c.ResolveCustomField(ctx, "b1", tt.ref)
		if err != nil {
			t.Errorf("ResolveCustomField(%q) error: %v", tt.ref, err)
			continue
		}
		if f.ID != tt.want {
			t.Errorf("ResolveCustomField(%q) = %s, want %s", tt.ref, f.ID, tt.want)
		}
	}

	// No fuzzy matching.
	_, err := c.ResolveCustomField(ctx, "b1", "ETA ")
	if err == nil {
		t.Error("exact match should fail for a different string")
	} else {
		var nf *CustomFieldNotFoundError
		if !errors.As(err, &nf) {
			t.Errorf("expected CustomFieldNotFoundError, got %T", err)
		}
		if output.CodeFor(err) != output.ExitConfig {
			t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitConfig)
		}
	}
}

func TestSetCustomFieldItem(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/cards/c1/customField/cf1/item" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	payload := map[string]any{"value": map[string]any{"text": "hello"}}
	if err := c.SetCustomFieldItem(context.Background(), "c1", "cf1", payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"text":"hello"`) {
		t.Errorf("body = %q, want the text payload", gotBody)
	}
}

func TestCreateCustomField(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/customFields" {
			t.Errorf("path = %q", r.URL.Path)
		}
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		fmt.Fprint(w, `{"id":"cf1","name":"Priority","type":"list","idModel":"b1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	f, err := c.CreateCustomField(context.Background(), "b1", "Priority", "list", []string{"High", "Low"})
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "cf1" || f.Type != "list" {
		t.Errorf("unexpected field: %+v", f)
	}
	for _, want := range []string{`"idModel":"b1"`, `"modelType":"board"`, `"name":"Priority"`, `"type":"list"`, `"text":"High"`, `"text":"Low"`} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("body missing %s: %q", want, gotBody)
		}
	}
}

func TestDeleteCustomField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/customFields/cf1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.DeleteCustomField(context.Background(), "cf1"); err != nil {
		t.Fatal(err)
	}
}

func TestValidCustomFieldType(t *testing.T) {
	for _, typ := range CustomFieldTypes {
		if !ValidCustomFieldType(typ) {
			t.Errorf("ValidCustomFieldType(%q) = false, want true", typ)
		}
	}
	for _, typ := range []string{"checkboxs", "", "LIST", "dropdown"} {
		if ValidCustomFieldType(typ) {
			t.Errorf("ValidCustomFieldType(%q) = true, want false", typ)
		}
	}
}
