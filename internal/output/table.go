/**
 * package output — TableRenderer
 *
 * <purpose-start>
 * Renders quota data as a human-readable table using text/tabwriter. Column
 * headers are printed on the first line; each Quota occupies one row. Nil
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
	"fmt"
	"io"
	"log/slog"
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
 * Formats each Quota as a table row with columns: PROVIDER, ACCOUNT, STATUS,
 * USED, LIMIT, RESETS_AT. Nil pointer fields become "-". Uses text/tabwriter
 * with tab-padded alignment and a minimum column width of 1 and tab width of 8.
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

	tw := tabwriter.NewWriter(w, 1, 8, 2, ' ', 0)

	// Header row.
	if _, err := fmt.Fprintln(tw, "PROVIDER\tACCOUNT\tWINDOW\tSTATUS\tUSED\tLIMIT\tRESETS_AT\tRESETS_IN"); err != nil {
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

		row := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			q.Account.Provider,
			q.Account.Name,
			window,
			q.Status,
			used,
			limit,
			resetsAt,
			resetsIn,
		)
		if _, err := fmt.Fprintln(tw, row); err != nil {
			return fmt.Errorf("table: write row for %s/%s: %w", q.Account.Provider, q.Account.Name, err)
		}
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("table: flush: %w", err)
	}

	slog.Debug("table: render complete")
	return nil
}
