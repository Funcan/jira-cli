package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
)

var watchCmd = &cobra.Command{
	Use:   "watch <key|url>",
	Short: "Watch an issue",
	Long:  `Add me as a watcher of an issue. Already watching it counts as success.`,
	Args:  cobra.ExactArgs(1),
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
		me, err := c.Myself(ctx)
		if err != nil {
			return err
		}
		if err := c.AddWatcher(ctx, key, me); err != nil {
			return err
		}
		out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
		theme := newTheme(out.Profile)
		fmt.Fprintln(out, "Watching", theme.IssueKey.Hyperlink(c.BrowseURL(key)).Render(key), c.BrowseURL(key))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
