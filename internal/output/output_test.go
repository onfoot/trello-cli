package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, &buf, false, false)
	err := w.Table([]string{"ID", "NAME"}, [][]string{
		{"abc123", "Alpha"},
		{"abcdef0123456789abcdef", "Beta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	// The NAME column must start at the same offset on every line.
	wantCol := strings.Index(lines[0], "NAME")
	if wantCol < 0 {
		t.Fatalf("header missing NAME: %q", lines[0])
	}
	for _, line := range lines[1:] {
		if strings.Index(line, "Beta") != wantCol && strings.Index(line, "Alpha") != wantCol {
			t.Errorf("data column not aligned with header: %q", line)
		}
	}
}

func TestJSONOutDeterministic(t *testing.T) {
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	v := []item{{ID: "abc", Name: "Alpha"}, {ID: "def", Name: "Beta"}}

	var a, b bytes.Buffer
	wa := New(&a, &bytes.Buffer{}, true, false)
	wb := New(&b, &bytes.Buffer{}, true, false)
	if err := wa.JSONOut(v); err != nil {
		t.Fatal(err)
	}
	if err := wb.JSONOut(v); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Errorf("JSON output not deterministic:\n%s\n%s", a.String(), b.String())
	}
	if !strings.HasPrefix(a.String(), "[\n  {") {
		t.Errorf("expected indented JSON array, got: %q", a.String())
	}
	if !strings.Contains(a.String(), "\"id\": \"abc\"") {
		t.Errorf("expected raw string ID, got: %q", a.String())
	}
}

func TestKeyValue(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, &buf, false, false)
	if err := w.KeyValue([][2]string{{"Username", "bob"}, {"ID", "abc123"}}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "Username:") || !strings.Contains(got, "bob") {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestColorEnabled(t *testing.T) {
	// A non-terminal writer never enables color.
	if ColorEnabled(false, &bytes.Buffer{}) {
		t.Error("ColorEnabled should be false for a buffer")
	}
	// --no-color forces color off.
	if ColorEnabled(true, &bytes.Buffer{}) {
		t.Error("ColorEnabled should be false with noColor")
	}
}

func TestBoldRespectsColor(t *testing.T) {
	plain := New(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	if got := plain.Bold("x"); got != "x" {
		t.Errorf("Bold without color = %q, want %q", got, "x")
	}
	colored := New(&bytes.Buffer{}, &bytes.Buffer{}, false, true)
	if got := colored.Bold("x"); !strings.Contains(got, "\x1b[") {
		t.Errorf("Bold with color should emit ANSI, got %q", got)
	}
}
