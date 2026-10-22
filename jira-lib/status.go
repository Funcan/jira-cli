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

// StatusGroup is a logical grouping of statuses, in the order they should be
// shown.
type StatusGroup int

const (
	StatusToDo StatusGroup = iota
	StatusInProgress
	StatusBlocked
	StatusDone
)

// StatusRank returns the logical group of a status: not started, in
// progress, blocked, or done. Lower ranks come first.
//
// JIRA has no "blocked" category - blocked statuses are filed under To Do or
// In Progress depending on the workflow - so they are recognised by name.
func StatusRank(s Status) StatusGroup {
	name := strings.ToLower(s.Name)
	if strings.Contains(name, "block") || strings.Contains(name, "on hold") {
		return StatusBlocked
	}
	switch s.StatusCategory.Key {
	case "new":
		return StatusToDo
	case "done":
		return StatusDone
	default:
		return StatusInProgress
	}
}

// SortStatuses sorts statuses by StatusRank, then by name within a rank.
func SortStatuses(statuses []Status) {
	slices.SortStableFunc(statuses, compareStatuses)
}

func compareStatuses(a, b Status) int {
	if r := int(StatusRank(a)) - int(StatusRank(b)); r != 0 {
		return r
	}
	return strings.Compare(a.Name, b.Name)
}
