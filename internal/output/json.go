/**
 * package output — JSONRenderer
 *
 * <purpose-start>
 * Renders quota data as a JSON array, one object per Quota record. Nil pointer
 * fields are encoded as JSON null. The output is indented for readability.
 * This renderer is selected when the user passes the --json flag. It is
 * designed to be pipe-friendly (e.g. `tkncap show --json | jq .`).
 * <purpose-end>
 *
 * <inputs-start>
 * - w io.Writer: destination for the JSON output (typically os.Stdout).
 * - quotas []provider.Quota: the quota records to serialise.
 * <inputs-end>
 *
 * <outputs-start>
 * - error: non-nil if JSON encoding or writing fails.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Writes JSON to w.
 * - Logs the record count at debug level.
 * <side-effects-end>
 */
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/hieropold/tkncap/internal/provider"
)

// quotaJSON is the JSON representation of a single Quota record.
// It uses string types for Provider and Status so the JSON output is
// self-documenting (e.g. "claude" rather than an opaque integer).
type quotaJSON struct {
	Provider string     `json:"provider"`
	Account  string     `json:"account"`
	Status   string     `json:"status"`
	Used     *int64     `json:"used"`
	Limit    *int64     `json:"limit"`
	ResetsAt *time.Time `json:"resets_at"`
	Message  string     `json:"message,omitempty"`
}

// JSONRenderer writes quota data as a JSON array.
type JSONRenderer struct{}

/**
 * Render
 *
 * <purpose-start>
 * Converts each Quota to a quotaJSON struct and serialises the resulting slice
 * as an indented JSON array. An empty quotas slice produces "[]". Writes a
 * trailing newline after the JSON so the output is shell-friendly.
 * <purpose-end>
 *
 * <inputs-start>
 * - w io.Writer: destination writer.
 * - quotas []provider.Quota: records to serialise.
 * <inputs-end>
 *
 * <outputs-start>
 * - error: encoding or write error, nil on success.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Writes JSON to w.
 * - Logs the record count at debug level.
 * <side-effects-end>
 */
func (j *JSONRenderer) Render(w io.Writer, quotas []provider.Quota) error {
	slog.Debug("json: rendering quota JSON", "records", len(quotas))

	records := make([]quotaJSON, 0, len(quotas))
	for _, q := range quotas {
		records = append(records, quotaJSON{
			Provider: string(q.Account.Provider),
			Account:  q.Account.Name,
			Status:   string(q.Status),
			Used:     q.Used,
			Limit:    q.Limit,
			ResetsAt: q.ResetsAt,
			Message:  q.Message,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		return fmt.Errorf("json: encode: %w", err)
	}

	slog.Debug("json: render complete")
	return nil
}
