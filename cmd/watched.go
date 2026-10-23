package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
)

var (
	watchedProject  string
	watchedOpen     bool
	watchedClosed   bool
	watchedAll      bool
	watchedStatuses []string
	watchedSort     string
)

var watchedCmd = &cobra.Command{
	Use:   "watched",
	Short: "List tickets I'm watching, grouped by status",
	Long: `List tickets I'm watching, grouped by status. By default only tickets
that aren't closed (i.e. not in the Done status category) are listed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, p, err := jira.NewClientForProject(watchedProject)
		if err != nil {
			return err
		}

		clauses := []string{"project = " + jqlString(p.Project), "watcher = currentUser()"}
		switch {
		case len(watchedStatuses) > 0:
			var quoted []string
			for _, s := range watchedStatuses {
				quoted = append(quoted, jqlString(s))
			}
			clauses = append(clauses, fmt.Sprintf("status in (%s)", strings.Join(quoted, ", ")))
		case watchedClosed:
			clauses = append(clauses, "statusCategory = Done")
		case watchedAll:
		default: // --open
			clauses = append(clauses, "statusCategory != Done")
		}
		return listIssues(cmd, c, strings.Join(clauses, " AND "), watchedSort)
	},
}

func init() {
	watchedCmd.Flags().StringVar(&watchedProject, "project", "", "project to list (default from ~/.jira.yaml)")
	watchedCmd.Flags().BoolVar(&watchedOpen, "open", false, "list only open tickets (the default)")
	watchedCmd.Flags().BoolVar(&watchedClosed, "closed", false, "list only closed tickets")
	watchedCmd.Flags().StringArrayVar(&watchedStatuses, "status", nil, "list only tickets in this status (repeatable; matches any)")
	watchedCmd.Flags().BoolVar(&watchedAll, "all", false, "list tickets in any status")
	watchedCmd.Flags().StringVar(&watchedSort, "sort", defaultSort, sortFlagUsage)
	watchedCmd.MarkFlagsMutuallyExclusive("open", "closed", "all", "status")
	rootCmd.AddCommand(watchedCmd)
}
