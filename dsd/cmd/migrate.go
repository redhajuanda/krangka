package cmd

import (
	"context"
	"log"
	"strconv"

	"gitlab.sicepat.tech/pka/sds/cmd/bootstrap"
	"github.com/spf13/cobra"
)

// command for database migration
var migrateCmd = &cobra.Command{
	Use: "migrate",
	Run: func(c *cobra.Command, _ []string) {
		log.Println("use -h to show available commands")
	},
}

// command for database migration up
var migrateUpCmd = &cobra.Command{
	Use:   "up [repository] [max]",
	Short: "Run migrations up for the specified repository",
	Args:  cobra.MatchAll(cobra.MaximumNArgs(2), cobra.MinimumNArgs(1)),
	Run: func(c *cobra.Command, args []string) {
		runMigrate("up", args)
	},
}

// command for database migration down
var migrateDownCmd = &cobra.Command{
	Use:   "down [repository] [max]",
	Short: "Run migrations down for the specified repository",
	Args:  cobra.MatchAll(cobra.MaximumNArgs(2), cobra.MinimumNArgs(1)),
	Run: func(c *cobra.Command, args []string) {
		runMigrate("down", args)
	},
}

// command for database migration new
var migrateNewCmd = &cobra.Command{
	Use:   "new [repository] [migration_name]",
	Short: "Create a new migration file",
	Long:  "Create a new migration file with the specified repository and migration name.",
	Args:  cobra.MatchAll(cobra.MaximumNArgs(2), cobra.MinimumNArgs(2)),
	Run: func(c *cobra.Command, args []string) {
		runMigrateGenerate(cfgFile, args[0], args[1])
	},
}

// runMigrate runs the migrate up / down command
func runMigrate(migrateType string, args []string) {

	max := "0"
	if len(args) > 1 {
		max = args[1]
	}
	maxInt, err := strconv.Atoi(max)
	if err != nil {
		log.Fatalf("failed to convert max to int: %v", err)
	}

	repository := args[0]

	var (
		ctx    = context.Background()
		dep    = bootstrap.NewDependency(cfgFile)
		logger = dep.GetLogger()
		runner = dep.GetMigrate(migrateType, maxInt, repository)
		boot   = bootstrap.New(dep)
	)

	err = boot.Execute(ctx, runner)
	if err != nil {
		logger.Fatalf("failed to execute migrate: %v", err)
	}

}

// runMigrateGenerate runs the migrate generate command
func runMigrateGenerate(cfgFile string, repository string, fileName string) {

	var (
		ctx    = context.Background()
		dep    = bootstrap.NewDependency(cfgFile)
		logger = dep.GetLogger()
		runner = dep.GetMigrateGenerate(repository, fileName)
		boot   = bootstrap.New(dep)
	)

	err := boot.Execute(ctx, runner)
	if err != nil {
		logger.Fatalf("failed to execute migrate generate: %v", err)
	}

}