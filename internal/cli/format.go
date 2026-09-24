package cli

import (
	"strconv"
	"time"
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
