// Package output renders command results in human-readable or JSON form and
// owns the exit-code contract shared by every command.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

// Writer renders results to a destination. JSON toggles between machine
// readable and human output; Color controls ANSI embellishment.
type Writer struct {
	Out   io.Writer
	Err   io.Writer
	JSON  bool
	Color bool
}

// New returns a Writer that emits to out/err.
func New(out, err io.Writer, jsonMode, color bool) *Writer {
	return &Writer{Out: out, Err: err, JSON: jsonMode, Color: color}
}

// ColorEnabled reports whether ANSI color should be used: never with
// --no-color, and only when out is a real terminal.
func ColorEnabled(noColor bool, out io.Writer) bool {
	if noColor {
		return false
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Printf writes formatted output to the primary destination.
func (w *Writer) Printf(format string, args ...any) {
	fmt.Fprintf(w.Out, format, args...)
}

// Errorf writes a formatted message to the error destination.
func (w *Writer) Errorf(format string, args ...any) {
	fmt.Fprintf(w.Err, format, args...)
}

// Warnf writes a warning to the error destination.
func (w *Writer) Warnf(format string, args ...any) {
	fmt.Fprintf(w.Err, "trello: warning: "+format+"\n", args...)
}

// PrintError writes a command error to the error destination.
func (w *Writer) PrintError(err error) {
	prefix := "trello:"
	if w.Color {
		prefix = "\x1b[31mtrello:\x1b[0m"
	}
	w.Errorf("%s %s\n", prefix, err.Error())
}

// Bold wraps s in ANSI bold when color is enabled.
func (w *Writer) Bold(s string) string {
	if !w.Color {
		return s
	}
	return "\x1b[1m" + s + "\x1b[0m"
}

// JSONOut marshals v as indented, stable JSON (struct field order) and writes
// it to the primary destination.
func (w *Writer) JSONOut(v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := w.Out.Write(buf.Bytes())
	return err
}

// Table renders a header row and data rows with aligned columns.
func (w *Writer) Table(headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w.Out, 0, 0, 2, ' ', 0)
	hs := make([]string, len(headers))
	for i, h := range headers {
		hs[i] = w.Bold(h)
	}
	if _, err := fmt.Fprintln(tw, strings.Join(hs, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// KeyValue renders aligned "key: value" lines.
func (w *Writer) KeyValue(rows [][2]string) error {
	tw := tabwriter.NewWriter(w.Out, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s:\t%s\n", w.Bold(row[0]), row[1]); err != nil {
			return err
		}
	}
	return tw.Flush()
}
