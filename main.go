// Package main is kept thin, delegating immediately to cmd.Execute so the
// cobra root command stays testable and reusable in integration tests
// without spawning a subprocess.
package main

import "github.com/hieropold/tkncap/cmd"

func main() {
	cmd.Execute()
}
