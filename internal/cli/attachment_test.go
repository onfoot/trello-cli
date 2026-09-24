package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const testAttachmentsFixture = `[
	{"id":"5bc79d4206526d2279c1e6ea","name":"Notice","mimeType":"application/pdf","bytes":52845,"url":"https://example.com/notice.pdf","isUpload":false},
	{"id":"5bc79d4206526d2279c1e6eb","name":"Image","mimeType":"image/png","bytes":1234,"url":"https://example.com/img.png","isUpload":true}
]`

func TestAttachmentList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, testAttachmentsFixture)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"attachment", "list", "c1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	for _, want := range []string{"Notice", "Image", "application/pdf", "52845"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q: %q", want, stdout)
		}
	}
}

func TestAttachmentGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments/a1" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"a1","name":"Notice","mimeType":"application/pdf","bytes":52845,"url":"https://example.com/n.pdf"}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"attachment", "get", "c1", "a1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Notice") || !strings.Contains(stdout, "52845") {
		t.Errorf("output missing attachment data: %q", stdout)
	}
}

func TestAttachmentAddByURL(t *testing.T) {
	var gotURL, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotURL = r.URL.Query().Get("url")
		gotName = r.URL.Query().Get("name")
		fmt.Fprint(w, `{"id":"a1","name":"doc.pdf","url":"https://example.com/doc.pdf","isUpload":false}`)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"attachment", "add", "c1", "--url", "https://example.com/doc.pdf", "--name", "doc.pdf"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotURL != "https://example.com/doc.pdf" || gotName != "doc.pdf" {
		t.Errorf("query params: url=%q name=%q", gotURL, gotName)
	}
	if !strings.Contains(stdout, "a1") {
		t.Errorf("output should include the new attachment id: %q", stdout)
	}
}

func TestAttachmentAddByFile(t *testing.T) {
	var gotFilename, gotFileContent, gotNameField string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
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
				buf := make([]byte, 64)
				n, _ := f.Read(buf)
				gotFileContent = string(buf[:n])
				f.Close()
			}
		}
		if vals := r.MultipartForm.Value["name"]; len(vals) > 0 {
			gotNameField = vals[0]
		}
		fmt.Fprint(w, `{"id":"a1","name":"notes.txt","mimeType":"text/plain","bytes":11,"isUpload":true}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runCLI(t, []string{"attachment", "add", "c1", "--file", path, "--name", "notes.txt"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if gotFilename != "notes.txt" || gotFileContent != "hello world" || gotNameField != "notes.txt" {
		t.Errorf("multipart: filename=%q content=%q name=%q", gotFilename, gotFileContent, gotNameField)
	}
	if !strings.Contains(stdout, "a1") {
		t.Errorf("output should include the new attachment id: %q", stdout)
	}
}

func TestAttachmentAddRequiresURLOrFile(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"attachment", "add", "c1"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "--url") || !strings.Contains(stderr, "--file") {
		t.Errorf("stderr should mention --url and --file: %q", stderr)
	}
}

func TestAttachmentAddRejectsBothURLAndFile(t *testing.T) {
	code, _, _ := runCLI(t, []string{"attachment", "add", "c1", "--url", "https://example.com/x", "--file", "/tmp/x"}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
}

func TestAttachmentAddMissingFile(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"attachment", "add", "c1", "--file", filepath.Join(t.TempDir(), "nope.txt")}, strings.NewReader(""), credsEnv(t), "")
	if code != output.ExitError {
		t.Errorf("exit = %d, want %d (runtime error)", code, output.ExitError)
	}
	if !strings.Contains(stderr, "nope.txt") {
		t.Errorf("stderr should mention the missing file: %q", stderr)
	}
}

func TestAttachmentDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/c1/attachments/a1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, stdout, _ := runCLI(t, []string{"attachment", "delete", "c1", "a1"}, strings.NewReader(""), credsEnv(t), srv.URL)
	if code != output.ExitOK {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "a1") {
		t.Errorf("output should mention the deleted attachment: %q", stdout)
	}
}

func TestAttachmentUnknownSubcommandExitUsage(t *testing.T) {
	code, _, stderr := runCLI(t, []string{"attachment", "bogus"}, strings.NewReader(""), testEnv(t, ""), "")
	if code != output.ExitUsage {
		t.Errorf("exit = %d, want %d", code, output.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention the unknown command: %q", stderr)
	}
}
