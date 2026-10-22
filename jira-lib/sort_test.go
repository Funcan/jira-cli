package jira

import (
	"slices"
	"testing"
	"time"
)

func TestParseSort(t *testing.T) {
	keys, err := ParseSort("updated, +opened,-type,openned,Status")
	if err != nil {
		t.Fatal(err)
	}
	want := []SortKey{
		{SortUpdated, true},
		{SortOpened, false},
		{SortType, true},
		{SortOpened, true},
		{SortStatus, false},
	}
	if !slices.Equal(keys, want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	if _, err := ParseSort("bogus"); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestSortIssues(t *testing.T) {
	day := func(d int) Time { return Time{time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC)} }
	issue := func(key, status, cat, priority string, created int) Issue {
		i := Issue{Key: key, Fields: IssueFields{
			Status:  Status{Name: status, StatusCategory: StatusCategory{Key: cat}},
			Created: day(created),
		}}
		if priority != "" {
			i.Fields.Priority = &Priority{Name: priority}
		}
		return i
	}
	issues := []Issue{
		issue("X-10", "Done", "done", "High", 1),
		issue("X-9", "To Do", "new", "Low", 2),
		issue("X-2", "In Progress", "indeterminate", "", 3),
		issue("X-1", "To Do", "new", "High", 4),
		issue("X-3", "To Do", "new", "High", 5),
	}
	priorities := []Priority{{Name: "High"}, {Name: "Medium"}, {Name: "Low"}}
	keys := func(issues []Issue) []string {
		var k []string
		for _, i := range issues {
			k = append(k, i.Key)
		}
		return k
	}

	for _, tc := range []struct {
		sort string
		want []string
	}{
		{"status,priority", []string{"X-1", "X-3", "X-9", "X-2", "X-10"}},
		{"-status", []string{"X-10", "X-2", "X-1", "X-3", "X-9"}},
		{"priority", []string{"X-1", "X-3", "X-10", "X-9", "X-2"}},
		{"+priority,opened", []string{"X-2", "X-9", "X-3", "X-1", "X-10"}},
		{"opened", []string{"X-3", "X-1", "X-2", "X-9", "X-10"}},
		{"", []string{"X-1", "X-2", "X-3", "X-9", "X-10"}},
	} {
		k, err := ParseSort(tc.sort)
		if err != nil {
			t.Fatal(err)
		}
		s := slices.Clone(issues)
		SortIssues(s, k, priorities)
		if got := keys(s); !slices.Equal(got, tc.want) {
			t.Errorf("sort %q: got %v, want %v", tc.sort, got, tc.want)
		}
	}
}
