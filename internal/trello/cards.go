package trello

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// cardFields is the minimal field set requested for cards.
const cardFields = "id,name,desc,idList,idBoard,closed,due,shortLink,pos,url"

// CardCreate holds the fields for POST /cards.
type CardCreate struct {
	IDList string
	Name   string
	Desc   string
	Pos    *float64
	Due    *string // RFC3339, sent verbatim once validated by the caller
}

// CardUpdate holds the optional fields for PUT /cards/{id}. Nil fields are
// not sent.
type CardUpdate struct {
	Name   *string
	Desc   *string
	Closed *bool
	Due    *string // RFC3339, sent verbatim once validated by the caller
	Pos    *float64
	IDList *string // resolved list id; moves the card when set
}

// ListBoardCards returns the open cards on a board.
func (c *Client) ListBoardCards(ctx context.Context, boardID string) ([]Card, error) {
	var cards []Card
	q := url.Values{"fields": {cardFields}}
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/cards", q, nil, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// ListListCards returns the cards in a list.
func (c *Client) ListListCards(ctx context.Context, listID string) ([]Card, error) {
	var cards []Card
	q := url.Values{"fields": {cardFields}}
	if err := c.Do(ctx, http.MethodGet, "/lists/"+listID+"/cards", q, nil, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// GetCard returns a single card by id.
func (c *Client) GetCard(ctx context.Context, id string) (*Card, error) {
	var card Card
	q := url.Values{"fields": {cardFields}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+id, q, nil, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// CreateCard creates a card in a list. Writes use the query-param convention.
func (c *Client) CreateCard(ctx context.Context, in CardCreate) (*Card, error) {
	q := url.Values{"name": {in.Name}, "idList": {in.IDList}}
	if in.Desc != "" {
		q.Set("desc", in.Desc)
	}
	if in.Pos != nil {
		q.Set("pos", strconv.FormatFloat(*in.Pos, 'f', -1, 64))
	}
	if in.Due != nil {
		q.Set("due", *in.Due)
	}
	var card Card
	if err := c.Do(ctx, http.MethodPost, "/cards", q, nil, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateCard updates a card's fields. Setting IDList moves the card.
func (c *Client) UpdateCard(ctx context.Context, id string, u CardUpdate) (*Card, error) {
	q := url.Values{}
	if u.Name != nil {
		q.Set("name", *u.Name)
	}
	if u.Desc != nil {
		q.Set("desc", *u.Desc)
	}
	if u.Closed != nil {
		q.Set("closed", strconv.FormatBool(*u.Closed))
	}
	if u.Due != nil {
		q.Set("due", *u.Due)
	}
	if u.Pos != nil {
		q.Set("pos", strconv.FormatFloat(*u.Pos, 'f', -1, 64))
	}
	if u.IDList != nil {
		q.Set("idList", *u.IDList)
	}
	var card Card
	if err := c.Do(ctx, http.MethodPut, "/cards/"+id, q, nil, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// MoveCard moves a card to another list via PUT /cards/{id}/idList.
func (c *Client) MoveCard(ctx context.Context, id, listID string) (*Card, error) {
	q := url.Values{"value": {listID}}
	var card Card
	if err := c.Do(ctx, http.MethodPut, "/cards/"+id+"/idList", q, nil, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// ArchiveCard closes (archives) a card.
func (c *Client) ArchiveCard(ctx context.Context, id string) (*Card, error) {
	q := url.Values{"closed": {"true"}}
	var card Card
	if err := c.Do(ctx, http.MethodPut, "/cards/"+id, q, nil, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// DeleteCard permanently deletes a card.
func (c *Client) DeleteCard(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/cards/"+id, nil, nil, nil)
}
