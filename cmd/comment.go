package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
	"github.com/Funcan/jira-cli/wiki2markdown2wiki"
)

var commentCmd = &cobra.Command{
	Use:   "comment <key|url>",
	Short: "Comment on an issue, written in your editor",
	Long: `Open an empty Markdown file in $VISUAL or $EDITOR (default vi) and post
what you write as a comment on the issue. Nothing is posted if the editor
exits with an error or the file is left empty.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
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
		// Check the issue exists before the user writes anything.
		issue, err := c.GetIssue(ctx, key, []string{"summary"})
		if err != nil {
			return err
		}
		key = issue.Key

		return editLoop(cmd, "", func(text string) error {
			text = strings.TrimSpace(text)
			if text == "" {
				return errNoChanges
			}
			if err := c.AddComment(ctx, key, wiki2markdown2wiki.MarkdownToWiki(text)); err != nil {
				return err
			}
			out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			theme := newTheme(out.Profile)
			fmt.Fprintln(out, "Commented on", theme.IssueKey.Hyperlink(c.BrowseURL(key)).Render(key), c.BrowseURL(key))
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(commentCmd)
}
