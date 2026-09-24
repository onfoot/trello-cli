package trello

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// attachmentFields is the minimal field set requested for attachments.
const attachmentFields = "id,name,bytes,date,isUpload,mimeType,url"

// ListCardAttachments returns the attachments on a card.
func (c *Client) ListCardAttachments(ctx context.Context, cardRef string) ([]Attachment, error) {
	var atts []Attachment
	q := url.Values{"fields": {attachmentFields}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/attachments", q, nil, &atts); err != nil {
		return nil, err
	}
	return atts, nil
}

// GetAttachment returns a single attachment by id.
func (c *Client) GetAttachment(ctx context.Context, cardRef, id string) (*Attachment, error) {
	var att Attachment
	q := url.Values{"fields": {attachmentFields}}
	if err := c.Do(ctx, http.MethodGet, "/cards/"+cardRef+"/attachments/"+id, q, nil, &att); err != nil {
		return nil, err
	}
	return &att, nil
}

// AddAttachmentByURL attaches a remote file by URL.
func (c *Client) AddAttachmentByURL(ctx context.Context, cardRef, attachURL, name string) (*Attachment, error) {
	q := url.Values{"url": {attachURL}}
	if name != "" {
		q.Set("name", name)
	}
	var att Attachment
	if err := c.Do(ctx, http.MethodPost, "/cards/"+cardRef+"/attachments", q, nil, &att); err != nil {
		return nil, err
	}
	return &att, nil
}

// UploadAttachment attaches a local file via multipart/form-data. The file is
// streamed as the "file" part; name is an optional "name" form field.
func (c *Client) UploadAttachment(ctx context.Context, cardRef, filePath, name string) (*Attachment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening attachment file %q: %w", filePath, err)
	}
	defer file.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if name != "" {
		if err := mw.WriteField("name", name); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	var att Attachment
	if err := c.do(ctx, http.MethodPost, "/cards/"+cardRef+"/attachments", nil, &buf, mw.FormDataContentType(), &att); err != nil {
		return nil, err
	}
	return &att, nil
}

// DeleteAttachment removes an attachment from a card.
func (c *Client) DeleteAttachment(ctx context.Context, cardRef, id string) error {
	return c.Do(ctx, http.MethodDelete, "/cards/"+cardRef+"/attachments/"+id, nil, nil, nil)
}
