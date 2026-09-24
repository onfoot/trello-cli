package trello

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// CustomFieldTypes lists the valid custom field types, in canonical order.
var CustomFieldTypes = []string{"checkbox", "list", "number", "text", "date"}

// validCustomFieldTypeSet is a set for O(1) membership checks.
var validCustomFieldTypeSet = func() map[string]bool {
	m := make(map[string]bool, len(CustomFieldTypes))
	for _, t := range CustomFieldTypes {
		m[t] = true
	}
	return m
}()

// ValidCustomFieldType reports whether t is a valid custom field type.
func ValidCustomFieldType(t string) bool {
	return validCustomFieldTypeSet[t]
}

// CustomFieldNotFoundError is returned when a custom field reference matches
// no field on the board. Matching is exact (id or name) — never fuzzy.
type CustomFieldNotFoundError struct {
	Ref string
}

func (e *CustomFieldNotFoundError) Error() string {
	return fmt.Sprintf("custom field %q not found on the board (matched by exact id or name); run 'trello customfield list' to see the board's custom fields", e.Ref)
}

// ExitCode reports the config exit code: an explicit custom field that
// matches nothing is a configuration error, never a silent fallback.
func (e *CustomFieldNotFoundError) ExitCode() int { return output.ExitConfig }

// ListBoardCustomFields returns the custom fields defined on a board.
func (c *Client) ListBoardCustomFields(ctx context.Context, boardID string) ([]CustomField, error) {
	var fields []CustomField
	if err := c.Do(ctx, http.MethodGet, "/boards/"+boardID+"/customFields", nil, nil, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// GetCustomField returns a single custom field by id, including its options.
func (c *Client) GetCustomField(ctx context.Context, id string) (*CustomField, error) {
	var f CustomField
	if err := c.Do(ctx, http.MethodGet, "/customFields/"+id, nil, nil, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// ResolveCustomField matches ref against the board's custom fields by exact
// id or name. A reference that matches nothing returns a
// CustomFieldNotFoundError.
func (c *Client) ResolveCustomField(ctx context.Context, boardID, ref string) (*CustomField, error) {
	fields, err := c.ListBoardCustomFields(ctx, boardID)
	if err != nil {
		return nil, err
	}
	for i := range fields {
		f := &fields[i]
		if f.ID == ref || f.Name == ref {
			return f, nil
		}
	}
	return nil, &CustomFieldNotFoundError{Ref: ref}
}

// SetCustomFieldItem sets a card's value for one custom field. payload is the
// type-shaped JSON body ({"value":{...}} or {"idValue": optionId}) and is
// sent as application/json.
func (c *Client) SetCustomFieldItem(ctx context.Context, cardRef, fieldID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.Do(ctx, http.MethodPut, "/cards/"+cardRef+"/customField/"+fieldID+"/item", nil, bytes.NewReader(body), nil)
}

// CreateCustomField defines a new custom field on a board. options is only
// meaningful for list type.
func (c *Client) CreateCustomField(ctx context.Context, boardID, name, fieldType string, options []string) (*CustomField, error) {
	payload := map[string]any{
		"idModel":   boardID,
		"modelType": "board",
		"name":      name,
		"type":      fieldType,
		"pos":       "bottom",
	}
	if len(options) > 0 {
		opts := make([]map[string]any, 0, len(options))
		for _, o := range options {
			opts = append(opts, map[string]any{"value": map[string]any{"text": o}})
		}
		payload["options"] = opts
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var f CustomField
	if err := c.Do(ctx, http.MethodPost, "/customFields", nil, bytes.NewReader(body), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// DeleteCustomField permanently deletes a custom field definition.
func (c *Client) DeleteCustomField(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/customFields/"+id, nil, nil, nil)
}
