package cmd

import (
	"gitlab.sicepat.tech/pka/sds/cmd/bootstrap"
	"github.com/spf13/cobra"
)

// command for running http server
var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(_ *cobra.Command, _ []string) {

		var (
			dep        = bootstrap.NewDependency(cfgFile)
			cfg        = dep.GetConfig()
			logger     = dep.GetLogger()
			runnerHTTP = dep.GetHTTP()
			opts       = bootstrap.RunOptions{
				StartTimeout: cfg.Http.StartTimeout,
				StopTimeout:  cfg.Http.StopTimeout,
			}
		)

		err := bootstrap.New(dep).Run(runnerHTTP, opts)
		if err != nil {
			logger.Fatal(err)
		}

	},
}