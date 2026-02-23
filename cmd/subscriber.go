package cmd

import (
	"time"

	"github.com/redhajuanda/krangka/cmd/bootstrap"
	"github.com/spf13/cobra"
)

var subscriberCmd = &cobra.Command{
	Use:   "subscriber",
	Short: "Start the subscriber",
	Run: func(c *cobra.Command, args []string) {
		var (
			opts = bootstrap.RunOptions{
				StartTimeout: 30 * time.Second,
				StopTimeout:  30 * time.Second,
			}
			dep    = bootstrap.NewDependency(cfgFile)
			logger = dep.GetLogger()
			runner = dep.GetSubscriber(opts.StopTimeout)
			boot   = bootstrap.New(dep)
		)

		if err := boot.Run(runner, opts); err != nil {
			logger.Fatalf("failed to execute subscriber: %v", err)
		}
	},
}
