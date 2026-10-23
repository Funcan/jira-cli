package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
)

// defaultSort is the issue list order when none is given.
const defaultSort = "status,priority"

// sortFlagUsage is the help text for --sort flags.
const sortFlagUsage = `comma separated sort fields: opened, updated, status, priority, type,
project. Prefix with "+" for ascending or "-" for descending; otherwise
dates and priority sort descending, the rest ascending. Grouped by status
if status is first`

// listIssues searches for issues matching jql, sorts them by sortSpec (see
// jira.ParseSort) and prints them. If the first sort field is status, issues
// are printed as one table per status; otherwise as a single table with a
// status column.
func listIssues(cmd *cobra.Command, c *jira.Client, jql, sortSpec string) error {
	keys, err := jira.ParseSort(sortSpec)
	if err != nil {
		return err
	}
	issues, err := c.Search(cmd.Context(), jql, []string{"summary", "status", "issuetype", "priority", "project", "created", "updated"})
	if err != nil {
		return err
	}
	var priorities []jira.Priority
	if slices.ContainsFunc(keys, func(k jira.SortKey) bool { return k.Field == jira.SortPriority }) {
		if priorities, err = c.Priorities(cmd.Context()); err != nil {
			return err
		}
	}
	jira.SortIssues(issues, keys, priorities)

	out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
	theme := newTheme(out.Profile)
	if len(keys) == 0 || keys[0].Field != jira.SortStatus {
		printIssueTable(out, theme, c, issues, true)
		return nil
	}
	// Sorting by status first leaves each status's issues contiguous.
	for start := 0; start < len(issues); {
		end := start + 1
		for end < len(issues) && issues[end].Fields.Status.Name == issues[start].Fields.Status.Name {
			end++
		}
		if start > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, theme.Heading.Render(issues[start].Fields.Status.Name))
		printIssueTable(out, theme, c, issues[start:end], false)
		start = end
	}
	return nil
}

// printIssueTable prints issues as a table, optionally with a status column.
func printIssueTable(w io.Writer, theme Theme, c *jira.Client, issues []jira.Issue, withStatus bool) {
	headers := []string{"KEY", "TYPE", "PRIORITY", "SUMMARY"}
	if withStatus {
		headers = []string{"KEY", "STATUS", "TYPE", "PRIORITY", "SUMMARY"}
	}
	t := table.New().
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle()
			if row == table.HeaderRow {
				s = theme.TableHeader
			}
			if col < len(headers)-1 {
				s = s.PaddingRight(2)
			}
			return s
		})
	for _, issue := range issues {
		priority := ""
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}
		row := []string{
			theme.IssueKey.Hyperlink(c.BrowseURL(issue.Key)).Render(issue.Key),
			theme.IssueType(issue.Fields.IssueType.Name),
			theme.Priority(priority),
			issue.Fields.Summary,
		}
		if withStatus {
			row = slices.Insert(row, 1, issue.Fields.Status.Name)
		}
		t.Row(row...)
	}
	// The table pads every cell to its column width, including the last,
	// which leaves trailing whitespace on each line.
	for _, line := range strings.Split(t.Render(), "\n") {
		fmt.Fprintln(w, strings.TrimRight(line, " "))
	}
}

// jqlString quotes s as a JQL string literal.
func jqlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
