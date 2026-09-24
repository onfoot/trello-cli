package trello

import (
	"encoding/json"
	"testing"
)

func TestDecodeCard(t *testing.T) {
	fixture := `{
		"id":"5abbe4b7ddc1b351ef961414",
		"name":"Bowie",
		"desc":"A card",
		"closed":false,
		"idBoard":"5abbe4b7ddc1b351ef961414",
		"idList":"5abbe4b7ddc1b351ef961415",
		"idShort":7,
		"shortLink":"3CsPkqOF",
		"shortUrl":"https://trello.com/c/3CsPkqOF",
		"url":"https://trello.com/c/3CsPkqOF/bowie",
		"pos":65535,
		"dateLastActivity":"2019-09-16T16:19:17.156Z"
	}`
	var c Card
	if err := json.Unmarshal([]byte(fixture), &c); err != nil {
		t.Fatal(err)
	}
	if c.ID != "5abbe4b7ddc1b351ef961414" || c.Name != "Bowie" || c.IDList != "5abbe4b7ddc1b351ef961415" {
		t.Errorf("unexpected card: %+v", c)
	}
	if c.IDShort != 7 || c.ShortLink != "3CsPkqOF" || c.Pos != 65535 {
		t.Errorf("unexpected card fields: %+v", c)
	}
	if c.DateLastActivity == nil || c.DateLastActivity.IsZero() {
		t.Error("dateLastActivity should be parsed")
	}
}

func TestDecodeList(t *testing.T) {
	fixture := `{"id":"l1","name":"Things to buy today","closed":false,"pos":123.5,"idBoard":"b1"}`
	var l List
	if err := json.Unmarshal([]byte(fixture), &l); err != nil {
		t.Fatal(err)
	}
	if l.ID != "l1" || l.Name != "Things to buy today" || l.IDBoard != "b1" || l.Pos != 123.5 {
		t.Errorf("unexpected list: %+v", l)
	}
}

func TestDecodeChecklistWithItems(t *testing.T) {
	fixture := `{
		"id":"cl1",
		"name":"Release checklist",
		"idBoard":"b1",
		"checkItems":[
			{"id":"ci1","idChecklist":"cl1","name":"Update docs","state":"complete","pos":1673},
			{"id":"ci2","idChecklist":"cl1","name":"Tag release","state":"incomplete","pos":32767}
		]
	}`
	var cl Checklist
	if err := json.Unmarshal([]byte(fixture), &cl); err != nil {
		t.Fatal(err)
	}
	if cl.ID != "cl1" || cl.Name != "Release checklist" {
		t.Errorf("unexpected checklist: %+v", cl)
	}
	if len(cl.CheckItems) != 2 {
		t.Fatalf("expected 2 check items, got %d", len(cl.CheckItems))
	}
	if cl.CheckItems[0].State != "complete" || cl.CheckItems[1].Name != "Tag release" {
		t.Errorf("unexpected check items: %+v", cl.CheckItems)
	}
}

func TestDecodeMember(t *testing.T) {
	fixture := `{"id":"m1","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`
	var m Member
	if err := json.Unmarshal([]byte(fixture), &m); err != nil {
		t.Fatal(err)
	}
	if m.ID != "m1" || m.Username != "bentleycook" || !m.Confirmed {
		t.Errorf("unexpected member: %+v", m)
	}
}
