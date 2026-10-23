package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func jsonBody(v any) (*bytes.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// UpdateIssue sets fields on an issue. Values are in the REST API's format
// (see Client.FieldPayload).
func (c *Client) UpdateIssue(ctx context.Context, key string, fields map[string]any) error {
	body, err := jsonBody(map[string]any{"fields": fields})
	if err != nil {
		return err
	}
	return c.Do(ctx, http.MethodPut, "rest/api/2/issue/"+url.PathEscape(key), nil, body, nil)
}

// CreateIssue creates an issue and returns its key. Fields must include the
// project and issue type.
func (c *Client) CreateIssue(ctx context.Context, fields map[string]any) (string, error) {
	body, err := jsonBody(map[string]any{"fields": fields})
	if err != nil {
		return "", err
	}
	var created struct {
		Key string `json:"key"`
	}
	if err := c.Do(ctx, http.MethodPost, "rest/api/2/issue", nil, body, &created); err != nil {
		return "", err
	}
	return created.Key, nil
}

// Transition is a workflow transition available on an issue.
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

// Transitions returns the workflow transitions currently available on an
// issue.
func (c *Client) Transitions(ctx context.Context, key string) ([]Transition, error) {
	var resp struct {
		Transitions []Transition `json:"transitions"`
	}
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/issue/"+url.PathEscape(key)+"/transitions", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Transitions, nil
}

// DoTransition moves an issue through a workflow transition.
func (c *Client) DoTransition(ctx context.Context, key, transitionID string) error {
	body, err := jsonBody(map[string]any{"transition": map[string]string{"id": transitionID}})
	if err != nil {
		return err
	}
	return c.Do(ctx, http.MethodPost, "rest/api/2/issue/"+url.PathEscape(key)+"/transitions", nil, body, nil)
}

// FindUser finds the single user matching query (an email address, username
// or name). If the search finds several users, one whose email, username,
// account ID or display name exactly matches query is chosen.
func (c *Client) FindUser(ctx context.Context, query string) (*User, error) {
	param := "username"
	if c.isCloud() {
		param = "query"
	}
	var users []User
	if err := c.Do(ctx, http.MethodGet, "rest/api/2/user/search", url.Values{param: {query}}, nil, &users); err != nil {
		return nil, err
	}
	if len(users) == 1 {
		return &users[0], nil
	}
	var names []string
	for i, u := range users {
		for _, s := range []string{u.EmailAddress, u.Name, u.AccountID, u.DisplayName} {
			if s != "" && strings.EqualFold(s, query) {
				return &users[i], nil
			}
		}
		names = append(names, u.DisplayName+" <"+u.EmailAddress+">")
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no user found matching %q", query)
	}
	return nil, fmt.Errorf("%d users match %q: %s", len(users), query, strings.Join(names, ", "))
}

// UserRef returns the REST API value that refers to a user: by username on
// Data Center, by account ID on Cloud.
func (c *Client) UserRef(u *User) map[string]string {
	if c.isCloud() {
		return map[string]string{"accountId": u.AccountID}
	}
	return map[string]string{"name": u.Name}
}

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

// FindTeam returns the ID of the team with the given name, as the team field
// expects it. Data Center uses Advanced Roadmaps teams; Cloud uses Atlassian
// teams, which are found via JQL autocomplete for the field.
func (c *Client) FindTeam(ctx context.Context, field FieldMeta, name string) (string, error) {
	var ids, names []string
	if c.isCloud() {
		q := url.Values{"fieldName": {"cf[" + strings.TrimPrefix(field.ID, "customfield_") + "]"}, "fieldValue": {name}}
		var resp struct {
			Results []struct {
				DisplayName string `json:"displayName"`
				Value       string `json:"value"`
			} `json:"results"`
		}
		if err := c.Do(ctx, http.MethodGet, "rest/api/2/jql/autocompletedata/suggestions", q, nil, &resp); err != nil {
			return "", err
		}
		for _, r := range resp.Results {
			ids = append(ids, r.Value)
			// The matching part of the name is highlighted with <b> tags.
			names = append(names, htmlTagRE.ReplaceAllString(r.DisplayName, ""))
		}
	} else {
		body, err := jsonBody(map[string]any{"query": name, "maxResults": 50})
		if err != nil {
			return "", err
		}
		var resp struct {
			Teams []struct {
				ID    json.Number `json:"id"`
				Title string      `json:"title"`
			} `json:"teams"`
		}
		if err := c.Do(ctx, http.MethodPost, "rest/teams/1.0/teams/find", nil, body, &resp); err != nil {
			return "", err
		}
		for _, t := range resp.Teams {
			ids = append(ids, t.ID.String())
			names = append(names, t.Title)
		}
	}
	for i, n := range names {
		if strings.EqualFold(n, name) {
			return ids[i], nil
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no team found matching %q", name)
	}
	return "", fmt.Errorf("no team named %q; similar teams: %s", name, strings.Join(names, ", "))
}
