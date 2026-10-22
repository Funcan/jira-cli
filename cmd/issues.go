package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"

	jira "jira-cli/jira-lib"
)

// issueListFields are the fields printIssuesByStatus needs from a search.
var issueListFields = []string{"summary", "status", "issuetype", "priority"}

// printIssuesByStatus prints issues as one table per status, with statuses in
// logical order. Statuses with no issues are omitted. Styling and links are
// stripped when w isn't a terminal.
func printIssuesByStatus(w io.Writer, c *jira.Client, issues []jira.Issue) {
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

	out := colorprofile.NewWriter(w, os.Environ())
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
}

// jqlString quotes s as a JQL string literal.
func jqlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
