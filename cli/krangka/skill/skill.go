package skill

import "github.com/spf13/cobra"

func Commands() []*cobra.Command {
	skill := &cobra.Command{
		Use:   "skill",
		Short: "Manage krangka agent skills, rules, and configs",
	}
	skill.AddCommand(newUpgradeCmd())
	return []*cobra.Command{skill}
}
