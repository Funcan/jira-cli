package cmd

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	jira "jira-cli/jira-lib"
)

// A document is the Markdown file edited by `jira edit` and `jira create`:
// YAML front matter holding the fields, followed by the description.

// docField is a field in the front matter of a document.
type docField struct {
	Key     string
	Value   any
	Comment string
}

// renderDocument renders a document. header is shown as comment lines at the
// top of the front matter.
func renderDocument(header []string, fields []docField, body string) (string, error) {
	m := &yaml.Node{Kind: yaml.MappingNode}
	for _, f := range fields {
		var v yaml.Node
		if jira.IsEmpty(f.Value) && !isList(f.Value) {
			// Show empty fields as "Key:" rather than "Key: null".
			v = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
		} else if err := v.Encode(f.Value); err != nil {
			return "", fmt.Errorf("%s: %w", f.Key, err)
		}
		if v.Kind == yaml.SequenceNode {
			v.Style = yaml.FlowStyle
		}
		if f.Comment != "" {
			v.LineComment = "# " + f.Comment
		}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: f.Key}, &v)
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{m}}
	for _, line := range header {
		doc.HeadComment += "# " + line + "\n"
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	buf.WriteString("---\n")
	buf.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

func isList(v any) bool {
	_, ok := v.([]any)
	return ok
}

// parsedDocument is a document read back from the editor.
type parsedDocument struct {
	// Keys are the front matter keys, in order.
	Keys   []string
	Values map[string]any
	Body   string
}

// parseDocument parses a document. The front matter must be a YAML mapping
// between "---" lines at the start of the file.
func parseDocument(text string) (*parsedDocument, error) {
	rest, ok := strings.CutPrefix(strings.TrimLeft(text, "\n"), "---\n")
	if !ok {
		return nil, fmt.Errorf("the file must start with front matter between --- lines")
	}
	lines := strings.SplitAfter(rest, "\n")
	end := slices.IndexFunc(lines, func(l string) bool { return strings.TrimRight(l, "\r\n") == "---" })
	if end < 0 {
		return nil, fmt.Errorf("no closing --- line after the front matter")
	}
	front := strings.Join(lines[:end], "")

	doc := &parsedDocument{Values: map[string]any{}, Body: strings.Join(lines[end+1:], "")}
	var node yaml.Node
	// Keep the opening --- so line numbers in errors match the file.
	if err := yaml.Unmarshal([]byte("---\n"+front), &node); err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	if len(node.Content) == 0 {
		return doc, nil
	}
	m := node.Content[0]
	if m.Tag == "!!null" {
		return doc, nil
	}
	if m.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("front matter must be a mapping of field names to values")
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		key := m.Content[i].Value
		if _, dup := doc.Values[key]; dup {
			return nil, fmt.Errorf("front matter: line %d: %q is given twice", m.Content[i].Line, key)
		}
		var v any
		if err := m.Content[i+1].Decode(&v); err != nil {
			return nil, fmt.Errorf("front matter: %s: %w", key, err)
		}
		doc.Keys = append(doc.Keys, key)
		doc.Values[key] = v
	}
	return doc, nil
}

// fieldValues maps a document's keys to fields, returning the values keyed by
// field ID.
func fieldValues(doc *parsedDocument, fields map[string]jira.FieldMeta) (map[string]any, error) {
	values := map[string]any{}
	for _, key := range doc.Keys {
		f, err := jira.FindField(fields, key)
		if err != nil {
			return nil, err
		}
		if _, dup := values[f.ID]; dup {
			return nil, fmt.Errorf("%s is given twice", f.Name)
		}
		values[f.ID] = doc.Values[key]
	}
	return values, nil
}

// selectFields looks up the named fields (IDs or display names), in order,
// skipping duplicates. Names that aren't in fields, or whose type can't be
// edited, are returned as skipped. Fields in keep are always kept, even if
// their type isn't supported.
func selectFields(fields map[string]jira.FieldMeta, names []string, keep ...string) (selected []jira.FieldMeta, skipped []string) {
	for _, name := range names {
		f, err := jira.FindField(fields, name)
		if err != nil || (!f.Supported() && !slices.Contains(keep, f.ID)) {
			skipped = append(skipped, name)
			continue
		}
		if !slices.ContainsFunc(selected, func(s jira.FieldMeta) bool { return s.ID == f.ID }) {
			selected = append(selected, f)
		}
	}
	return selected, skipped
}

// docKey is the key for a field in the front matter: its display name,
// unless another field shares the name, in which case its ID.
func docKey(fields map[string]jira.FieldMeta, f jira.FieldMeta) string {
	for id, other := range fields {
		if id != f.ID && strings.EqualFold(other.Name, f.Name) {
			return f.ID
		}
	}
	return f.Name
}

// fieldComment describes a field's constraints: whether it's required, and
// its allowed values if there are only a few.
func fieldComment(f jira.FieldMeta) string {
	var parts []string
	if f.Required {
		parts = append(parts, "required")
	}
	if n := len(f.AllowedValues); n > 0 && n <= 10 {
		var names []string
		for _, a := range f.AllowedValues {
			names = append(names, a.Text())
		}
		parts = append(parts, strings.Join(names, ", "))
	}
	return strings.Join(parts, "; ")
}
