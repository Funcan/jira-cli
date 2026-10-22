package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// RawIssue is an issue with its fields left as raw JSON, for displaying
// arbitrary (including custom) fields.
type RawIssue struct {
	Key    string                     `json:"key"`
	Fields map[string]json.RawMessage `json:"fields"`
	// Names maps field IDs to their display names on this instance.
	Names map[string]string `json:"names"`
}

// GetIssue fetches a single issue with the given fields.
func (c *Client) GetIssue(ctx context.Context, key string, fields []string) (*RawIssue, error) {
	q := url.Values{"fields": {strings.Join(fields, ",")}, "expand": {"names"}}
	var issue RawIssue
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/issue/"+url.PathEscape(key), q, nil, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// Comment is a comment on an issue. Body is JIRA wiki markup.
type Comment struct {
	Author  User   `json:"author"`
	Body    string `json:"body"`
	Created Time   `json:"created"`
	Updated Time   `json:"updated"`
}

// Comments returns all comments on an issue, oldest first.
func (c *Client) Comments(ctx context.Context, key string) ([]Comment, error) {
	var comments []Comment
	for {
		q := url.Values{"startAt": {strconv.Itoa(len(comments))}, "maxResults": {"100"}}
		var page struct {
			Comments []Comment `json:"comments"`
			Total    int       `json:"total"`
		}
		if err := c.Do(ctx, http.MethodGet, "rest/api/2/issue/"+url.PathEscape(key)+"/comment", q, nil, &page); err != nil {
			return nil, err
		}
		comments = append(comments, page.Comments...)
		if len(page.Comments) == 0 || len(comments) >= page.Total {
			break
		}
	}
	slices.SortStableFunc(comments, func(a, b Comment) int { return a.Created.Compare(b.Created.Time) })
	return comments, nil
}

var issueKeyRE = regexp.MustCompile(`^([A-Z][A-Z0-9_]*)-[0-9]+$`)

// ResolveIssue works out the instance and issue key for ref, which is either
// an issue key (e.g. "ABC-123", case-insensitive) or an issue URL on a
// configured instance (e.g. "https://host/browse/ABC-123").
//
// A key's project is looked up in the config; keys for unconfigured projects
// are assumed to be on the default project's instance.
func (c *Config) ResolveIssue(ref string) (*InstanceConfig, string, error) {
	if u, err := url.Parse(ref); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return c.resolveIssueURL(u)
	}

	key := strings.ToUpper(ref)
	m := issueKeyRE.FindStringSubmatch(key)
	if m == nil {
		return nil, "", fmt.Errorf("%q is not an issue key or URL", ref)
	}
	p, err := c.Project(m[1])
	if err != nil {
		if p, err = c.Project(""); err != nil {
			return nil, "", fmt.Errorf("project %s not in config and no default project: %w", m[1], err)
		}
	}
	inst, err := c.Instance(p.Instance)
	if err != nil {
		return nil, "", err
	}
	return inst, key, nil
}

func (c *Config) resolveIssueURL(u *url.URL) (*InstanceConfig, string, error) {
	for i := range c.Instances {
		inst := &c.Instances[i]
		base, err := url.Parse(inst.URL)
		if err != nil || !strings.EqualFold(base.Hostname(), u.Hostname()) {
			continue
		}
		rest, ok := strings.CutPrefix(strings.Trim(u.Path, "/"), strings.Trim(base.Path, "/"))
		if !ok {
			continue
		}
		// Boards link to issues with ?selectedIssue=KEY rather than /browse/KEY.
		key := u.Query().Get("selectedIssue")
		if k, ok := strings.CutPrefix(strings.Trim(rest, "/"), "browse/"); ok {
			key = k
		}
		key = strings.ToUpper(key)
		if !issueKeyRE.MatchString(key) {
			return nil, "", fmt.Errorf("no issue key found in URL %s", u)
		}
		return inst, key, nil
	}
	return nil, "", fmt.Errorf("URL %s doesn't match any configured instance", u)
}
