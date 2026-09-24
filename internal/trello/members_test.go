package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListBoardMembers(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/b1/members" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true},
			{"id":"m2","username":"bobloblaw","fullName":"Bob Loblaw","initials":"BL","confirmed":true}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	members, err := c.ListBoardMembers(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].Username != "bentleycook" || members[1].FullName != "Bob Loblaw" {
		t.Errorf("unexpected members: %+v", members)
	}
	if gotFields != "id,username,fullName,initials,url,avatarUrl,bio,email,confirmed" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestGetMember(t *testing.T) {
	tests := []struct {
		ref  string
		path string
	}{
		{"bentleycook", "/members/bentleycook"},
		{"5b02e7f4e1facdc393169f9d", "/members/5b02e7f4e1facdc393169f9d"},
		{"me", "/members/me"},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Errorf("path = %q, want %q", r.URL.Path, tt.path)
				}
				fmt.Fprint(w, `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
			}))
			defer srv.Close()

			c := New("k", "t", srv.URL, srv.Client())
			m, err := c.GetMember(context.Background(), tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			if m.Username != "bentleycook" || m.ID != "m1" {
				t.Errorf("unexpected member: %+v", m)
			}
		})
	}
}
