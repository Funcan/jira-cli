package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
)

var (
	whoamiInstance string
	whoamiProject  string
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the authenticated user, to validate config",
	Long: `Show the user each JIRA instance authenticates you as, to validate the
credentials in ~/.jira.yaml. Checks every configured instance unless
--instance or --project is given.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := jira.LoadConfig("")
		if err != nil {
			return err
		}

		var instances []jira.InstanceConfig
		switch {
		case whoamiInstance != "":
			inst, err := cfg.Instance(whoamiInstance)
			if err != nil {
				return err
			}
			instances = append(instances, *inst)
		case whoamiProject != "":
			p, err := cfg.Project(whoamiProject)
			if err != nil {
				return err
			}
			inst, err := cfg.Instance(p.Instance)
			if err != nil {
				return fmt.Errorf("project %s: %w", p.Project, err)
			}
			instances = append(instances, *inst)
		default:
			instances = cfg.Instances
		}

		failed := 0
		out := cmd.OutOrStdout()
		for _, inst := range instances {
			c, err := jira.NewClient(inst)
			if err != nil {
				fmt.Fprintf(out, "%s: ERROR: %v\n", inst.Instance, err)
				failed++
				continue
			}
			u, err := c.Myself(cmd.Context())
			if err != nil {
				fmt.Fprintf(out, "%s: ERROR: %v\n", inst.Instance, err)
				failed++
				continue
			}
			if u.EmailAddress != "" {
				fmt.Fprintf(out, "%s: %s <%s>\n", inst.Instance, u.DisplayName, u.EmailAddress)
			} else {
				fmt.Fprintf(out, "%s: %s\n", inst.Instance, u.DisplayName)
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d of %d instances failed", failed, len(instances))
		}
		return nil
	},
}

func init() {
	whoamiCmd.Flags().StringVar(&whoamiInstance, "instance", "", "instance to check")
	whoamiCmd.Flags().StringVar(&whoamiProject, "project", "", "check the instance hosting this project")
	whoamiCmd.MarkFlagsMutuallyExclusive("instance", "project")
	rootCmd.AddCommand(whoamiCmd)
}
