// Package cmdutil holds cobra helpers shared by the command packages.
package cmdutil

import "github.com/spf13/cobra"

// Quiet stops cobra from printing a runtime error and the usage text once the
// arguments parsed. The root command then prints the error alone, or nothing
// for an error that reports itself as silent. Call it first in RunE.
func Quiet(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
}
