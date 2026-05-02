/**
 * package cmd — root command
 *
 * <purpose-start>
 * Defines the root cobra.Command for tkncap and exposes the Execute function
 * called by main.go. The root command sets up global persistent flags that are
 * inherited by all subcommands:
 *   --json        emit JSON instead of a table
 *   --log-level   override TKNCAP_LOG_LEVEL (debug|info|warn|error)
 *
 * The logging package is initialised in PersistentPreRunE so the level is
 * applied before any subcommand body runs. The --log-level flag, if set,
 * overrides the TKNCAP_LOG_LEVEL env-var by writing to the environment before
 * logging.Setup is called.
 * <purpose-end>
 *
 * <inputs-start>
 * - os.Args (implicit, parsed by cobra).
 * - TKNCAP_LOG_LEVEL environment variable (read by logging.Setup).
 * <inputs-end>
 *
 * <outputs-start>
 * - None directly; subcommands write to os.Stdout.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Initialises the global slog logger via logging.Setup.
 * - Exits with code 1 on cobra Execute error.
 * <side-effects-end>
 */
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
	Short: "View current token quota for Claude Code, Gemini, and Antigravity accounts",
	Long: `tkncap reports the current token quota and usage for all configured
provider accounts. Accounts are configured via environment variables:

  TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>

Supported providers: claude, gemini, antigravity

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

/**
 * Execute
 *
 * <purpose-start>
 * Entry point called by main.go. Runs the cobra command tree against os.Args
 * and exits with code 1 if the command returns an error. Cobra prints its own
 * usage/error messages, so Execute only needs to handle the exit code.
 * <purpose-end>
 *
 * <inputs-start>
 * - None (reads os.Args implicitly via cobra).
 * <inputs-end>
 *
 * <outputs-start>
 * - None.
 * <outputs-end>
 *
 * <side-effects-start>
 * - May call os.Exit(1) on error.
 * <side-effects-end>
 */
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "output as JSON instead of a table")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "log level: debug, info, warn, error (overrides TKNCAP_LOG_LEVEL)")
}
