package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/funcan/showmd"
	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
	"github.com/Funcan/jira-cli/wiki2markdown2wiki"
)

// showFields are the fields shown, in order, between the summary and the
// description, for projects without show_fields in ~/.jira.yaml. Empty fields
// are omitted.
var showFields = []string{
	"issuetype", "status", "priority", "assignee", "reporter",
	"created", "updated", "labels", "components", "fixVersions", "parent",
}

var showCmd = &cobra.Command{
	Use:   "show <key|url>",
	Short: "Show a single issue",
	Long: `Show a single issue, given its key (e.g. ABC-123) or its URL. Keys for
projects not in ~/.jira.yaml are looked up on the default project's instance.
The fields shown are the project's show_fields in ~/.jira.yaml (field IDs or
display names), or a built-in list.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := jira.LoadConfig("")
		if err != nil {
			return err
		}
		inst, key, err := cfg.ResolveIssue(args[0])
		if err != nil {
			return err
		}
		c, err := jira.NewClient(*inst)
		if err != nil {
			return err
		}
		names := issueProject(cfg, key).ShowFields
		if len(names) == 0 {
			names = showFields
		}
		// Configured fields may be display names, which can only be mapped
		// to IDs once we have the issue's field names.
		issue, err := c.GetIssue(cmd.Context(), key, []string{"*all"})
		if err != nil {
			return err
		}
		comments, err := c.Comments(cmd.Context(), issue.Key)
		if err != nil {
			return err
		}

		out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
		theme := newTheme(out.Profile)
		md := markdownRenderer(cmd.OutOrStdout(), out.Profile)

		key = theme.IssueKey.Hyperlink(c.BrowseURL(issue.Key)).Render(issue.Key)
		fmt.Fprintln(out, key+"  "+theme.Summary.Render(fieldText(issue.Fields["summary"])))
		fmt.Fprintln(out)

		t := table.New().
			BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
			BorderHeader(false).BorderColumn(false).
			StyleFunc(func(row, col int) lipgloss.Style {
				if col == 0 {
					return theme.FieldName.PaddingRight(2)
				}
				return lipgloss.NewStyle()
			})
		for _, f := range showFieldIDs(issue, names) {
			value := fieldText(issue.Fields[f])
			if value == "" {
				continue
			}
			switch f {
			case "issuetype":
				value = theme.IssueType(value)
			case "priority":
				value = theme.Priority(value)
			}
			name := issue.Names[f]
			if name == "" {
				name = f
			}
			t.Row(name, value)
		}
		// The table pads every cell to its column width, including the last,
		// which leaves trailing whitespace on each line.
		for _, line := range strings.Split(t.Render(), "\n") {
			fmt.Fprintln(out, strings.TrimRight(line, " "))
		}

		if desc := strings.TrimSpace(wikiToMarkdown(fieldText(issue.Fields["description"]))); desc != "" {
			fmt.Fprintln(out)
			fmt.Fprintln(out, theme.Heading.Render(issue.Names["description"]))
			fmt.Fprintln(out, md(desc))
		}

		if len(comments) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, theme.Heading.Render("Comments"))
			for i, comment := range comments {
				if i > 0 {
					fmt.Fprintln(out)
				}
				fmt.Fprintln(out, theme.CommentAuthor.Render(comment.Author.DisplayName)+"  "+
					theme.CommentDate.Render(comment.Created.Local().Format("2006-01-02 15:04")))
				fmt.Fprintln(out, md(strings.TrimSpace(wikiToMarkdown(comment.Body))))
			}
		}
		return nil
	},
}

// showFieldIDs maps field names (IDs or display names, case-insensitively)
// to the IDs of the issue's fields, skipping names that don't match.
func showFieldIDs(issue *jira.RawIssue, names []string) []string {
	var ids []string
	for _, name := range names {
		var match string
		for id, n := range issue.Names {
			if strings.EqualFold(id, name) {
				match = id
				break
			}
			if strings.EqualFold(n, name) && (match == "" || id < match) {
				match = id
			}
		}
		if match != "" {
			ids = append(ids, match)
		}
	}
	return ids
}

// wikiToMarkdown converts JIRA wiki markup to Markdown. Data Center text
// often has Windows line endings.
func wikiToMarkdown(wiki string) string {
	return wiki2markdown2wiki.WikiToMarkdown(strings.ReplaceAll(wiki, "\r\n", "\n"))
}

// markdownRenderer returns a function that renders Markdown for w. It wraps
// at the terminal's width, or not at all if w isn't a terminal.
func markdownRenderer(w io.Writer, profile colorprofile.Profile) func(string) string {
	t := showmd.Terminal{Color: profile, Hyperlinks: true}
	wrap := false
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(f.Fd()); err == nil && width > 0 {
			t.Width, wrap = width, true
		}
	}
	return func(markdown string) string {
		return strings.TrimRight(showmd.Render(markdown, t, showmd.Options{Wrap: &wrap}), "\n")
	}
}

// fieldText renders a raw field value as text. It handles the shapes JIRA
// uses for standard and most custom fields: strings, timestamps, numbers,
// objects such as users, statuses and versions (shown by display name, name
// or value), linked issues (key and summary), and arrays of any of these.
func fieldText(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return valueText(v)
}

func valueText(v any) string {
	switch v := v.(type) {
	case string:
		if t, err := time.Parse(jira.TimeFormat, v); err == nil {
			return t.Local().Format("2006-01-02 15:04")
		}
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []any:
		var parts []string
		for _, item := range v {
			if s := valueText(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		for _, k := range []string{"displayName", "name", "value"} {
			if s, ok := v[k].(string); ok && s != "" {
				return s
			}
		}
		if key, ok := v["key"].(string); ok {
			if fields, ok := v["fields"].(map[string]any); ok {
				if summary, ok := fields["summary"].(string); ok {
					return key + " " + summary
				}
			}
			return key
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(showCmd)
}
