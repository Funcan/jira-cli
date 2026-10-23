package cmd

import (
	"reflect"
	"strings"
	"testing"

	jira "github.com/Funcan/jira-cli/jira-lib"
)

func TestDocumentRoundTrip(t *testing.T) {
	text, err := renderDocument([]string{"Header line"}, []docField{
		{Key: "Summary", Value: "Fix: the thing", Comment: "required"},
		{Key: "Assignee", Value: nil},
		{Key: "Labels", Value: []any{"a", "b"}},
		{Key: "Components", Value: []any{}},
		{Key: "Story Points", Value: 3},
		{Key: "Due Date", Value: "2026-11-01"},
	}, "Body *text*\n\n---\nmore")
	if err != nil {
		t.Fatal(err)
	}
	want := `---
# Header line

Summary: 'Fix: the thing' # required
Assignee:
Labels: [a, b]
Components: []
Story Points: 3
Due Date: "2026-11-01"
---
Body *text*

---
more
`
	if text != want {
		t.Errorf("renderDocument =\n%s\nwant\n%s", text, want)
	}

	doc, err := parseDocument(text)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc.Keys, []string{"Summary", "Assignee", "Labels", "Components", "Story Points", "Due Date"}) {
		t.Errorf("keys = %q", doc.Keys)
	}
	wantValues := map[string]any{
		"Summary": "Fix: the thing", "Assignee": nil, "Labels": []any{"a", "b"},
		"Components": []any{}, "Story Points": 3, "Due Date": "2026-11-01",
	}
	if !reflect.DeepEqual(doc.Values, wantValues) {
		t.Errorf("values = %#v", doc.Values)
	}
	// A "---" in the body isn't the end of the front matter.
	if doc.Body != "Body *text*\n\n---\nmore\n" {
		t.Errorf("body = %q", doc.Body)
	}
}

func TestParseDocumentErrors(t *testing.T) {
	for text, want := range map[string]string{
		"Summary: x\n":                   "must start with front matter",
		"---\nSummary: x\n":              "no closing ---",
		"---\n- a\n---\n":                "must be a mapping",
		"---\nA: 1\nA: 2\n---\n":         "line 3",
		"---\nSummary: [unclosed\n---\n": "front matter",
	} {
		if _, err := parseDocument(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseDocument(%q) error = %v, want %q", text, err, want)
		}
	}
	doc, err := parseDocument("---\n---\nbody\n")
	if err != nil || len(doc.Keys) != 0 || doc.Body != "body\n" {
		t.Errorf("empty front matter: %+v, %v", doc, err)
	}
}

func TestFieldValues(t *testing.T) {
	fields := map[string]jira.FieldMeta{
		"summary":       {ID: "summary", Name: "Summary"},
		"customfield_1": {ID: "customfield_1", Name: "Story Points"},
	}
	doc := &parsedDocument{Keys: []string{"summary", "Story Points"}, Values: map[string]any{"summary": "s", "Story Points": 2}}
	got, err := fieldValues(doc, fields)
	if err != nil || !reflect.DeepEqual(got, map[string]any{"summary": "s", "customfield_1": 2}) {
		t.Errorf("fieldValues = %v, %v", got, err)
	}
	doc = &parsedDocument{Keys: []string{"Summary", "summary"}, Values: map[string]any{"Summary": "a", "summary": "b"}}
	if _, err := fieldValues(doc, fields); err == nil {
		t.Error("fieldValues accepted the same field twice")
	}
}

func TestFindTransition(t *testing.T) {
	transitions := []jira.Transition{
		{ID: "1", Name: "Start", To: jira.Status{Name: "In Progress"}},
		{ID: "2", Name: "Reopen", To: jira.Status{Name: "Reopened"}},
	}
	for status, want := range map[string]string{"in progress": "1", "Reopened": "2", "reopen": "2", "start": "1"} {
		if tr, err := findTransition(transitions, status); err != nil || tr.ID != want {
			t.Errorf("findTransition(%q) = %v, %v, want %s", status, tr, err, want)
		}
	}
	if _, err := findTransition(transitions, "Done"); err == nil || !strings.Contains(err.Error(), "In Progress, Reopened") {
		t.Errorf("findTransition(Done) error = %v", err)
	}
}
