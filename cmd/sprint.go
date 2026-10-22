package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
)

var sprintProject string

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "List my tickets in the current sprint, grouped by status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, p, err := jira.NewClientForProject(sprintProject)
		if err != nil {
			return err
		}

		jql := fmt.Sprintf(`project = %s AND assignee = currentUser() AND sprint in openSprints() ORDER BY priority DESC, key ASC`, jqlString(p.Project))
		issues, err := c.Search(cmd.Context(), jql, issueListFields)
		if err != nil {
			return err
		}
		printIssuesByStatus(cmd.OutOrStdout(), c, issues)
		return nil
	},
}

func init() {
	sprintCmd.Flags().StringVar(&sprintProject, "project", "", "project to list (default from ~/.jira.yaml)")
	rootCmd.AddCommand(sprintCmd)
}
