package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// checklistFields is the minimal field set requested for checklists.
const checklistFields = "id,name,idBoard,idCard"

// ListCardChecklists returns the checklists on a card, including their items.
func (c *Client) ListCardChecklists(ctx context.Context, cardRef string) ([]Checklist, error) {
	var checklists []Checklist
	q := url.Values{"fields": {checklistFields}, "checkItems": {"all"}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/checklists", q, nil, &checklists); err != nil {
		return nil, err
	}
	return checklists, nil
}

// GetChecklist returns a single checklist, including its items.
func (c *Client) GetChecklist(ctx context.Context, id string) (*Checklist, error) {
	var cl Checklist
	q := url.Values{"fields": {checklistFields}, "checkItems": {"all"}}
	if err := c.Do(ctx, http.MethodGet, "/checklists/"+id, q, nil, &cl); err != nil {
		return nil, err
	}
	return &cl, nil
}

// CreateChecklist creates a checklist on a card.
func (c *Client) CreateChecklist(ctx context.Context, cardRef, name string) (*Checklist, error) {
	q := url.Values{"idCard": {cardRef}, "name": {name}}
	var cl Checklist
	if err := c.Do(ctx, http.MethodPost, "/checklists", q, nil, &cl); err != nil {
		return nil, err
	}
	return &cl, nil
}

// AddCheckItem adds an item to a checklist.
func (c *Client) AddCheckItem(ctx context.Context, checklistID, name string) (*CheckItem, error) {
	q := url.Values{"name": {name}}
	var item CheckItem
	if err := c.Do(ctx, http.MethodPost, "/checklists/"+checklistID+"/checkItems", q, nil, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// SetCheckItemState sets a checklist item's state to complete or incomplete.
// The state endpoint is scoped to the item's card, so the checklist is
// fetched first to learn the card id.
func (c *Client) SetCheckItemState(ctx context.Context, checklistID, checkItemID, state string) (*CheckItem, error) {
	cl, err := c.GetChecklist(ctx, checklistID)
	if err != nil {
		return nil, err
	}
	if cl.IDCard == "" {
		return nil, fmt.Errorf("checklist %s did not report an idCard", checklistID)
	}
	q := url.Values{"state": {state}}
	var item CheckItem
	if err := c.Do(ctx, http.MethodPut, "/cards/"+cl.IDCard+"/checkItem/"+checkItemID, q, nil, &item); err != nil {
		return nil, err
	}
	return &item, nil
}
