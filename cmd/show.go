// Package cmd (show.go) implements `tkncap show`, also the default action
// when tkncap is invoked without a subcommand. Provider implementations
// register themselves via init() in their own files (claude.go, gemini.go);
// this file blank-imports the provider package to trigger those
// registrations.
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

// runShow is the command body for `tkncap show`, orchestrating account
// discovery, provider dispatch, and output rendering. It returns an error
// only for rendering failures; per-account fetch errors are instead encoded
// in Quota.Status and rendered inline so the user sees all results even when
// some accounts fail.
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

		qs := p.Fetch(ctx, acc)
		for _, q := range qs {
			slog.Debug("show: quota fetched",
				"provider", acc.Provider, "account", acc.Name, "status", q.Status, "name", q.Name)
		}
		quotas = append(quotas, qs...)
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
