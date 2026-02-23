package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use: "application",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("use -h to show available commands")
	},
}

func Run() {

	// flags for config file
	var (
		flagName  = "config"
		flagValue = "configs/files/default.yaml"
		flagUsage = fmt.Sprintf("config file (default is %s)", flagValue)
	)

	// add persistent flag to root command
	rootCmd.PersistentFlags().StringVar(&cfgFile, flagName, flagValue, flagUsage)

	// http command
	rootCmd.AddCommand(httpCmd)

	// migrate command
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateNewCmd)
	rootCmd.AddCommand(migrateCmd)

	// subscriber command
	rootCmd.AddCommand(subscriberCmd)

	// worker command
	workerCmd.AddCommand(workerGenerateIDCmd)
	workerCmd.AddCommand(workerOutboxRelayCmd)
	rootCmd.AddCommand(workerCmd)

	// execute root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute root command: %v", err)
	}

}
