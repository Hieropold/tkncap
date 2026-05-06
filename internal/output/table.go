/**
 * package output — TableRenderer
 *
 * <purpose-start>
 * Renders quota data as a human-readable table using text/tabwriter. Column
 * headers are printed on the first line; each Quota occupies one row. Rows are
 * color-coded from green to red based on the percentage of USED quota. Nil
 * pointer fields (Used, Limit, ResetsAt) are rendered as "-" to indicate the
 * value is unavailable. The table is written to the provided io.Writer and
 * flushed before returning. No external dependencies beyond the standard
 * library are used.
 * <purpose-end>
 *
 * <inputs-start>
 * - w io.Writer: destination for the rendered table (typically os.Stdout).
 * - quotas []provider.Quota: the quota records to render; may be empty.
 * <inputs-end>
 *
 * <outputs-start>
 * - error: non-nil if writing to w or flushing tabwriter fails.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Writes to w.
 * - Logs each rendered row at debug level.
 * <side-effects-end>
 */
package output

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hieropold/tkncap/internal/provider"
)

// TableRenderer writes quota data as tab-aligned columns.
type TableRenderer struct{}

/**
 * Render
 *
 * <purpose-start>
 * Formats each Quota as a table row with columns: PROVIDER, ACCOUNT, WINDOW,
 * STATUS, USED, LIMIT, RESETS_AT, RESETS_IN, MESSAGE. Nil pointer fields 
 * become "-". Uses text/tabwriter to align columns in an intermediate buffer 
 * before applying ANSI color sequences to entire rows based on quota 
 * utilization. This ensures perfect alignment regardless of escape sequences.
 * <purpose-end>
 *
 * <inputs-start>
 * - w io.Writer: destination writer.
 * - quotas []provider.Quota: records to render.
 * <inputs-end>
 *
 * <outputs-start>
 * - error: write or flush error, nil on success.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Writes formatted table to w.
 * - Logs row count at debug level.
 * <side-effects-end>
 */
func (t *TableRenderer) Render(w io.Writer, quotas []provider.Quota) error {
	slog.Debug("table: rendering quota table", "rows", len(quotas))

	// Use a buffer to collect aligned rows without colors first.
	// This ensures tabwriter calculates widths correctly.
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 1, 8, 2, ' ', 0)

	// Header row.
	if _, err := fmt.Fprintln(tw, "PROVIDER\tACCOUNT\tWINDOW\tSTATUS\tUSED\tLIMIT\tRESETS_AT\tRESETS_IN\tMESSAGE"); err != nil {
		return fmt.Errorf("table: write header: %w", err)
	}

	now := time.Now()
	for _, q := range quotas {
		used := "-"
		if q.Used != nil {
			used = fmt.Sprintf("%d", *q.Used)
		}
		limit := "-"
		if q.Limit != nil {
			limit = fmt.Sprintf("%d", *q.Limit)
		}
		resetsAt := "-"
		resetsIn := "-"
		if q.ResetsAt != nil {
			resetsAt = q.ResetsAt.Format("2006-01-02T15:04:05Z07:00")
			hours := q.ResetsAt.Sub(now).Hours()
			if hours < 0 {
				hours = 0
			}
			resetsIn = fmt.Sprintf("%.1fh", hours)
		}

		window := q.Name
		if window == "" {
			window = "-"
		}

		slog.Debug("table: rendering row",
			"provider", q.Account.Provider,
			"account", q.Account.Name,
			"window", window,
			"status", q.Status,
		)

		row := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			q.Account.Provider,
			q.Account.Name,
			window,
			q.Status,
			used,
			limit,
			resetsAt,
			resetsIn,
			q.Message,
		)
		if _, err := fmt.Fprintln(tw, row); err != nil {
			return fmt.Errorf("table: write row for %s/%s: %w", q.Account.Provider, q.Account.Name, err)
		}
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("table: flush: %w", err)
	}

	// Now read the aligned lines and apply colors.
	output := buf.String()
	if output == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	// Write header.
	if _, err := fmt.Fprintln(w, lines[0]); err != nil {
		return err
	}

	// Write data rows with colors.
	for i, q := range quotas {
		if i+1 >= len(lines) {
			break
		}

		colorPrefix := ""
		colorSuffix := ""
		if q.Used != nil && q.Limit != nil && *q.Limit > 0 {
			pct := (float64(*q.Used) / float64(*q.Limit)) * 100
			switch {
			case pct <= 50:
				colorPrefix = "\x1b[38;5;46m" // Green
			case pct <= 60:
				colorPrefix = "\x1b[38;5;154m" // Yellow-Green
			case pct <= 70:
				colorPrefix = "\x1b[38;5;226m" // Yellow
			case pct <= 80:
				colorPrefix = "\x1b[38;5;214m" // Orange
			case pct <= 90:
				colorPrefix = "\x1b[38;5;202m" // Red-Orange
			default:
				colorPrefix = "\x1b[38;5;196m" // Red
			}
			colorSuffix = "\x1b[0m" // Reset
		}

		if _, err := fmt.Fprintf(w, "%s%s%s\n", colorPrefix, lines[i+1], colorSuffix); err != nil {
			return err
		}
	}

	slog.Debug("table: render complete")
	return nil
}
