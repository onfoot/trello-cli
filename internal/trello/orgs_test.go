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

const orgsFixture = `[
	{"id":"o1","name":"acme","displayName":"Acme Inc","desc":"The Acme team","url":"https://trello.com/acme","website":"https://acme.example.com"},
	{"id":"o2","name":"globex","displayName":"Globex","desc":"","url":"https://trello.com/globex","website":""}
]`

func TestListMyOrganizations(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/members/me/organizations" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, orgsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	orgs, err := c.ListMyOrganizations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(orgs) != 2 || orgs[0].Name != "acme" || orgs[1].DisplayName != "Globex" {
		t.Errorf("unexpected orgs: %+v", orgs)
	}
	if gotFields != "id,name,displayName,desc,url,website" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestGetOrganization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/acme" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"o1","name":"acme","displayName":"Acme Inc","desc":"The Acme team","url":"https://trello.com/acme","website":"https://acme.example.com"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	o, err := c.GetOrganization(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if o.ID != "o1" || o.Name != "acme" || o.DisplayName != "Acme Inc" || o.Website != "https://acme.example.com" {
		t.Errorf("unexpected org: %+v", o)
	}
}

func TestListOrgBoards(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/o1/boards" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"b1","name":"Trello Platform Changes","shortLink":"3CsPkqOF","closed":false},
			{"id":"b2","name":"Release Planning","shortLink":"AbCdEf12","closed":true}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	boards, err := c.ListOrgBoards(context.Background(), "o1")
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 2 || boards[0].ShortLink != "3CsPkqOF" || !boards[1].Closed {
		t.Errorf("unexpected boards: %+v", boards)
	}
	if gotFields != "id,name,shortLink,closed" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestListOrgMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organizations/o1/members" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `[{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	members, err := c.ListOrgMembers(context.Background(), "o1")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Username != "bentleycook" {
		t.Errorf("unexpected members: %+v", members)
	}
}

func TestResolveOrgByID(t *testing.T) {
	// A 24-hex id is used directly without listing.
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"id":"5abbe4b7ddc1b351ef961414","name":"acme","displayName":"Acme Inc"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	o, err := c.ResolveOrg(context.Background(), "5abbe4b7ddc1b351ef961414")
	if err != nil {
		t.Fatal(err)
	}
	if o.ID != "5abbe4b7ddc1b351ef961414" {
		t.Errorf("unexpected org: %+v", o)
	}
	if gotPath != "/organizations/5abbe4b7ddc1b351ef961414" {
		t.Errorf("path = %q, want direct GET by id", gotPath)
	}
}

func TestResolveOrgByNameAndDisplayName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/organizations" {
			t.Errorf("path = %q, want the member orgs listing", r.URL.Path)
		}
		fmt.Fprint(w, orgsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string
	}{
		{"acme", "o1"},
		{"Globex", "o2"},
		{"Acme Inc", "o1"},
	}
	for _, tt := range tests {
		o, err := c.ResolveOrg(ctx, tt.ref)
		if err != nil {
			t.Errorf("ResolveOrg(%q) error: %v", tt.ref, err)
			continue
		}
		if o.ID != tt.want {
			t.Errorf("ResolveOrg(%q) = %s, want %s", tt.ref, o.ID, tt.want)
		}
	}
}

func TestResolveOrgNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, orgsFixture)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.ResolveOrg(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected an error for a no-match ref")
	}
	var nf *OrgNotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("expected OrgNotFoundError, got %T", err)
	}
	if output.CodeFor(err) != output.ExitConfig {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitConfig)
	}
}

func TestGetOrganizationNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"notfound","message":"no such organization"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.GetOrganization(context.Background(), "missing")
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

func TestIsTrelloID(t *testing.T) {
	for _, s := range []string{"5abbe4b7ddc1b351ef961414", "5ABBE4B7DDC1B351EF961414"} {
		if !IsTrelloID(s) {
			t.Errorf("IsTrelloID(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"acme", "", "5abbe4b7ddc1b351ef96141", "5abbe4b7ddc1b351ef961414z", "5abbe4b7ddc1b351ef9614141"} {
		if IsTrelloID(s) {
			t.Errorf("IsTrelloID(%q) = true, want false", s)
		}
	}
}
