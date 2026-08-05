// Package output defines the Renderer interface implemented by each output
// format (table, JSON). cmd/show.go selects the concrete implementation
// based on the --json flag.
package output

import (
	"io"

	"github.com/hieropold/tkncap/internal/provider"
)

// Renderer is implemented by each output format (table, JSON, etc.).
// Implementations must handle an empty quotas slice gracefully.
type Renderer interface {
	Render(w io.Writer, quotas []provider.Quota) error
}
