package cmd

import (
	"fmt"
	"text/tabwriter"

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

		jql := fmt.Sprintf(`project = "%s" AND assignee = currentUser() AND sprint in openSprints() ORDER BY priority DESC, key ASC`, p.Project)
		issues, err := c.Search(cmd.Context(), jql, []string{"summary", "status", "issuetype", "priority"})
		if err != nil {
			return err
		}

		var statuses []jira.Status
		byStatus := map[string][]jira.Issue{}
		for _, issue := range issues {
			name := issue.Fields.Status.Name
			if _, ok := byStatus[name]; !ok {
				statuses = append(statuses, issue.Fields.Status)
			}
			byStatus[name] = append(byStatus[name], issue)
		}
		jira.SortStatuses(statuses)

		out := cmd.OutOrStdout()
		for i, status := range statuses {
			if i > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprintln(out, status.Name)
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "KEY\tTYPE\tPRIORITY\tSUMMARY")
			for _, issue := range byStatus[status.Name] {
				priority := ""
				if issue.Fields.Priority != nil {
					priority = issue.Fields.Priority.Name
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", issue.Key, issue.Fields.IssueType.Name, priority, issue.Fields.Summary)
			}
			if err := tw.Flush(); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	sprintCmd.Flags().StringVar(&sprintProject, "project", "", "project to list (default from ~/.jira.yaml)")
	rootCmd.AddCommand(sprintCmd)
}
