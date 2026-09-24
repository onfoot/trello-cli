package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// listFields is the minimal field set requested for lists.
const listFields = "id,name,closed,pos,idBoard"

// ListUpdate holds the optional fields for PUT /lists/{id}. Nil fields are
// not sent.
type ListUpdate struct {
	Name   *string
	Closed *bool
	Pos    *float64
}

// ListNotFoundError is returned when a list reference matches no list on the
// board. Matching is exact (id or name) — never fuzzy.
type ListNotFoundError struct {
	Ref string
}

func (e *ListNotFoundError) Error() string {
	return fmt.Sprintf("list %q not found on the board (matched by exact id or name); run 'trello list list' to see the board's lists", e.Ref)
}

// ExitCode reports the config exit code: an explicit list that matches
// nothing is a configuration error, never a silent fallback.
func (e *ListNotFoundError) ExitCode() int { return output.ExitConfig }

// ListBoardLists returns the lists on a board.
func (c *Client) ListBoardLists(ctx context.Context, boardID string) ([]List, error) {
	var lists []List
	q := url.Values{"fields": {listFields}}
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/lists", q, nil, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

// GetList returns a single list by id.
func (c *Client) GetList(ctx context.Context, id string) (*List, error) {
	var l List
	q := url.Values{"fields": {listFields}}
	if err := c.Do(ctx, http.MethodGet, "/lists/"+id, q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateList creates a list on a board.
func (c *Client) CreateList(ctx context.Context, boardID, name string, pos *float64) (*List, error) {
	q := url.Values{"name": {name}, "idBoard": {boardID}}
	if pos != nil {
		q.Set("pos", strconv.FormatFloat(*pos, 'f', -1, 64))
	}
	var l List
	if err := c.Do(ctx, http.MethodPost, "/lists", q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateList updates a list's name, closed state, or position.
func (c *Client) UpdateList(ctx context.Context, id string, u ListUpdate) (*List, error) {
	q := url.Values{}
	if u.Name != nil {
		q.Set("name", *u.Name)
	}
	if u.Closed != nil {
		q.Set("closed", strconv.FormatBool(*u.Closed))
	}
	if u.Pos != nil {
		q.Set("pos", strconv.FormatFloat(*u.Pos, 'f', -1, 64))
	}
	var l List
	if err := c.Do(ctx, http.MethodPut, "/lists/"+id, q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// ArchiveList closes (archives) a list.
func (c *Client) ArchiveList(ctx context.Context, id string) (*List, error) {
	q := url.Values{"value": {"true"}}
	var l List
	if err := c.Do(ctx, http.MethodPut, "/lists/"+id+"/closed", q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// ResolveList matches ref against the board's lists by exact id or name.
// A reference that matches nothing returns a ListNotFoundError.
func (c *Client) ResolveList(ctx context.Context, boardID, ref string) (*List, error) {
	lists, err := c.ListBoardLists(ctx, boardID)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		l := &lists[i]
		if l.ID == ref || l.Name == ref {
			return l, nil
		}
	}
	return nil, &ListNotFoundError{Ref: ref}
}
