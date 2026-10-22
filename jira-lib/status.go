package jira

import (
	"slices"
	"strings"
)

// Status is the workflow status of an issue.
type Status struct {
	Name           string         `json:"name"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

// StatusCategory is the broad category JIRA groups statuses into. Key is one
// of "new" (To Do), "indeterminate" (In Progress) or "done".
type StatusCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// StatusRank orders statuses logically: not started, in progress, blocked,
// then done. Lower ranks come first.
//
// JIRA has no "blocked" category - blocked statuses are filed under To Do or
// In Progress depending on the workflow - so they are recognised by name.
func StatusRank(s Status) int {
	name := strings.ToLower(s.Name)
	if strings.Contains(name, "block") || strings.Contains(name, "on hold") {
		return 2
	}
	switch s.StatusCategory.Key {
	case "new":
		return 0
	case "done":
		return 3
	default:
		return 1
	}
}

// SortStatuses sorts statuses by StatusRank, then by name within a rank.
func SortStatuses(statuses []Status) {
	slices.SortStableFunc(statuses, func(a, b Status) int {
		if r := StatusRank(a) - StatusRank(b); r != 0 {
			return r
		}
		return strings.Compare(a.Name, b.Name)
	})
}
