package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
)

var (
	sprintProject string
	sprintSort    string
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "List my tickets in the current sprint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, p, err := jira.NewClientForProject(sprintProject)
		if err != nil {
			return err
		}

		jql := fmt.Sprintf(`project = %s AND assignee = currentUser() AND sprint in openSprints()`, jqlString(p.Project))
		return listIssues(cmd, c, jql, sprintSort)
	},
}

func init() {
	sprintCmd.Flags().StringVar(&sprintProject, "project", "", "project to list (default from ~/.jira.yaml)")
	sprintCmd.Flags().StringVar(&sprintSort, "sort", defaultSort, sortFlagUsage)
	rootCmd.AddCommand(sprintCmd)
}
