package main

import (
	"log"

	"github.com/redhajuanda/krangka/cli/krangka/gonew"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "krangka",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("use -h to show available commands")
	},
}

func main() {
	commands := []*cobra.Command{}
	commands = append(commands, gonew.Commands()...)
	rootCmd.AddCommand(commands...)
	rootCmd.Execute()
}

// go install github.com/redhajuanda/krangka/cli@latest
