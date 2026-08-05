// JSONRenderer writes quota data as an indented JSON array (pipe-friendly,
// e.g. `tkncap show --json | jq .`); it is selected when the user passes
// --json.
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
	Name     string     `json:"name,omitempty"`
	Status   string     `json:"status"`
	Used     *int64     `json:"used"`
	Limit    *int64     `json:"limit"`
	ResetsAt *time.Time `json:"resets_at"`
	ResetsIn *float64   `json:"resets_in_hours,omitempty"`
	Message  string     `json:"message,omitempty"`
}

// JSONRenderer writes quota data as a JSON array.
type JSONRenderer struct{}

// Render serialises quotas as an indented JSON array to w (an empty slice
// produces "[]"), with a trailing newline so the output stays shell-friendly.
//
// Side effects: writes to w.
func (j *JSONRenderer) Render(w io.Writer, quotas []provider.Quota) error {
	slog.Debug("json: rendering quota JSON", "records", len(quotas))

	now := time.Now()
	records := make([]quotaJSON, 0, len(quotas))
	for _, q := range quotas {
		var resetsIn *float64
		if q.ResetsAt != nil {
			hours := q.ResetsAt.Sub(now).Hours()
			if hours < 0 {
				hours = 0
			}
			resetsIn = &hours
		}

		records = append(records, quotaJSON{
			Provider: string(q.Account.Provider),
			Account:  q.Account.Name,
			Name:     q.Name,
			Status:   string(q.Status),
			Used:     q.Used,
			Limit:    q.Limit,
			ResetsAt: q.ResetsAt,
			ResetsIn: resetsIn,
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
