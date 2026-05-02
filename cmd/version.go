/**
 * package cmd — version command
 *
 * <purpose-start>
 * Implements `tkncap version`, which prints the build version, commit hash,
 * and build date. The three variables (version, commit, date) default to
 * development placeholders and are overridden at release time via -ldflags:
 *
 *   go build -ldflags "-X github.com/hieropold/tkncap/cmd.version=1.0.0
 *                       -X github.com/hieropold/tkncap/cmd.commit=$(git rev-parse --short HEAD)
 *                       -X github.com/hieropold/tkncap/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
 * <purpose-end>
 *
 * <inputs-start>
 * - version, commit, date: linker-injected build metadata.
 * <inputs-end>
 *
 * <outputs-start>
 * - Prints "tkncap version <version> (commit: <commit>, built: <date>)" to
 *   os.Stdout.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Writes to os.Stdout.
 * <side-effects-end>
 */
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are populated at link time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the tkncap version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tkncap version %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
