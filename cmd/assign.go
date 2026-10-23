package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
)

var (
	assignUser   string
	assignTeam   string
	assignDryRun bool
)

var assignCmd = &cobra.Command{
	Use:   "assign <key|url>",
	Short: "Assign an issue to you, or to another user or team",
	Long: `Assign an issue. By default it's assigned to you: the fields in the
project's "me" section of ~/.jira.yaml are set, or if there isn't one, the
assignee is set to the current user.

--user sets just the assignee and --team sets just the Team field; give both
to set both. Users are matched by email or username, teams by name.`,
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

		values := map[string]any{}
		if assignUser != "" {
			values["assignee"] = assignUser
		}
		if assignTeam != "" {
			values["team"] = assignTeam
		}
		if len(values) == 0 {
			values = issueProject(cfg, key).Me
		}
		if len(values) == 0 {
			u, err := c.Myself(ctx)
			if err != nil {
				return err
			}
			me := u.EmailAddress
			if me == "" {
				me = u.Name
			}
			values = map[string]any{"assignee": me}
		}

		meta, err := c.EditMeta(ctx, key)
		if err != nil {
			return err
		}
		update := map[string]any{}
		for name, v := range values {
			f, err := jira.FindField(meta, name)
			if err != nil {
				return err
			}
			if update[f.ID], err = c.FieldPayload(ctx, f, v); err != nil {
				return err
			}
		}

		out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
		if assignDryRun {
			return printDryRun(out, "PUT "+key, map[string]any{"fields": update}, nil)
		}
		if err := c.UpdateIssue(ctx, key, update); err != nil {
			return err
		}
		theme := newTheme(out.Profile)
		fmt.Fprintln(out, "Assigned", theme.IssueKey.Hyperlink(c.BrowseURL(key)).Render(key), c.BrowseURL(key))
		return nil
	},
}

func init() {
	assignCmd.Flags().StringVar(&assignUser, "user", "", "assign to this user (email or username)")
	assignCmd.Flags().StringVar(&assignTeam, "team", "", "set the Team field to this team")
	assignCmd.Flags().BoolVar(&assignDryRun, "dry-run", false, "print the request instead of sending it")
	rootCmd.AddCommand(assignCmd)
}
