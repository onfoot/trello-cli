package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// labelFields is the minimal field set requested for labels.
const labelFields = "id,name,color,idBoard"

// LabelColors lists the valid label colors, in canonical order.
var LabelColors = []string{"yellow", "purple", "blue", "red", "green", "orange", "black", "sky", "pink", "lime"}

// validLabelColors is a set for O(1) membership checks.
var validLabelColors = func() map[string]bool {
	m := make(map[string]bool, len(LabelColors))
	for _, c := range LabelColors {
		m[c] = true
	}
	return m
}()

// ValidLabelColor reports whether c is a valid label color.
func ValidLabelColor(c string) bool {
	return validLabelColors[c]
}

// LabelUpdate holds the optional fields for PUT /labels/{id}. Nil fields are
// not sent.
type LabelUpdate struct {
	Name  *string
	Color *string
}

// LabelNotFoundError is returned when a label reference matches no label on
// the board. Matching is exact (id or name) — never fuzzy.
type LabelNotFoundError struct {
	Ref string
}

func (e *LabelNotFoundError) Error() string {
	return fmt.Sprintf("label %q not found on the board (matched by exact id or name); run 'trello label list' to see the board's labels", e.Ref)
}

// ExitCode reports the config exit code: an explicit label that matches
// nothing is a configuration error, never a silent fallback.
func (e *LabelNotFoundError) ExitCode() int { return output.ExitConfig }

// ListBoardLabels returns the labels on a board.
func (c *Client) ListBoardLabels(ctx context.Context, boardID string) ([]Label, error) {
	var labels []Label
	q := url.Values{"fields": {labelFields}}
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/labels", q, nil, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// GetLabel returns a single label by id.
func (c *Client) GetLabel(ctx context.Context, id string) (*Label, error) {
	var l Label
	q := url.Values{"fields": {labelFields}}
	if err := c.Do(ctx, http.MethodGet, "/labels/"+id, q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateBoardLabel creates a label on a board. Writes use the query-param
// convention.
func (c *Client) CreateBoardLabel(ctx context.Context, boardID, name string, color *string) (*Label, error) {
	q := url.Values{"name": {name}}
	if color != nil {
		q.Set("color", *color)
	}
	var l Label
	if err := c.Do(ctx, http.MethodPost, "/boards/"+boardID+"/labels", q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateLabel updates a label's name or color.
func (c *Client) UpdateLabel(ctx context.Context, id string, u LabelUpdate) (*Label, error) {
	q := url.Values{}
	if u.Name != nil {
		q.Set("name", *u.Name)
	}
	if u.Color != nil {
		q.Set("color", *u.Color)
	}
	var l Label
	if err := c.Do(ctx, http.MethodPut, "/labels/"+id, q, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// DeleteLabel permanently deletes a label.
func (c *Client) DeleteLabel(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/labels/"+id, nil, nil, nil)
}

// AddLabelToCard adds a label to a card.
func (c *Client) AddLabelToCard(ctx context.Context, cardRef, labelID string) error {
	q := url.Values{"value": {labelID}}
	return c.Do(ctx, http.MethodPost, "/cards/"+cardRef+"/idLabels", q, nil, nil)
}

// RemoveLabelFromCard removes a label from a card.
func (c *Client) RemoveLabelFromCard(ctx context.Context, cardRef, labelID string) error {
	return c.Do(ctx, http.MethodDelete, "/cards/"+cardRef+"/idLabels/"+labelID, nil, nil, nil)
}

// ResolveLabel matches ref against the board's labels by exact id or name.
// A reference that matches nothing returns a LabelNotFoundError.
func (c *Client) ResolveLabel(ctx context.Context, boardID, ref string) (*Label, error) {
	labels, err := c.ListBoardLabels(ctx, boardID)
	if err != nil {
		return nil, err
	}
	for i := range labels {
		l := &labels[i]
		if l.ID == ref || l.Name == ref {
			return l, nil
		}
	}
	return nil, &LabelNotFoundError{Ref: ref}
}

// GetCardBoard returns the board a card belongs to (used to derive the board
// for label resolution on label add/remove).
func (c *Client) GetCardBoard(ctx context.Context, cardRef string) (*Board, error) {
	var b Board
	q := url.Values{"fields": {"id"}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/board", q, nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
