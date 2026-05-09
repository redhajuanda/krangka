package upgrade

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const cliInstallTarget = "github.com/redhajuanda/krangka/cli/krangka@latest"
const cliModulePath = "github.com/redhajuanda/krangka/cli/krangka"

// Commands returns the upgrade command list.
func Commands() []*cobra.Command {
	return []*cobra.Command{newUpgradeCmd()}
}

// newUpgradeCmd updates the krangka CLI binary to the latest version.
func newUpgradeCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade krangka CLI to latest version",
		Long: `Upgrade the krangka CLI binary by reinstalling from the latest module version.

This command updates only the CLI executable. It does not change project files.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if checkOnly {
				latest, err := latestCLIVersion()
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stdout, "Latest available krangka CLI version: %s\n", latest)
				return nil
			}

			fmt.Fprintln(os.Stderr, "krangka: upgrading CLI to latest...")

			cmd := exec.Command("go", "install", cliInstallTarget)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to upgrade krangka CLI: %w", err)
			}

			fmt.Fprintln(os.Stderr, "krangka: CLI upgrade complete.")
			fmt.Fprintln(os.Stderr, "krangka: run `krangka --version` to verify the installed version.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "show latest available CLI version without installing")
	return cmd
}

// latestCLIVersion resolves the latest published version for krangka CLI module.
func latestCLIVersion() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", cliModulePath+"@latest")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to check latest krangka CLI version: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	version := strings.TrimSpace(stdout.String())
	if version == "" {
		return "", fmt.Errorf("failed to check latest krangka CLI version: empty response")
	}
	return version, nil
}
