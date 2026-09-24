package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// orgFields is the minimal field set requested for organizations.
const orgFields = "id,name,displayName,desc,url,website"

// OrgNotFoundError is returned when an organization reference matches no
// organization. Matching is exact (id, name, or displayName) — never fuzzy.
type OrgNotFoundError struct {
	Ref string
}

func (e *OrgNotFoundError) Error() string {
	return fmt.Sprintf("organization %q not found (matched by exact id, name, or displayName); run 'trello org list' to see your organizations", e.Ref)
}

// ExitCode reports the config exit code: an explicit organization that
// matches nothing is a configuration error, never a silent fallback.
func (e *OrgNotFoundError) ExitCode() int { return output.ExitConfig }

// ListMyOrganizations returns the authenticated member's organizations.
func (c *Client) ListMyOrganizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	q := url.Values{"fields": {orgFields}}
	if err := c.Do(ctx, http.MethodGet, "/members/me/organizations", q, nil, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

// GetOrganization returns an organization by id or short name.
func (c *Client) GetOrganization(ctx context.Context, ref string) (*Organization, error) {
	var o Organization
	if err := c.Do(ctx, http.MethodGet, "/organizations/"+ref, nil, nil, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// ListOrgBoards returns the boards of an organization.
func (c *Client) ListOrgBoards(ctx context.Context, orgID string) ([]Board, error) {
	var boards []Board
	q := url.Values{"fields": {"id,name,shortLink,closed"}}
	if err := c.Do(ctx, http.MethodGet, "/organizations/"+orgID+"/boards", q, nil, &boards); err != nil {
		return nil, err
	}
	return boards, nil
}

// ListOrgMembers returns the members of an organization.
func (c *Client) ListOrgMembers(ctx context.Context, orgID string) ([]Member, error) {
	var members []Member
	q := url.Values{"fields": {memberFields}}
	if err := c.Do(ctx, http.MethodGet, "/organizations/"+orgID+"/members", q, nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// ResolveOrg resolves a reference to an organization: a 24-hex id is used
// directly; otherwise the member's organizations are listed and matched by
// exact name or displayName. A reference that matches nothing returns an
// OrgNotFoundError.
func (c *Client) ResolveOrg(ctx context.Context, ref string) (*Organization, error) {
	if IsTrelloID(ref) {
		return c.GetOrganization(ctx, ref)
	}
	orgs, err := c.ListMyOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	for i := range orgs {
		o := &orgs[i]
		if o.Name == ref || o.DisplayName == ref {
			return o, nil
		}
	}
	return nil, &OrgNotFoundError{Ref: ref}
}

// IsTrelloID reports whether s looks like a 24-hex Trello id.
func IsTrelloID(s string) bool {
	if len(s) != 24 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
