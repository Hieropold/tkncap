/**
 * package main
 *
 * <purpose-start>
 * Entrypoint for tkncap. Delegates immediately to cmd.Execute() so that all
 * command wiring, flag parsing, and business logic live in the cmd package.
 * Keeping main.go thin ensures the CLI is testable and the cobra root command
 * can be reused in integration tests without spawning a subprocess.
 * <purpose-end>
 *
 * <inputs-start>
 * - os.Args (implicit): command-line arguments parsed by cobra.
 * <inputs-end>
 *
 * <outputs-start>
 * - Exits with code 0 on success, non-zero on error.
 * <outputs-end>
 *
 * <side-effects-start>
 * - Initialises the global slog logger via cmd.Execute → logging.Setup.
 * - Writes output to os.Stdout / os.Stderr.
 * <side-effects-end>
 */
package main

import "github.com/hieropold/tkncap/cmd"

func main() {
	cmd.Execute()
}
