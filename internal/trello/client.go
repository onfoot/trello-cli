// Package trello is a hand-written, typed client for the Phase-1 subset of
// the Trello REST API. All requests flow through Client.Do, which injects the
// key/token query params, sets Accept, checks the status, decodes JSON, and
// maps HTTP statuses to typed errors carrying exit codes.
package trello

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/nomadicworks/trello-cli/internal/output"
)

// DefaultBaseURL is the Trello REST API root.
const DefaultBaseURL = "https://api.trello.com/1"

// maxErrorBody caps how much of an error response body is retained.
const maxErrorBody = 1 << 20 // 1 MiB

// HTTPClient is the transport seam used by Client so tests can inject an
// httptest server or a fake transport. *http.Client satisfies it.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a Trello API client. Key and Token are injected as query params
// on every request.
type Client struct {
	Key     string
	Token   string
	BaseURL string
	HTTP    HTTPClient
}

// New returns a Client. An empty baseURL selects DefaultBaseURL.
func New(key, token, baseURL string, httpClient HTTPClient) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{Key: key, Token: token, BaseURL: baseURL, HTTP: httpClient}
}

// Do performs a single JSON API request: it builds the URL from baseURL+path,
// injects the key/token query params, sets Accept: application/json, executes
// the request with ctx, checks the status, and decodes a JSON response into
// out (when out is non-nil). body is optional and, when non-nil, is sent with
// Content-Type: application/json.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	return c.do(ctx, method, path, query, body, "application/json", out)
}

// do performs a single API request. contentType is the request Content-Type,
// applied only when body is non-nil; pass "" to leave the header unset.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader, contentType string, out any) error {
	if c.HTTP == nil {
		return &NetworkError{Op: "request", Cause: errors.New("no HTTP client configured")}
	}
	if query == nil {
		query = url.Values{}
	}

	// Merge any query string embedded in path with the caller's query values
	// so credentials are never swallowed by a path like "/cards/c1?fields=labels".
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return &NetworkError{Op: "request", Cause: err}
	}
	merged := u.Query()
	for k, vs := range query {
		for _, v := range vs {
			merged.Add(k, v)
		}
	}
	// Credentials win over any user-supplied values.
	if c.Key != "" {
		merged.Set("key", c.Key)
	}
	if c.Token != "" {
		merged.Set("token", c.Token)
	}
	u.RawQuery = merged.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return &NetworkError{Op: "request", Cause: err}
	}
	req.Header.Set("Accept", "application/json")
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return classifyNetworkError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if err != nil {
		return &NetworkError{Op: "read response body", Cause: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newStatusError(resp.StatusCode, resp.Status, respBody)
	}

	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(respBody)) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return &DecodeError{Path: path, Err: err}
	}
	return nil
}

