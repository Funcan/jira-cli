package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
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

		// Strips all styling and links when not writing to a terminal.
		out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
		theme := newTheme(out.Profile)
		for i, status := range statuses {
			if i > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprintln(out, theme.Heading.Render(status.Name))
			t := table.New().
				BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
				BorderHeader(false).BorderColumn(false).
				Headers("KEY", "TYPE", "PRIORITY", "SUMMARY").
				StyleFunc(func(row, col int) lipgloss.Style {
					s := lipgloss.NewStyle()
					if row == table.HeaderRow {
						s = theme.TableHeader
					}
					if col < 3 {
						s = s.PaddingRight(2)
					}
					return s
				})
			for _, issue := range byStatus[status.Name] {
				priority := ""
				if issue.Fields.Priority != nil {
					priority = issue.Fields.Priority.Name
				}
				key := theme.IssueKey.Hyperlink(c.BrowseURL(issue.Key)).Render(issue.Key)
				t.Row(key, theme.IssueType(issue.Fields.IssueType.Name), theme.Priority(priority), issue.Fields.Summary)
			}
			// The table pads every cell to its column width, including the
			// last, which leaves trailing whitespace on each line.
			for _, line := range strings.Split(t.Render(), "\n") {
				fmt.Fprintln(out, strings.TrimRight(line, " "))
			}
		}
		return nil
	},
}

func init() {
	sprintCmd.Flags().StringVar(&sprintProject, "project", "", "project to list (default from ~/.jira.yaml)")
	rootCmd.AddCommand(sprintCmd)
}
