package trello

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// notificationFields is the minimal field set requested for notifications.
const notificationFields = "id,type,date,unread,data"

// ListMyNotifications returns the authenticated member's notifications.
// filter is a comma-separated list of notification types, or "all".
func (c *Client) ListMyNotifications(ctx context.Context, filter string) ([]Notification, error) {
	var notifs []Notification
	q := url.Values{"filter": {filter}, "fields": {notificationFields}}
	if err := c.Do(ctx, http.MethodGet, "/members/me/notifications", q, nil, &notifs); err != nil {
		return nil, err
	}
	return notifs, nil
}

// SetNotificationRead marks a notification read (unread=false) or unread.
func (c *Client) SetNotificationRead(ctx context.Context, id string, unread bool) (*Notification, error) {
	q := url.Values{"unread": {strconv.FormatBool(unread)}}
	var n Notification
	if err := c.Do(ctx, http.MethodPut, "/notifications/"+id, q, nil, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// MarkAllNotificationsRead marks all of the member's notifications as read.
func (c *Client) MarkAllNotificationsRead(ctx context.Context) error {
	return c.Do(ctx, http.MethodPost, "/notifications/all/read", nil, nil, nil)
}
