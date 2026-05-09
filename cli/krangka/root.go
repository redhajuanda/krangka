package main

import (
	"log"
	"runtime/debug"

	"github.com/redhajuanda/krangka/cli/krangka/gonew"
	"github.com/redhajuanda/krangka/cli/krangka/skill"
	"github.com/redhajuanda/krangka/cli/krangka/upgrade"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "krangka",
	Version: cliVersion(),
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("use -h to show available commands")
	},
}

// cliVersion returns the current CLI version from build info.
func cliVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func main() {
	commands := []*cobra.Command{}
	commands = append(commands, gonew.Commands()...)
	commands = append(commands, skill.Commands()...)
	commands = append(commands, upgrade.Commands()...)
	rootCmd.AddCommand(commands...)
	rootCmd.Execute()
}
