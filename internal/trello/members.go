package trello

import (
	"context"
	"net/http"
	"net/url"
)

// memberFields is the minimal field set requested for members.
const memberFields = "id,username,fullName,initials,url,avatarUrl,bio,email,confirmed"

// ListBoardMembers returns the members of a board.
func (c *Client) ListBoardMembers(ctx context.Context, boardID string) ([]Member, error) {
	var members []Member
	q := url.Values{"fields": {memberFields}}
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/members", q, nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// GetMember returns a member by id, username, or the special value "me".
func (c *Client) GetMember(ctx context.Context, ref string) (*Member, error) {
	var m Member
	q := url.Values{"fields": {memberFields}}
	if err := c.Do(ctx, http.MethodGet, "/members/"+ref, q, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
