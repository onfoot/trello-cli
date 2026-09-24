package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCardChecklists(t *testing.T) {
	var gotCheckItems string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/checklists" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotCheckItems = r.URL.Query().Get("checkItems")
		fmt.Fprint(w, `[
			{"id":"cl1","name":"Release","idBoard":"b1","checkItems":[
				{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673}
			]}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	checklists, err := c.ListCardChecklists(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(checklists) != 1 || checklists[0].Name != "Release" {
		t.Errorf("unexpected checklists: %+v", checklists)
	}
	if len(checklists[0].CheckItems) != 1 || checklists[0].CheckItems[0].State != "complete" {
		t.Errorf("unexpected check items: %+v", checklists[0].CheckItems)
	}
	if gotCheckItems != "all" {
		t.Errorf("checkItems = %q, want all", gotCheckItems)
	}
}

func TestGetChecklist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checklists/cl1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1","checkItems":[]}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	cl, err := c.GetChecklist(context.Background(), "cl1")
	if err != nil {
		t.Fatal(err)
	}
	if cl.ID != "cl1" || cl.Name != "Release" || cl.IDBoard != "b1" {
		t.Errorf("unexpected checklist: %+v", cl)
	}
}

func TestCreateChecklist(t *testing.T) {
	var gotCard, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/checklists" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotCard = r.URL.Query().Get("idCard")
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	cl, err := c.CreateChecklist(context.Background(), "c1", "Release")
	if err != nil {
		t.Fatal(err)
	}
	if cl.ID != "cl1" || cl.Name != "Release" {
		t.Errorf("unexpected checklist: %+v", cl)
	}
	if gotCard != "c1" || gotName != "Release" {
		t.Errorf("query params: idCard=%q name=%q", gotCard, gotName)
	}
}

func TestAddCheckItem(t *testing.T) {
	var gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/checklists/cl1/checkItems" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"incomplete","pos":1673}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	item, err := c.AddCheckItem(context.Background(), "cl1", "Update docs")
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "ci1" || item.IDChecklist != "cl1" || item.Name != "Update docs" {
		t.Errorf("unexpected item: %+v", item)
	}
	if gotName != "Update docs" {
		t.Errorf("name = %q, want Update docs", gotName)
	}
}

func TestSetCheckItemState(t *testing.T) {
	var gotChecklistPath, gotPutPath, gotState string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/checklists/cl1":
			// First: fetch the checklist to learn the card id.
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET for the checklist", r.Method)
			}
			gotChecklistPath = r.URL.Path
			fmt.Fprint(w, `{"id":"cl1","name":"Release","idBoard":"b1","idCard":"c1"}`)
		case r.URL.Path == "/cards/c1/checkItem/ci1":
			// Then: set the state on the card-scoped endpoint.
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT for the check item", r.Method)
			}
			gotPutPath = r.URL.Path
			gotState = r.URL.Query().Get("state")
			fmt.Fprint(w, `{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	item, err := c.SetCheckItemState(context.Background(), "cl1", "ci1", "complete")
	if err != nil {
		t.Fatal(err)
	}
	if item.State != "complete" {
		t.Errorf("state = %q, want complete", item.State)
	}
	if gotChecklistPath != "/checklists/cl1" {
		t.Errorf("checklist path = %q, want /checklists/cl1", gotChecklistPath)
	}
	if gotPutPath != "/cards/c1/checkItem/ci1" {
		t.Errorf("put path = %q, want /cards/c1/checkItem/ci1", gotPutPath)
	}
	if gotState != "complete" {
		t.Errorf("state param = %q, want complete", gotState)
	}
}

func TestSetCheckItemStateMissingCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"cl1","name":"Release"}`) // no idCard
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	_, err := c.SetCheckItemState(context.Background(), "cl1", "ci1", "complete")
	if err == nil {
		t.Fatal("expected an error when the checklist reports no idCard")
	}
	if !strings.Contains(err.Error(), "idCard") {
		t.Errorf("error should mention idCard: %v", err)
	}
}
