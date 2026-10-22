package jira

import (
	"slices"
	"testing"
)

func TestSortStatuses(t *testing.T) {
	st := func(name, cat string) Status {
		return Status{Name: name, StatusCategory: StatusCategory{Key: cat}}
	}
	statuses := []Status{
		st("Closed", "done"),
		st("Blocked", "new"),
		st("In Review", "indeterminate"),
		st("To Do", "new"),
		st("On Hold", "indeterminate"),
		st("In Progress", "indeterminate"),
		st("Backlog", "new"),
	}
	SortStatuses(statuses)

	var got []string
	for _, s := range statuses {
		got = append(got, s.Name)
	}
	want := []string{"Backlog", "To Do", "In Progress", "In Review", "Blocked", "On Hold", "Closed"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
