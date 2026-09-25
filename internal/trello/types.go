package trello

import (
	"encoding/json"
	"time"
)

// The response structs below model the Phase-1 subset of the Trello REST API
// schema (see trello-api.json). Fields are deliberately minimal but real:
// IDs are raw strings, timestamps are RFC3339-compatible time.Time values.

// Board is a Trello board.
type Board struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Desc             string     `json:"desc,omitempty"`
	Closed           bool       `json:"closed"`
	IDOrganization   string     `json:"idOrganization,omitempty"`
	URL              string     `json:"url,omitempty"`
	ShortURL         string     `json:"shortUrl,omitempty"`
	ShortLink        string     `json:"shortLink,omitempty"`
	DateLastActivity *time.Time `json:"dateLastActivity,omitempty"`
}

// List is a Trello list on a board.
type List struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Closed  bool    `json:"closed"`
	Pos     float64 `json:"pos,omitempty"`
	IDBoard string  `json:"idBoard,omitempty"`
}

// Card is a Trello card.
type Card struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Desc             string     `json:"desc,omitempty"`
	Closed           bool       `json:"closed"`
	IDBoard          string     `json:"idBoard,omitempty"`
	IDList           string     `json:"idList,omitempty"`
	Labels           []Label    `json:"labels,omitempty"`
	IDShort          int        `json:"idShort,omitempty"`
	ShortLink        string     `json:"shortLink,omitempty"`
	ShortURL         string     `json:"shortUrl,omitempty"`
	URL              string     `json:"url,omitempty"`
	Pos              float64    `json:"pos,omitempty"`
	Due              *time.Time `json:"due,omitempty"`
	DateLastActivity *time.Time `json:"dateLastActivity,omitempty"`
}

// Member is a Trello user.
type Member struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	Initials  string `json:"initials"`
	URL       string `json:"url,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Bio       string `json:"bio,omitempty"`
	Email     string `json:"email,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

// Organization is a Trello organization (team).
type Organization struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Desc        string `json:"desc,omitempty"`
	URL         string `json:"url,omitempty"`
	Website     string `json:"website,omitempty"`
}

// Label is a Trello label.
type Label struct {
	ID      string `json:"id"`
	IDBoard string `json:"idBoard,omitempty"`
	Name    string `json:"name,omitempty"`
	Color   string `json:"color,omitempty"`
}

// Attachment is a file or URL attached to a card.
type Attachment struct {
	ID       string     `json:"id"`
	Name     string     `json:"name,omitempty"`
	MimeType string     `json:"mimeType,omitempty"`
	Bytes    *int64     `json:"bytes,omitempty"`
	URL      string     `json:"url,omitempty"`
	IsUpload bool       `json:"isUpload"`
	Date     *time.Time `json:"date,omitempty"`
}

// CustomField is a board-level custom field definition.
type CustomField struct {
	ID      string              `json:"id"`
	IDModel string              `json:"idModel,omitempty"`
	Name    string              `json:"name"`
	Type    string              `json:"type"` // checkbox | list | number | text | date
	Options []CustomFieldOption `json:"options,omitempty"`
}

// UnmarshalJSON decodes a custom field. Trello may return name/options at the
// top level or nested under display (legacy vs current shapes); both are
// accepted so the model always exposes Name and Options.
func (f *CustomField) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID      string              `json:"id"`
		IDModel string              `json:"idModel"`
		Name    string              `json:"name"`
		Type    string              `json:"type"`
		Options []CustomFieldOption `json:"options"`
		Display *struct {
			Name    string              `json:"name"`
			Options []CustomFieldOption `json:"options"`
		} `json:"display"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.ID = raw.ID
	f.IDModel = raw.IDModel
	f.Type = raw.Type
	f.Name = raw.Name
	f.Options = raw.Options
	if f.Name == "" && raw.Display != nil {
		f.Name = raw.Display.Name
	}
	if len(f.Options) == 0 && raw.Display != nil {
		f.Options = raw.Display.Options
	}
	return nil
}

// CustomFieldOption is one selectable option of a list-type custom field.
type CustomFieldOption struct {
	ID    string                `json:"id"`
	Value CustomFieldOptionText `json:"value"`
}

// CustomFieldOptionText is the display text of a custom field option.
type CustomFieldOptionText struct {
	Text string `json:"text"`
}

// Checklist is a Trello checklist.
type Checklist struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	IDBoard    string      `json:"idBoard,omitempty"`
	IDCard     string      `json:"idCard,omitempty"`
	CheckItems []CheckItem `json:"checkItems,omitempty"`
}

// CheckItem is an item within a checklist.
type CheckItem struct {
	ID          string  `json:"id"`
	IDChecklist string  `json:"idChecklist,omitempty"`
	Name        string  `json:"name"`
	State       string  `json:"state,omitempty"` // "complete" | "incomplete"
	Pos         float64 `json:"pos,omitempty"`
}

// Action is a Trello action (e.g. commentCard, updateCard, createCard). The
// same type backs both the comment and action commands.
type Action struct {
	ID              string     `json:"id"`
	IDMemberCreator string     `json:"idMemberCreator,omitempty"`
	Type            string     `json:"type,omitempty"`
	Date            *time.Time `json:"date,omitempty"`
	Data            ActionData `json:"data,omitempty"`
	MemberCreator   *Member    `json:"memberCreator,omitempty"`
}

// ActionData holds the contextual objects referenced by an Action.
type ActionData struct {
	Text  string    `json:"text,omitempty"`
	Card  *CardRef  `json:"card,omitempty"`
	Board *BoardRef `json:"board,omitempty"`
	List  *ListRef  `json:"list,omitempty"`
}

// Notification is a Trello notification for a member.
type Notification struct {
	ID     string           `json:"id"`
	Type   string           `json:"type"`
	Date   *time.Time       `json:"date,omitempty"`
	Unread bool             `json:"unread"`
	Data   NotificationData `json:"data,omitempty"`
}

// NotificationData holds the contextual objects referenced by a notification.
type NotificationData struct {
	Text  string    `json:"text,omitempty"`
	Card  *CardRef  `json:"card,omitempty"`
	Board *BoardRef `json:"board,omitempty"`
	List  *ListRef  `json:"list,omitempty"`
}

// CardRef is a minimal card reference embedded in actions.
type CardRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IDShort   int    `json:"idShort,omitempty"`
	ShortLink string `json:"shortLink,omitempty"`
}

// BoardRef is a minimal board reference embedded in actions.
type BoardRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortLink string `json:"shortLink,omitempty"`
}

// ListRef is a minimal list reference embedded in actions.
type ListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
