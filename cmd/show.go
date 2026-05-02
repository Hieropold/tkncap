/**
 * package cmd — show command
 *
 * <purpose-start>
 * Implements the `tkncap show` subcommand, which is also the default action
 * when tkncap is invoked without any subcommand. The command:
 *   1. Discovers all configured accounts from os.Environ().
 *   2. If no accounts are found, prints a usage hint and exits 0.
 *   3. For each account, retrieves the registered Provider and calls Fetch.
 *   4. Selects the output renderer based on the --json flag.
 *   5. Renders all Quota records to os.Stdout.
 *
 * Provider implementations are registered via init() in their respective
 * files (claude.go, gemini.go, antigravity.go). This file imports the
 * provider package with a blank import to trigger those init() calls.
 * <purpose-end>
 *
 * <inputs-start>
 * - os.Environ() for account discovery.
 * - JSONOutput global flag (from root.go) to select renderer.
 * <inputs-end>
 *
 * <outputs-start>
 * - Quota table or JSON array written to os.Stdout.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Reads environment variables.
 * - Writes to os.Stdout.
 * - Logs progress at info/debug level.
 * - Exits with code 1 if rendering fails.
 * <side-effects-end>
 */
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/hieropold/tkncap/internal/account"
	"github.com/hieropold/tkncap/internal/output"
	"github.com/hieropold/tkncap/internal/provider"

	// Blank imports trigger init() registration of each provider stub.
	_ "github.com/hieropold/tkncap/internal/provider"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show token quota for all configured accounts",
	Long: `Reads TKNCAP_* environment variables to discover configured accounts,
fetches the current token quota for each, and renders the results.`,
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	// Wire show as the default action when no subcommand is given.
	rootCmd.RunE = runShow
}

/**
 * runShow
 *
 * <purpose-start>
 * Command body for `tkncap show`. Orchestrates account discovery, provider
 * dispatch, and output rendering. Returns an error only for rendering failures;
 * per-account errors are encoded in the Quota.Status field and rendered inline
 * so the user sees all results even when some accounts fail.
 * <purpose-end>
 *
 * <inputs-start>
 * - cmd *cobra.Command: the executing command (used for context).
 * - args []string: positional arguments (unused).
 * <inputs-end>
 *
 * <outputs-start>
 * - error: non-nil only if rendering fails.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Reads os.Environ() for account discovery.
 * - Calls provider.Fetch for each account (network I/O in real implementations).
 * - Writes to os.Stdout.
 * - Logs progress at info and debug levels.
 * <side-effects-end>
 */
func runShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	slog.Info("show: discovering accounts from environment")
	accounts, err := account.Discover(os.Environ())
	if err != nil {
		return fmt.Errorf("show: account discovery: %w", err)
	}
	slog.Info("show: account discovery complete", "count", len(accounts))

	if len(accounts) == 0 {
		fmt.Fprintln(os.Stderr, "No accounts configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Set environment variables in the form:")
		fmt.Fprintln(os.Stderr, "  TKNCAP_<PROVIDER>_<ACCOUNT>_<FIELD>=<value>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  TKNCAP_CLAUDE_WORK_CREDENTIALS_PATH=~/.claude/.credentials.json")
		fmt.Fprintln(os.Stderr, "  TKNCAP_GEMINI_MAIN_API_KEY=AIza...")
		fmt.Fprintln(os.Stderr, "  TKNCAP_ANTIGRAVITY_DEFAULT_TOKEN=tok...")
		return nil
	}

	quotas := make([]provider.Quota, 0, len(accounts))
	for _, acc := range accounts {
		slog.Debug("show: fetching quota", "provider", acc.Provider, "account", acc.Name)

		p := provider.For(acc.Provider)
		if p == nil {
			slog.Info("show: no provider registered for kind — marking as error",
				"provider", acc.Provider, "account", acc.Name)
			quotas = append(quotas, provider.Quota{
				Account: acc,
				Status:  provider.StatusError,
				Message: fmt.Sprintf("no provider registered for %q", acc.Provider),
			})
			continue
		}

		q := p.Fetch(ctx, acc)
		slog.Debug("show: quota fetched",
			"provider", acc.Provider, "account", acc.Name, "status", q.Status)
		quotas = append(quotas, q)
	}

	var renderer output.Renderer
	if JSONOutput {
		slog.Debug("show: using JSON renderer")
		renderer = &output.JSONRenderer{}
	} else {
		slog.Debug("show: using table renderer")
		renderer = &output.TableRenderer{}
	}

	if err := renderer.Render(os.Stdout, quotas); err != nil {
		return fmt.Errorf("show: render: %w", err)
	}

	slog.Info("show: complete")
	return nil
}
