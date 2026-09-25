package trello

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

type fakeHTTPClient struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) { return f.fn(req) }

func TestDoInjectsKeyTokenAndAccept(t *testing.T) {
	var gotKey, gotToken, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("key")
		gotToken = r.URL.Query().Get("token")
		gotAccept = r.Header.Get("Accept")
		if r.URL.Path != "/members/me" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m1"}`)
	}))
	defer srv.Close()

	c := New("k123", "t456", srv.URL, srv.Client())
	var out map[string]string
	if err := c.Do(context.Background(), http.MethodGet, "/members/me", nil, nil, &out); err != nil {
		t.Fatal(err)
	}
	if gotKey != "k123" || gotToken != "t456" {
		t.Errorf("key/token injection: key=%q token=%q", gotKey, gotToken)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if out["id"] != "m1" {
		t.Errorf("decoded id = %q, want m1", out["id"])
	}
}

func TestDoMergesCallerQuery(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	q := url.Values{"fields": {"id,name"}}
	if err := c.Do(context.Background(), http.MethodGet, "/members/me/boards", q, nil, &[]Board{}); err != nil {
		t.Fatal(err)
	}
	if gotFields != "id,name" {
		t.Errorf("fields = %q, want id,name", gotFields)
	}
}

// A query string embedded in path must be merged with the injected credentials
// rather than swallowing them.
func TestDoMergesEmbeddedQueryInPath(t *testing.T) {
	var gotPath, gotKey, gotToken, gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		gotToken = r.URL.Query().Get("token")
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	var out map[string]string
	if err := c.Do(context.Background(), http.MethodGet, "/cards/c1?fields=labels", nil, nil, &out); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/cards/c1" {
		t.Errorf("path = %q, want /cards/c1", gotPath)
	}
	if gotKey != "k" || gotToken != "t" {
		t.Errorf("key/token injection: key=%q token=%q", gotKey, gotToken)
	}
	if gotFields != "labels" {
		t.Errorf("embedded fields = %q, want labels", gotFields)
	}
}

// An embedded path query and a caller url.Values both arrive.
func TestDoMergesEmbeddedAndCallerQuery(t *testing.T) {
	var gotFields, gotMembers string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFields = r.URL.Query().Get("fields")
		gotMembers = r.URL.Query().Get("members")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	q := url.Values{"members": {"true"}}
	if err := c.Do(context.Background(), http.MethodGet, "/cards/c1?fields=labels", q, nil, &map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if gotFields != "labels" || gotMembers != "true" {
		t.Errorf("merged query: fields=%q members=%q, want labels/true", gotFields, gotMembers)
	}
}

// Injected credentials must win over any user-supplied key/token values.
func TestDoCredentialsWinOverCallerQuery(t *testing.T) {
	var gotKey, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("key")
		gotToken = r.URL.Query().Get("token")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := New("real-key", "real-token", srv.URL, srv.Client())
	q := url.Values{"key": {"spoofed-key"}, "token": {"spoofed-token"}}
	if err := c.Do(context.Background(), http.MethodGet, "/cards/c1", q, nil, &map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if gotKey != "real-key" || gotToken != "real-token" {
		t.Errorf("credentials must win: key=%q token=%q", gotKey, gotToken)
	}
}

func TestStatusErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		match    func(error) bool
		wantExit int
	}{
		{"401", http.StatusUnauthorized, `{"code":"unauthorized","message":"invalid key"}`, func(e error) bool {
			var ae *AuthError
			return errors.As(e, &ae)
		}, output.ExitAuth},
		{"404", http.StatusNotFound, `{"code":"notfound","message":"no such board"}`, func(e error) bool {
			var ne *NotFoundError
			return errors.As(e, &ne)
		}, output.ExitNotFound},
		{"400", http.StatusBadRequest, `{"code":"badrequest","message":"bad"}`, func(e error) bool {
			var ve *ValidationError
			return errors.As(e, &ve)
		}, output.ExitValidation},
		{"422", http.StatusUnprocessableEntity, `{"code":"unprocessable","message":"bad"}`, func(e error) bool {
			var ve *ValidationError
			return errors.As(e, &ve)
		}, output.ExitValidation},
		{"429", http.StatusTooManyRequests, `{"code":"ratelimit","message":"slow down"}`, func(e error) bool {
			var re *RateLimitError
			return errors.As(e, &re)
		}, output.ExitRateLimit},
		{"500 generic", http.StatusInternalServerError, `{"code":"error","message":"boom"}`, func(e error) bool {
			var he *HTTPError
			return errors.As(e, &he)
		}, output.ExitError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			c := New("k", "t", srv.URL, srv.Client())
			err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil, nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !tt.match(err) {
				t.Errorf("error %T does not match expected type", err)
			}
			if got := output.CodeFor(err); got != tt.wantExit {
				t.Errorf("CodeFor = %d, want %d", got, tt.wantExit)
			}
		})
	}
}

func TestAuthErrorSurfacesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":"unauthorized","message":"invalid key"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("error should surface status and message, got: %v", err)
	}
}

