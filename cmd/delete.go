package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "github.com/Funcan/jira-cli/jira-lib"
)

var (
	deleteYes      bool
	deleteSubtasks bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <key|url>",
	Short: "Permanently delete an issue",
	Long: `Permanently delete an issue. This can't be undone. Asks for confirmation
unless --yes is given. An issue with subtasks is only deleted with
--subtasks, which deletes the subtasks too.`,
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
		issue, err := c.GetIssue(ctx, key, []string{"summary", "subtasks"})
		if err != nil {
			return err
		}
		key = issue.Key
		var summary string
		_ = json.Unmarshal(issue.Fields["summary"], &summary)
		var subtasks []json.RawMessage
		_ = json.Unmarshal(issue.Fields["subtasks"], &subtasks)
		if len(subtasks) > 0 && !deleteSubtasks {
			return fmt.Errorf("%s has %d subtasks; use --subtasks to delete them too", key, len(subtasks))
		}

		out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
		theme := newTheme(out.Profile)
		if !deleteYes {
			prompt := fmt.Sprintf("Permanently delete %s %q", key, summary)
			if len(subtasks) > 0 {
				prompt += fmt.Sprintf(" and its %d subtasks", len(subtasks))
			}
			fmt.Fprint(cmd.ErrOrStderr(), prompt+"? [y/N] ")
			answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr())
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "y" && answer != "yes" {
				return fmt.Errorf("not deleted")
			}
		}
		if err := c.DeleteIssue(ctx, key, deleteSubtasks); err != nil {
			return err
		}
		fmt.Fprintln(out, "Deleted", theme.IssueKey.Render(key))
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "don't ask for confirmation")
	deleteCmd.Flags().BoolVar(&deleteSubtasks, "subtasks", false, "also delete the issue's subtasks")
	rootCmd.AddCommand(deleteCmd)
}
