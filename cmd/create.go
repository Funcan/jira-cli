package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"

	jira "jira-cli/jira-lib"
	"jira-cli/wiki2markdown2wiki"
)

var (
	createType   string
	createDryRun bool
)

var createCmd = &cobra.Command{
	Use:   "create [project]",
	Short: "Create an issue in your editor",
	Long: `Create an issue by filling in a Markdown file in $VISUAL or $EDITOR (default
vi). The fields are YAML front matter and the description is the body. The
project defaults to the default project in ~/.jira.yaml.

The template has the summary, issue type, the project's create_fields, any
fields in its create_defaults (which pre-fill their values) and any other
required fields. Any other field can be added by name. The issue type is
--type, else the issuetype in create_defaults, else Story.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		project := ""
		if len(args) > 0 {
			project = strings.ToUpper(args[0])
		}
		c, p, err := jira.NewClientForProject(project)
		if err != nil {
			return err
		}
		types, err := c.IssueTypes(ctx, p.Project)
		if err != nil {
			return err
		}

		typeName := createType
		defaults := map[string]any{}
		for k, v := range p.CreateDefaults {
			if strings.EqualFold(k, "issuetype") || strings.EqualFold(k, "issue type") {
				if typeName == "" {
					typeName = fmt.Sprint(v)
				}
				continue
			}
			defaults[k] = v
		}
		if typeName == "" {
			typeName = "Story"
		}
		issueType, err := findIssueType(types, typeName)
		if err != nil {
			return err
		}
		metas := map[string]map[string]jira.FieldMeta{}
		descriptionRequired := map[string]bool{}
		createMeta := func(t jira.IssueTypeMeta) (map[string]jira.FieldMeta, error) {
			if m, ok := metas[t.ID]; ok {
				return m, nil
			}
			m, err := c.CreateMeta(ctx, p.Project, t.ID)
			if err != nil {
				return nil, err
			}
			// Project and issue type are set separately, and the
			// description is the body.
			descriptionRequired[t.ID] = m["description"].Required
			delete(m, "project")
			delete(m, "description")
			m["issuetype"] = jira.FieldMeta{ID: "issuetype", Name: "Issue Type"}
			metas[t.ID] = m
			return m, nil
		}
		meta, err := createMeta(issueType)
		if err != nil {
			return err
		}

		defaultKeys := sortedKeys(defaults)
		names := append([]string{"summary", "issuetype"}, p.CreateFields...)
		names = append(names, defaultKeys...)
		for _, f := range sortedFields(meta) {
			if f.Required && !f.HasDefaultValue {
				names = append(names, f.ID)
			}
		}
		fields, skipped := selectFields(meta, names, "issuetype")
		values := map[string]any{"issuetype": issueType.Name}
		for _, k := range defaultKeys {
			f, err := jira.FindField(meta, k)
			if err != nil {
				return fmt.Errorf("create_defaults for %s: %w", p.Project, err)
			}
			values[f.ID] = defaults[k]
		}
		// Data Center requires a reporter, without defaulting it to you.
		if f, ok := meta["reporter"]; ok && f.Required && values["reporter"] == nil {
			me, err := c.Myself(ctx)
			if err != nil {
				return err
			}
			values["reporter"] = me.Name
			if me.EmailAddress != "" {
				values["reporter"] = me.EmailAddress
			}
		}

		header := []string{
			"Creating a " + p.Project + " issue. Empty fields are left unset. The",
			"description is the text below the front matter.",
		}
		if len(skipped) > 0 {
			header = append(header, "Can't set: "+strings.Join(skipped, ", "))
		}
		var docFields []docField
		for _, f := range fields {
			df := docField{Key: docKey(meta, f), Value: values[f.ID], Comment: fieldComment(f)}
			if df.Value == nil && f.IsArray() {
				df.Value = []any{}
			}
			if f.ID == "issuetype" {
				var names []string
				for _, t := range types {
					names = append(names, t.Name)
				}
				df.Comment = strings.Join(names, ", ")
			}
			docFields = append(docFields, df)
		}
		text, err := renderDocument(header, docFields, "")
		if err != nil {
			return err
		}

		return editLoop(cmd, text, func(text string) error {
			doc, err := parseDocument(text)
			if err != nil {
				return err
			}
			// The issue type decides which fields there are, so find it first.
			issueType := issueType
			for _, k := range doc.Keys {
				if strings.EqualFold(k, "issuetype") || strings.EqualFold(k, "issue type") {
					if issueType, err = findIssueType(types, fmt.Sprint(doc.Values[k])); err != nil {
						return err
					}
				}
			}
			meta, err := createMeta(issueType)
			if err != nil {
				return err
			}
			values, err := fieldValues(doc, meta)
			if err != nil {
				return err
			}

			create := map[string]any{
				"project":   map[string]string{"key": p.Project},
				"issuetype": map[string]string{"id": issueType.ID},
			}
			for id, v := range values {
				if id == "issuetype" || jira.IsEmpty(v) {
					continue
				}
				if id == "status" {
					return fmt.Errorf("the status can't be set when creating an issue")
				}
				if create[id], err = c.FieldPayload(ctx, meta[id], v); err != nil {
					return err
				}
			}
			if body := strings.TrimSpace(doc.Body); body != "" {
				create["description"] = wiki2markdown2wiki.MarkdownToWiki(body)
			} else if descriptionRequired[issueType.ID] {
				return fmt.Errorf("a description is required")
			}
			for _, f := range sortedFields(meta) {
				if _, ok := create[f.ID]; ok || !f.Required || f.HasDefaultValue || f.ID == "issuetype" {
					continue
				}
				msg := f.Name + " is required"
				if len(f.AllowedValues) > 0 {
					var choices []string
					for _, a := range f.AllowedValues {
						choices = append(choices, a.Text())
					}
					msg += "; one of: " + strings.Join(choices, ", ")
				}
				return fmt.Errorf("%s", msg)
			}

			out := colorprofile.NewWriter(cmd.OutOrStdout(), os.Environ())
			if createDryRun {
				return printDryRun(out, "POST "+p.Project, map[string]any{"fields": create}, nil)
			}
			key, err := c.CreateIssue(ctx, create)
			if err != nil {
				return err
			}
			theme := newTheme(out.Profile)
			fmt.Fprintln(out, "Created", theme.IssueKey.Hyperlink(c.BrowseURL(key)).Render(key), c.BrowseURL(key))
			return nil
		})
	},
}

func findIssueType(types []jira.IssueTypeMeta, name string) (jira.IssueTypeMeta, error) {
	var names []string
	for _, t := range types {
		if strings.EqualFold(t.Name, strings.TrimSpace(name)) {
			return t, nil
		}
		names = append(names, t.Name)
	}
	return jira.IssueTypeMeta{}, fmt.Errorf("unknown issue type %q; one of: %s", name, strings.Join(names, ", "))
}

func sortedKeys(m map[string]any) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// sortedFields returns fields in a stable order, by ID.
func sortedFields(fields map[string]jira.FieldMeta) []jira.FieldMeta {
	var list []jira.FieldMeta
	for _, f := range fields {
		list = append(list, f)
	}
	slices.SortFunc(list, func(a, b jira.FieldMeta) int { return strings.Compare(a.ID, b.ID) })
	return list
}

func init() {
	createCmd.Flags().StringVar(&createType, "type", "", "issue type (default from create_defaults, else Story)")
	createCmd.Flags().BoolVar(&createDryRun, "dry-run", false, "print the issue instead of creating it")
	rootCmd.AddCommand(createCmd)
}
