package trello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// SearchOptions controls a GET /search request.
type SearchOptions struct {
	// ModelTypes limits the search to these model types (boards, cards,
	// members, organizations). Empty means the API default.
	ModelTypes []string
	// IDBoards scopes the search to a single board id.
	IDBoards string
}

// SearchResult is the structured response of GET /search.
type SearchResult struct {
	Boards        []Board        `json:"boards"`
	Cards         []Card         `json:"cards"`
	Members       []Member       `json:"members"`
	Organizations []Organization `json:"organizations"`
}

// UnmarshalJSON decodes a /search response, normalizing absent or null group
// keys to empty slices so the four group keys always marshal as JSON arrays
// (never null) — keeping --json output deterministic for agents.
func (s *SearchResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		Boards        []Board        `json:"boards"`
		Cards         []Card         `json:"cards"`
		Members       []Member       `json:"members"`
		Organizations []Organization `json:"organizations"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.Boards = nonNilSlice(raw.Boards)
	s.Cards = nonNilSlice(raw.Cards)
	s.Members = nonNilSlice(raw.Members)
	s.Organizations = nonNilSlice(raw.Organizations)
	return nil
}

// nonNilSlice returns s, or an empty slice when s is nil.
func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// Search performs a full-text search across the requested model types.
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	q := url.Values{"query": {query}}
	if len(opts.ModelTypes) > 0 {
		q.Set("modelTypes", strings.Join(opts.ModelTypes, ","))
	}
	if opts.IDBoards != "" {
		q.Set("idBoards", opts.IDBoards)
	}
	var res SearchResult
	if err := c.Do(ctx, http.MethodGet, "/search", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