// HTTPError is a non-2xx API response. Body holds the (possibly truncated)
// response body; Code and Message are populated when the body matches the
// Trello error schema {"code","message"}.
type HTTPError struct {
	StatusCode int    `json:"-"`
	Status     string `json:"-"`
	Body       string `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *HTTPError) Error() string {
	switch {
	case e.Message != "":
		return fmt.Sprintf("HTTP %d (%s): %s", e.StatusCode, e.Status, e.Message)
	case e.Body != "":
		return fmt.Sprintf("HTTP %d (%s): %s", e.StatusCode, e.Status, e.Body)
	default:
		return fmt.Sprintf("HTTP %d (%s)", e.StatusCode, e.Status)
	}
}

// ExitCode maps an unmapped API error to the generic runtime exit code.
func (e *HTTPError) ExitCode() int { return output.ExitError }

// AuthError is an HTTP 401 response.
type AuthError struct{ *HTTPError }

// ExitCode reports the authentication-failure exit code.
func (e *AuthError) ExitCode() int { return output.ExitAuth }

// NotFoundError is an HTTP 404 response.
type NotFoundError struct{ *HTTPError }

// ExitCode reports the not-found exit code.
func (e *NotFoundError) ExitCode() int { return output.ExitNotFound }

// ValidationError is an HTTP 400 or 422 response.
type ValidationError struct{ *HTTPError }

// ExitCode reports the validation/conflict exit code.
func (e *ValidationError) ExitCode() int { return output.ExitValidation }

// RateLimitError is an HTTP 429 response.
type RateLimitError struct{ *HTTPError }

// ExitCode reports the rate-limit exit code.
func (e *RateLimitError) ExitCode() int { return output.ExitRateLimit }

// NetworkError wraps a transport-level failure (DNS, TLS, timeout, reset,
// EOF) that prevented a response from being received.
type NetworkError struct {
	Op      string
	Cause   error
	Timeout bool
}

func (e *NetworkError) Error() string {
	kind := "connection error"
	if e.Timeout {
		kind = "timeout"
	}
	return fmt.Sprintf("%s (%s): %v", kind, e.Op, e.Cause)
}

// Unwrap exposes the underlying cause.
func (e *NetworkError) Unwrap() error { return e.Cause }

// ExitCode reports the network-error exit code.
func (e *NetworkError) ExitCode() int { return output.ExitNetwork }

// DecodeError is a failure to parse a successful response body.
type DecodeError struct {
	Path string
	Err  error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("failed to decode response from %s: %v", e.Path, e.Err)
}

// Unwrap exposes the underlying parse error.
func (e *DecodeError) Unwrap() error { return e.Err }

// ExitCode reports the generic runtime exit code.
func (e *DecodeError) ExitCode() int { return output.ExitError }

// BoardNotFoundError is returned when a board reference matches no board.
type BoardNotFoundError struct {
	Ref string
}

func (e *BoardNotFoundError) Error() string {
	return fmt.Sprintf("board %q not found among your boards (matched by exact name, id, or shortLink); run 'trello board list' to see your boards", e.Ref)
}

// ExitCode reports the config exit code: an explicit board that matches
// nothing is a configuration error, never a silent fallback.
func (e *BoardNotFoundError) ExitCode() int { return output.ExitConfig }

// classifyNetworkError wraps a transport error, flagging timeouts so the CLI
// can surface a clear "timeout" message.
func classifyNetworkError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &NetworkError{Op: "request", Cause: err, Timeout: true}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &NetworkError{Op: "request", Cause: err, Timeout: true}
	}
	return &NetworkError{Op: "request", Cause: err}
}

// newStatusError maps an HTTP status to a typed error. The Trello error
// schema is parsed best-effort so the message is surfaced when available.
func newStatusError(statusCode int, status string, body []byte) error {
	e := &HTTPError{
		StatusCode: statusCode,
		Status:     status,
		Body:       strings.TrimSpace(string(body)),
	}
	var apiErr HTTPError
	if perr := json.Unmarshal(body, &apiErr); perr == nil && apiErr.Code != "" {
		e.Code = apiErr.Code
		e.Message = apiErr.Message
	}
	switch statusCode {
	case http.StatusUnauthorized:
		return &AuthError{HTTPError: e}
	case http.StatusNotFound:
		return &NotFoundError{HTTPError: e}
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return &ValidationError{HTTPError: e}
	case http.StatusTooManyRequests:
		return &RateLimitError{HTTPError: e}
	default:
		return e
	}
}

// GetMemberMe validates credentials and returns the authenticated member.
func (c *Client) GetMemberMe(ctx context.Context) (*Member, error) {
	var m Member
	q := url.Values{"fields": {"id,username,fullName,initials,url,avatarUrl,bio,email,confirmed"}}
	if err := c.Do(ctx, http.MethodGet, "/members/me", q, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMemberBoards returns the boards the authenticated member belongs to.
func (c *Client) ListMemberBoards(ctx context.Context) ([]Board, error) {
	var boards []Board
	q := url.Values{"fields": {"id,name,shortLink"}}
	if err := c.Do(ctx, http.MethodGet, "/members/me/boards", q, nil, &boards); err != nil {
		return nil, err
	}
	return boards, nil
}

// ResolveBoard matches ref against the member's boards by exact name, id, or
// shortLink. Matching is exact — no prefix or substring guessing. A reference
// that matches nothing returns a BoardNotFoundError.
func (c *Client) ResolveBoard(ctx context.Context, ref string) (*Board, error) {
	boards, err := c.ListMemberBoards(ctx)
	if err != nil {
		return nil, err
	}
	for i := range boards {
		b := &boards[i]
		if b.ID == ref || b.ShortLink == ref || b.Name == ref {
			return b, nil
		}
	}
	return nil, &BoardNotFoundError{Ref: ref}
}
