// Package cmd defines the root cobra.Command for tkncap and exposes Execute,
// called by main.go. The --log-level flag, when set, overrides the
// TKNCAP_LOG_LEVEL env-var by writing to the environment before
// logging.Setup runs in PersistentPreRunE, so the level applies before any
// subcommand body executes.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hieropold/tkncap/internal/logging"
)

// JSONOutput is set to true when --json is passed; subcommands read this to
// select the appropriate Renderer.
var JSONOutput bool

// logLevelFlag caches the --log-level flag value before logging.Setup runs.
var logLevelFlag string

var rootCmd = &cobra.Command{
	Use:   "tkncap",
	Short: "View current token quota for Claude Code and Gemini accounts",
	Long: `tkncap reports the current token quota and usage for all configured
provider accounts. Accounts are configured via environment variables:

  TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>

Supported providers: claude, gemini

Example:
  TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=~/.claude/.credentials.json \
  TKNCAP_GEMINI_MAIN_API_KEY=AIza... \
  tkncap show`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if logLevelFlag != "" {
			if err := os.Setenv("TKNCAP_LOG_LEVEL", logLevelFlag); err != nil {
				return fmt.Errorf("root: set TKNCAP_LOG_LEVEL: %w", err)
			}
		}
		logging.Setup()
		return nil
	},
}

// Execute runs the cobra command tree against os.Args and exits with code 1
// if it returns an error. Cobra already prints its own usage/error messages,
// so Execute only needs to translate a failure into the process exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "output as JSON instead of a table")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "log level: debug, info, warn, error (overrides TKNCAP_LOG_LEVEL)")
}
