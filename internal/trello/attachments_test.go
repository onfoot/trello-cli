package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCardAttachments(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/cards/c1/attachments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotFields = r.URL.Query().Get("fields")
		fmt.Fprint(w, `[
			{"id":"a1","name":"Notice","mimeType":"application/pdf","bytes":52845,"url":"https://example.com/n.pdf","isUpload":false},
			{"id":"a2","name":"Image","mimeType":"image/png","bytes":1234,"url":"https://example.com/i.png","isUpload":true}
		]`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	atts, err := c.ListCardAttachments(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 2 || atts[0].Name != "Notice" || atts[1].MimeType != "image/png" {
		t.Errorf("unexpected attachments: %+v", atts)
	}
	if atts[0].Bytes == nil || *atts[0].Bytes != 52845 {
		t.Errorf("unexpected bytes: %v", atts[0].Bytes)
	}
	if gotFields != "id,name,bytes,date,isUpload,mimeType,url" {
		t.Errorf("fields = %q", gotFields)
	}
}

func TestGetAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"a1","name":"Notice","mimeType":"application/pdf","bytes":52845,"url":"https://example.com/n.pdf"}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	att, err := c.GetAttachment(context.Background(), "c1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if att.ID != "a1" || att.Name != "Notice" || att.URL != "https://example.com/n.pdf" {
		t.Errorf("unexpected attachment: %+v", att)
	}
}

func TestAddAttachmentByURL(t *testing.T) {
	var gotURL, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/c1/attachments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotURL = r.URL.Query().Get("url")
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"a1","name":"doc.pdf","url":"https://example.com/doc.pdf","isUpload":false}`)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	att, err := c.AddAttachmentByURL(context.Background(), "c1", "https://example.com/doc.pdf", "doc.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if att.ID != "a1" {
		t.Errorf("unexpected attachment: %+v", att)
	}
	if gotURL != "https://example.com/doc.pdf" || gotName != "doc.pdf" {
		t.Errorf("query params: url=%q name=%q", gotURL, gotName)
	}
}

func TestUploadAttachment(t *testing.T) {
	var gotFilename, gotFileContent, gotNameField, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/c1/attachments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotContentType = r.Header.Get("Content-Type")
		if !strings.HasPrefix(gotContentType, "multipart/form-data; boundary=") {
			t.Errorf("Content-Type = %q, want multipart/form-data", gotContentType)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		files := r.MultipartForm.File["file"]
		if len(files) != 1 {
			t.Errorf("expected 1 file part, got %d", len(files))
		} else {
			gotFilename = files[0].Filename
			f, err := files[0].Open()
			if err != nil {
				t.Errorf("open file part: %v", err)
			} else {
				buf := make([]byte, 32)
				n, _ := f.Read(buf)
				gotFileContent = string(buf[:n])
				f.Close()
			}
		}
		gotNameField = r.MultipartForm.Value["name"][0]
		fmt.Fprint(w, `{"id":"a1","name":"notes.txt","mimeType":"text/plain","bytes":11,"isUpload":true}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New("k", "t", srv.URL, srv.Client())
	att, err := c.UploadAttachment(context.Background(), "c1", path, "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if att.ID != "a1" || !att.IsUpload {
		t.Errorf("unexpected attachment: %+v", att)
	}
	if gotFilename != "notes.txt" {
		t.Errorf("file part filename = %q, want notes.txt", gotFilename)
	}
	if gotFileContent != "hello world" {
		t.Errorf("file part content = %q, want hello world", gotFileContent)
	}
	if gotNameField != "notes.txt" {
		t.Errorf("name field = %q, want notes.txt", gotNameField)
	}
}

func TestUploadAttachmentMissingFile(t *testing.T) {
	c := New("k", "t", "http://example.invalid", &fakeHTTPClient{fn: func(req *http.Request) (*http.Response, error) {
		t.Error("request should not be sent when the file is missing")
		return nil, fmt.Errorf("unexpected")
	}})
	_, err := c.UploadAttachment(context.Background(), "c1", filepath.Join(t.TempDir(), "nope.txt"), "")
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !strings.Contains(err.Error(), "nope.txt") {
		t.Errorf("error should mention the file path: %v", err)
	}
}

func TestDeleteAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/cards/c1/attachments/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New("k", "t", srv.URL, srv.Client())
	if err := c.DeleteAttachment(context.Background(), "c1", "a1"); err != nil {
		t.Fatal(err)
	}
}

func TestMultipartContentType(t *testing.T) {
	// The multipart writer's content type carries a boundary and must be used
	// verbatim on the request.
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New("k", "t", srv.URL, srv.Client())
	if _, err := c.UploadAttachment(context.Background(), "c1", path, ""); err != nil {
		t.Fatal(err)
	}
	// boundary must be present (non-empty)
	if !strings.Contains(gotHeader, "boundary=") {
		t.Errorf("Content-Type = %q, want a multipart boundary", gotHeader)
	}
}
