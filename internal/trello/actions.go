package trello

import (
	"context"
	"net/http"
	"net/url"
)

// actionFields is the minimal field set requested for actions.
const actionFields = "id,type,date,idMemberCreator,data"

// ListBoardActions returns the actions on a board. filter is a
// comma-separated list of action types, or "all".
func (c *Client) ListBoardActions(ctx context.Context, boardID, filter string) ([]Action, error) {
	var actions []Action
	q := url.Values{"filter": {filter}, "fields": {actionFields}, "memberCreator": {"true"}, "memberCreator_fields": {"username,fullName"}}
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/actions", q, nil, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

// ListCardActions returns the actions on a card. cardRef accepts an id or a
// shortLink directly.
func (c *Client) ListCardActions(ctx context.Context, cardRef, filter string) ([]Action, error) {
	var actions []Action
	q := url.Values{"filter": {filter}, "fields": {actionFields}, "memberCreator": {"true"}, "memberCreator_fields": {"username,fullName"}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/actions", q, nil, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

// GetAction returns a single action by id, including its member creator.
func (c *Client) GetAction(ctx context.Context, id string) (*Action, error) {
	var act Action
	q := url.Values{"fields": {actionFields}, "memberCreator": {"true"}, "memberCreator_fields": {"username,fullName"}}
	if err := c.Do(ctx, http.MethodGet, "/actions/"+id, q, nil, &act); err != nil {
		return nil, err
	}
	return &act, nil
}
