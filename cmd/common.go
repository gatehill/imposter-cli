package cmd

import (
	"gatehill.io/imposter/engine"
	"github.com/spf13/cobra"
)

func registerEngineTypeCompletions(cmd *cobra.Command) {
	cmd.RegisterFlagCompletionFunc("engine-type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(engine.EngineTypeDockerCore),
			string(engine.EngineTypeDockerAll),
			string(engine.EngineTypeJvmSingleJar),
		}, cobra.ShellCompDirectiveNoFileComp
	})
}
