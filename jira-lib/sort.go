package jira

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Sort fields understood by ParseSort and SortIssues.
const (
	SortOpened   = "opened"
	SortUpdated  = "updated"
	SortStatus   = "status"
	SortPriority = "priority"
	SortType     = "type"
	SortProject  = "project"
)

// naturalDesc is whether each field sorts descending when no direction is
// given: newest dates and highest priorities first, everything else
// ascending.
var naturalDesc = map[string]bool{
	SortOpened:   true,
	SortUpdated:  true,
	SortStatus:   false,
	SortPriority: true,
	SortType:     false,
	SortProject:  false,
}

// SortKey is one field to sort issues by.
type SortKey struct {
	Field string
	Desc  bool
}

// ParseSort parses a comma separated list of sort fields. Each field may be
// prefixed with "+" for ascending or "-" for descending; otherwise it sorts
// in its natural direction (see naturalDesc). "created" and "openned" are
// accepted as aliases of "opened".
func ParseSort(s string) ([]SortKey, error) {
	var keys []SortKey
	for _, f := range strings.Split(s, ",") {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == "" {
			continue
		}
		var dir byte
		if f[0] == '+' || f[0] == '-' {
			dir, f = f[0], f[1:]
		}
		if f == "created" || f == "openned" {
			f = SortOpened
		}
		desc, ok := naturalDesc[f]
		if !ok {
			return nil, fmt.Errorf("unknown sort field %q: must be one of opened, updated, status, priority, type, project", f)
		}
		switch dir {
		case '+':
			desc = false
		case '-':
			desc = true
		}
		keys = append(keys, SortKey{Field: f, Desc: desc})
	}
	return keys, nil
}

// SortIssues sorts issues by keys in order, then by issue key. Ascending
// status order is that of SortStatuses. priorities is the instance's
// priorities, highest first, as returned by Client.Priorities; ascending
// priority order is lowest first, with issues without a priority lowest of
// all. priorities may be nil if not sorting by priority.
func SortIssues(issues []Issue, keys []SortKey, priorities []Priority) {
	priorityRank := map[string]int{}
	for i, p := range priorities {
		priorityRank[p.Name] = len(priorities) - i
	}
	rank := func(p *Priority) int {
		if p == nil {
			return 0
		}
		return priorityRank[p.Name]
	}

	slices.SortStableFunc(issues, func(a, b Issue) int {
		for _, k := range keys {
			var r int
			switch k.Field {
			case SortOpened:
				r = a.Fields.Created.Compare(b.Fields.Created.Time)
			case SortUpdated:
				r = a.Fields.Updated.Compare(b.Fields.Updated.Time)
			case SortStatus:
				r = compareStatuses(a.Fields.Status, b.Fields.Status)
			case SortPriority:
				r = cmp.Compare(rank(a.Fields.Priority), rank(b.Fields.Priority))
			case SortType:
				r = strings.Compare(a.Fields.IssueType.Name, b.Fields.IssueType.Name)
			case SortProject:
				r = strings.Compare(a.Fields.Project.Key, b.Fields.Project.Key)
			}
			if k.Desc {
				r = -r
			}
			if r != 0 {
				return r
			}
		}
		return compareIssueKeys(a.Key, b.Key)
	})
}

// compareIssueKeys orders issue keys like "ABC-9" and "ABC-10" by project,
// then numerically.
func compareIssueKeys(a, b string) int {
	ap, an, _ := strings.Cut(a, "-")
	bp, bn, _ := strings.Cut(b, "-")
	if r := strings.Compare(ap, bp); r != 0 {
		return r
	}
	ai, _ := strconv.Atoi(an)
	bi, _ := strconv.Atoi(bn)
	return cmp.Compare(ai, bi)
}
