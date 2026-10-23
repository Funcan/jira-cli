package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
)

var (
	mineProject  string
	mineOpen     bool
	mineClosed   bool
	mineAll      bool
	mineStatuses []string
	mineSort     string
)

var mineCmd = &cobra.Command{
	Use:   "mine",
	Short: "List my tickets, grouped by status",
	Long: `List tickets assigned to me, grouped by status. By default only tickets
that aren't closed (i.e. not in the Done status category) are listed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, p, err := jira.NewClientForProject(mineProject)
		if err != nil {
			return err
		}

		clauses := []string{"project = " + jqlString(p.Project), "assignee = currentUser()"}
		switch {
		case len(mineStatuses) > 0:
			var quoted []string
			for _, s := range mineStatuses {
				quoted = append(quoted, jqlString(s))
			}
			clauses = append(clauses, fmt.Sprintf("status in (%s)", strings.Join(quoted, ", ")))
		case mineClosed:
			clauses = append(clauses, "statusCategory = Done")
		case mineAll:
		default: // --open
			clauses = append(clauses, "statusCategory != Done")
		}
		return listIssues(cmd, c, strings.Join(clauses, " AND "), mineSort)
	},
}

func init() {
	mineCmd.Flags().StringVar(&mineProject, "project", "", "project to list (default from ~/.jira.yaml)")
	mineCmd.Flags().BoolVar(&mineOpen, "open", false, "list only open tickets (the default)")
	mineCmd.Flags().BoolVar(&mineClosed, "closed", false, "list only closed tickets")
	mineCmd.Flags().StringArrayVar(&mineStatuses, "status", nil, "list only tickets in this status (repeatable; matches any)")
	mineCmd.Flags().BoolVar(&mineAll, "all", false, "list tickets in any status")
	mineCmd.Flags().StringVar(&mineSort, "sort", defaultSort, sortFlagUsage)
	mineCmd.MarkFlagsMutuallyExclusive("open", "closed", "all", "status")
	rootCmd.AddCommand(mineCmd)
}
