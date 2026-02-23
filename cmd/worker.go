package cmd

import (
	"context"
	"log"
	"time"

	"github.com/redhajuanda/krangka/cmd/bootstrap"
	"github.com/spf13/cobra"
)

// command for running worker
var workerCmd = &cobra.Command{
	Use:   "worker [worker_name]",
	Short: "Start a worker process",
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.MinimumNArgs(1)),
	Run: func(c *cobra.Command, args []string) {
		log.Println("use -h to show available commands")
	},
}

// command for running worker generate id
var workerGenerateIDCmd = &cobra.Command{
	Use:   "generate-id",
	Short: "Start the generate id worker",
	Run: func(c *cobra.Command, args []string) {

		var (
			ctx    = context.Background()
			dep    = bootstrap.NewDependency(cfgFile)
			logger = dep.GetLogger()
			runner = dep.GetWorkerGenerateID()
			boot   = bootstrap.New(dep)
		)

		err := boot.Execute(ctx, runner)
		if err != nil {
			logger.Fatalf("failed to execute worker generate id: %v", err)
		}

	},
}

// command for running worker relay outbox
var workerOutboxRelayCmd = &cobra.Command{
	Use:   "outbox-relay",
	Short: "Start the outbox relay worker",
	Run: func(c *cobra.Command, args []string) {

		var (
			dep    = bootstrap.NewDependency(cfgFile)
			cfg    = dep.GetConfig()
			logger = dep.GetLogger()
			runner = dep.GetWorkerOutboxRelay()
			boot   = bootstrap.New(dep)
			opts   = bootstrap.ScheduleOptions{
				ShutdownTimeout: 30 * time.Second,
				SingletonMode:   true, // Prevent overlapping executions
			}
		)

		err := boot.Schedule(cfg.Event.Outbox.RelayPattern, runner, opts)
		if err != nil {
			logger.Fatalf("failed to schedule worker outbox relay: %v", err)
		}

	},
}