func TestNetworkErrorTimeout(t *testing.T) {
	fake := &fakeHTTPClient{fn: func(req *http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Get", URL: req.URL.String(), Err: context.DeadlineExceeded}
	}}
	c := New("k", "t", "http://example.invalid", fake)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil, nil)
	var ne *NetworkError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
	if !ne.Timeout {
		t.Error("expected timeout to be flagged")
	}
	if output.CodeFor(err) != output.ExitNetwork {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitNetwork)
	}
}

func TestNetworkErrorConnectionReset(t *testing.T) {
	fake := &fakeHTTPClient{fn: func(req *http.Request) (*http.Response, error) {
		return nil, &url.Error{
			Op:  "Get",
			URL: req.URL.String(),
			Err: &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
		}
	}}
	c := New("k", "t", "http://example.invalid", fake)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil, nil)
	var ne *NetworkError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
	if ne.Timeout {
		t.Error("connection reset is not a timeout")
	}
	if output.CodeFor(err) != output.ExitNetwork {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitNetwork)
	}
}

func TestNetworkErrorDNS(t *testing.T) {
	fake := &fakeHTTPClient{fn: func(req *http.Request) (*http.Response, error) {
		return nil, &net.DNSError{Err: "no such host", Name: "example.invalid"}
	}}
	c := New("k", "t", "http://example.invalid", fake)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil, nil)
	var ne *NetworkError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
	if output.CodeFor(err) != output.ExitNetwork {
		t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitNetwork)
	}
}

func TestGetMemberMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"5b02e7f4e1facdc393169f9d","username":"bentleycook","fullName":"Bentley Cook","initials":"BC","confirmed":true}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	m, err := c.GetMemberMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "5b02e7f4e1facdc393169f9d" || m.Username != "bentleycook" || m.FullName != "Bentley Cook" {
		t.Errorf("unexpected member: %+v", m)
	}
}

func TestListMemberBoards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/boards" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		fmt.Fprint(w, `[
			{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
			{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	boards, err := c.ListMemberBoards(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 2 {
		t.Fatalf("expected 2 boards, got %d", len(boards))
	}
	if boards[0].ShortLink != "3CsPkqOF" || boards[1].Name != "Release Planning" {
		t.Errorf("unexpected boards: %+v", boards)
	}
}

func TestResolveBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":"5abbe4b7ddc1b351ef961414","name":"Trello Platform Changes","shortLink":"3CsPkqOF"},
			{"id":"5abbe4b7ddc1b351ef961415","name":"Release Planning","shortLink":"AbCdEf12"}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string // expected board ID
	}{
		{"5abbe4b7ddc1b351ef961414", "5abbe4b7ddc1b351ef961414"}, // exact id
		{"3CsPkqOF", "5abbe4b7ddc1b351ef961414"},                 // shortLink
		{"Release Planning", "5abbe4b7ddc1b351ef961415"},         // exact name
		{"Trello Platform Changes", "5abbe4b7ddc1b351ef961414"},  // exact name
	}
	for _, tt := range tests {
		b, err := c.ResolveBoard(ctx, tt.ref)
		if err != nil {
			t.Errorf("ResolveBoard(%q) error: %v", tt.ref, err)
			continue
		}
		if b.ID != tt.want {
			t.Errorf("ResolveBoard(%q) = %s, want %s", tt.ref, b.ID, tt.want)
		}
	}

	// No fuzzy matching: a prefix of a name must not match.
	if _, err := c.ResolveBoard(ctx, "Trello"); err == nil {
		t.Error("prefix match should fail")
	} else {
		var nf *BoardNotFoundError
		if !errors.As(err, &nf) {
			t.Errorf("expected BoardNotFoundError, got %T", err)
		}
		if output.CodeFor(err) != output.ExitConfig {
			t.Errorf("CodeFor = %d, want %d", output.CodeFor(err), output.ExitConfig)
		}
	}
}

func TestDecodeComment(t *testing.T) {
	fixture := `{
		"id":"5dc9b507756e182c76007621",
		"idMemberCreator":"5b02e7f4e1facdc393169f9d",
		"type":"commentCard",
		"date":"2020-03-09T19:41:51.396Z",
		"data":{
			"text":"Can never go wrong with bowie",
			"card":{"id":"c1","name":"Bowie","idShort":7,"shortLink":"3CsPkqOF"},
			"board":{"id":"b1","name":"Mullets","shortLink":"3CsPkqOF"},
			"list":{"id":"l1","name":"Amazing"}
		}
	}`
	var c Action
	if err := json.Unmarshal([]byte(fixture), &c); err != nil {
		t.Fatal(err)
	}
	if c.ID != "5dc9b507756e182c76007621" || c.Type != "commentCard" {
		t.Errorf("unexpected comment: %+v", c)
	}
	if c.Data.Text != "Can never go wrong with bowie" {
		t.Errorf("unexpected text: %q", c.Data.Text)
	}
	if c.Data.Card == nil || c.Data.Card.ShortLink != "3CsPkqOF" {
		t.Errorf("unexpected card ref: %+v", c.Data.Card)
	}
	if c.Data.Board == nil || c.Data.Board.Name != "Mullets" {
		t.Errorf("unexpected board ref: %+v", c.Data.Board)
	}
	if c.Data.List == nil || c.Data.List.ID != "l1" {
		t.Errorf("unexpected list ref: %+v", c.Data.List)
	}
	if c.Date == nil || c.Date.IsZero() {
		t.Error("date should be parsed")
	}
}
