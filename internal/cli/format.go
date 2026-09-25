package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nomadicworks/trello-cli/internal/trello"
)

// formatPos renders a position without trailing zeros.
func formatPos(pos float64) string {
	return strconv.FormatFloat(pos, 'f', -1, 64)
}

// formatDue renders a due timestamp in RFC3339, or "-" when unset.
func formatDue(d *time.Time) string {
	if d == nil {
		return "-"
	}
	return d.Format(time.RFC3339)
}

// formatTimePtr renders an optional timestamp in RFC3339, or "-" when unset.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
}

// formatBytes renders an attachment size, or "-" when unknown.
func formatBytes(b *int64) string {
	if b == nil {
		return "-"
	}
	return strconv.FormatInt(*b, 10)
}

// formatCardLabels renders a card's labels for human output, or "-" when the
// card has none. Each label renders as "Name (color)" when both are set,
// "Name" or "(color)" when only one is, and the label id otherwise.
func formatCardLabels(labels []trello.Label) string {
	if len(labels) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		switch {
		case l.Name != "" && l.Color != "":
			parts = append(parts, fmt.Sprintf("%s (%s)", l.Name, l.Color))
		case l.Name != "":
			parts = append(parts, l.Name)
		case l.Color != "":
			parts = append(parts, "("+l.Color+")")
		default:
			parts = append(parts, l.ID)
		}
	}
	return strings.Join(parts, ", ")
}
