/**
 * package output
 *
 * <purpose-start>
 * Defines the Renderer interface that all output format implementations must
 * satisfy. The interface is intentionally minimal: given a writer and a slice
 * of Quota values, render them. Callers (cmd/show.go) select the concrete
 * implementation based on the --json flag and write to os.Stdout.
 * <purpose-end>
 *
 * <inputs-start>
 * - N/A (package definition).
 * <inputs-end>
 *
 * <outputs-start>
 * - N/A (package definition).
 * <outputs-end>
 *
 * <side-effects-start>
 * - None.
 * <side-effects-end>
 */
package output

import (
	"io"

	"github.com/hieropold/tkncap/internal/provider"
)

/**
 * Renderer
 *
 * <purpose-start>
 * Interface satisfied by all output format implementations (table, JSON, etc.).
 * Render writes the quota data to w and returns an error if writing fails.
 * The quotas slice may be empty; renderers should handle that case gracefully.
 * <purpose-end>
 *
 * <inputs-start>
 * - N/A (interface definition).
 * <inputs-end>
 *
 * <outputs-start>
 * - N/A (interface definition).
 * <outputs-end>
 *
 * <side-effects-start>
 * - None (interface contract; concrete implementations write to w).
 * <side-effects-end>
 */
type Renderer interface {
	Render(w io.Writer, quotas []provider.Quota) error
}
