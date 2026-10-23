package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
	"jira-cli/wiki2markdown2wiki"
)

var editDryRun bool

var editCmd = &cobra.Command{
	Use:   "edit <key|url>",
	Short: "Edit an issue in your editor",
	Long: `Edit an issue as a Markdown file in $VISUAL or $EDITOR (default vi). The
fields are YAML front matter and the description is the body. The fields
shown are the summary, issue type and status, plus the project's
edit_fields in ~/.jira.yaml; any other editable field can be added by name.

Only changed fields are updated. Removing a field leaves it unchanged; leaving
it empty clears it. Changing the status moves the issue through the matching
workflow transition.`,
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
		meta, err := c.EditMeta(ctx, key)
		if err != nil {
			return err
		}
		// The description is the body; status is changed by transition.
		delete(meta, "description")
		meta["status"] = jira.FieldMeta{ID: "status", Name: "Status"}

		p := issueProject(cfg, key)
		fields, skipped := selectFields(meta, append([]string{"summary", "issuetype", "status"}, p.EditFields...), "status")
		ids := []string{"description"}
		for _, f := range fields {
			ids = append(ids, f.ID)
		}
		issue, err := c.GetIssue(ctx, key, ids)
		if err != nil {
			return err
		}
		key = issue.Key
		transitions, err := c.Transitions(ctx, key)
		if err != nil {
			return err
		}

		header := []string{
			"Editing " + key + ". Fields you remove are left unchanged; empty fields",
			"are cleared. The description is the text below the front matter.",
		}
		if len(skipped) > 0 {
			header = append(header, "Can't edit: "+strings.Join(skipped, ", "))
		}
		var docFields []docField
		for _, f := range fields {
			df := docField{Key: docKey(meta, f), Value: f.Value(issue.Fields[f.ID]), Comment: fieldComment(f)}
			if f.ID == "status" {
				df.Value = fieldText(issue.Fields["status"])
				df.Comment = "can move to: " + strings.Join(transitionTargets(transitions), ", ")
			}
			docFields = append(docFields, df)
		}
		var description string
		_ = json.Unmarshal(issue.Fields["description"], &description)
		text, err := renderDocument(header, docFields, wikiToMarkdown(description))
		if err != nil {
			return err
		}
		original, err := parseDocument(text)
		if err != nil {
			return err
		}
		originalValues, err := fieldValues(original, meta)
		if err != nil {
			return err
		}

		err = editLoop(cmd, text, func(text string) error {
			doc, err := parseDocument(text)
			if err != nil {
				return err
			}
			values, err := fieldValues(doc, meta)
			if err != nil {
				return err
			}
			update := map[string]any{}
			var transition *jira.Transition
			for id, v := range values {
				if old, ok := originalValues[id]; ok && (reflect.DeepEqual(old, v) || jira.IsEmpty(old) && jira.IsEmpty(v)) {
					continue
				}
				if id == "status" {
					if jira.IsEmpty(v) {
						return fmt.Errorf("status can't be empty")
					}
					if transition, err = findTransition(transitions, fmt.Sprint(v)); err != nil {
						return err
					}
					continue
				}
				if update[id], err = c.FieldPayload(ctx, meta[id], v); err != nil {
					return err
				}
			}
			if strings.TrimSpace(doc.Body) != strings.TrimSpace(original.Body) {
				update["description"] = wiki2markdown2wiki.MarkdownToWiki(strings.TrimSpace(doc.Body))
			}
			if len(update) == 0 && transition == nil {
				return errNoChanges
			}

			out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			if editDryRun {
				return printDryRun(out, "PUT "+key, map[string]any{"fields": update}, transition)
			}
			if len(update) > 0 {
				if err := c.UpdateIssue(ctx, key, update); err != nil {
					return err
				}
			}
			if transition != nil {
				if err := c.DoTransition(ctx, key, transition.ID); err != nil {
					return fmt.Errorf("fields updated, but moving to %s failed: %w", transition.To.Name, err)
				}
			}
			theme := newTheme(out.Profile)
			fmt.Fprintln(out, "Updated", theme.IssueKey.Hyperlink(c.BrowseURL(key)).Render(key), c.BrowseURL(key))
			return nil
		})
		return err
	},
}

// issueProject returns the config for the project an issue key belongs to,
// or an empty config if the project isn't configured.
func issueProject(cfg *jira.Config, key string) *jira.ProjectConfig {
	project, _, _ := strings.Cut(key, "-")
	if p, err := cfg.Project(project); err == nil {
		return p
	}
	return &jira.ProjectConfig{}
}

// transitionTargets lists the statuses an issue can move to.
func transitionTargets(transitions []jira.Transition) []string {
	var names []string
	for _, t := range transitions {
		names = append(names, t.To.Name)
	}
	return names
}

// findTransition finds the transition to a status, matching the target
// status name or, failing that, the transition's own name.
func findTransition(transitions []jira.Transition, status string) (*jira.Transition, error) {
	status = strings.TrimSpace(status)
	for i, t := range transitions {
		if strings.EqualFold(t.To.Name, status) {
			return &transitions[i], nil
		}
	}
	for i, t := range transitions {
		if strings.EqualFold(t.Name, status) {
			return &transitions[i], nil
		}
	}
	return nil, fmt.Errorf("can't move to status %q; can move to: %s", status, strings.Join(transitionTargets(transitions), ", "))
}

// printDryRun prints the request that would be sent, and the transition that
// would follow it, if any.
func printDryRun(w io.Writer, request string, body any, transition *jira.Transition) error {
	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Dry run, would send:\n%s\n%s\n", request, b)
	if transition != nil {
		fmt.Fprintf(w, "then transition %q (id %s) to %s\n", transition.Name, transition.ID, transition.To.Name)
	}
	return nil
}

func init() {
	editCmd.Flags().BoolVar(&editDryRun, "dry-run", false, "print the changes instead of making them")
	rootCmd.AddCommand(editCmd)
}
