package jira

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Issue is a JIRA issue, with the subset of fields this package understands.
type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields are the fields of an Issue. Fields not requested in the search
// are left empty.
type IssueFields struct {
	Summary   string    `json:"summary"`
	Status    Status    `json:"status"`
	IssueType IssueType `json:"issuetype"`
	Priority  *Priority `json:"priority"`
	Project   Project   `json:"project"`
	Created   Time      `json:"created"`
	Updated   Time      `json:"updated"`
}

// Project identifies the project an issue belongs to.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Time is a timestamp in the format JIRA uses in issue fields.
type Time struct {
	time.Time
}

// TimeFormat is the layout of timestamps in JIRA issue fields, e.g.
// "2026-10-08T16:34:30.000+0000".
const TimeFormat = "2006-01-02T15:04:05.000-0700"

// UnmarshalJSON parses a JIRA timestamp.
func (t *Time) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil || s == "" {
		return err
	}
	t.Time, err = time.Parse(TimeFormat, s)
	return err
}

// IssueType is the type of an issue, e.g. Story or Bug.
type IssueType struct {
	Name string `json:"name"`
}

// Priority is the priority of an issue. Some projects don't use priorities,
// in which case the issue's Priority is nil.
type Priority struct {
	Name string `json:"name"`
}

// Priorities returns all priorities defined on the instance, highest first.
func (c *Client) Priorities(ctx context.Context) ([]Priority, error) {
	var ps []Priority
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/priority", nil, nil, &ps); err != nil {
		return nil, err
	}
	return ps, nil
}

// Search returns all issues matching jql, fetching the given fields.
// Results are fetched in pages until exhausted.
func (c *Client) Search(ctx context.Context, jql string, fields []string) ([]Issue, error) {
	const pageSize = 100
	q := url.Values{
		"jql":        {jql},
		"fields":     {strings.Join(fields, ",")},
		"maxResults": {strconv.Itoa(pageSize)},
	}

	var issues []Issue
	// Cloud has removed rest/api/2/search in favour of rest/api/2/search/jql,
	// which pages with a token rather than an offset; Data Center only has
	// the former.
	if c.isCloud() {
		for {
			var page struct {
				Issues        []Issue `json:"issues"`
				NextPageToken string  `json:"nextPageToken"`
				IsLast        bool    `json:"isLast"`
			}
			if err := c.Do(ctx, http.MethodGet, "rest/api/2/search/jql", q, nil, &page); err != nil {
				return nil, err
			}
			issues = append(issues, page.Issues...)
			if page.IsLast || page.NextPageToken == "" {
				return issues, nil
			}
			q.Set("nextPageToken", page.NextPageToken)
		}
	}

	for {
		q.Set("startAt", strconv.Itoa(len(issues)))
		var page struct {
			Issues []Issue `json:"issues"`
			Total  int     `json:"total"`
		}
		if err := c.Do(ctx, http.MethodGet, "rest/api/2/search", q, nil, &page); err != nil {
			return nil, err
		}
		issues = append(issues, page.Issues...)
		if len(page.Issues) == 0 || len(issues) >= page.Total {
			return issues, nil
		}
	}
}
