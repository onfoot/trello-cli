package trello

import (
	"context"
	"net/http"
	"net/url"
)

// Comments are Trello Actions of type commentCard. Card references accept an
// id or a shortLink directly — no board resolution is required.

// AddComment posts a comment to a card.
func (c *Client) AddComment(ctx context.Context, cardRef, text string) (*Action, error) {
	q := url.Values{"text": {text}}
	var comment Action
	if err := c.Do(ctx, http.MethodPost, "/cards/"+cardRef+"/actions/comments", q, nil, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListComments returns the comments on a card.
func (c *Client) ListComments(ctx context.Context, cardRef string) ([]Action, error) {
	var comments []Action
	q := url.Values{"filter": {"commentCard"}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/actions", q, nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}
